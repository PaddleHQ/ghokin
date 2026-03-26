package ghokin

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/cucumber/gherkin/go/v28"
)

// CmdError is thrown when an error occurred when calling
// a command on an input, both stdout and stderr are stored.
type CmdError struct {
	output string
}

// Error outputs both stdout and stderr.
func (e CmdError) Error() string {
	return e.output
}

var errParse = fmt.Errorf("failed to parse gherkin")

func extractSections(content []byte) (*section, error) {
	section := &section{}
	builder := &tokenGenerator{section: section}
	matcher := gherkin.NewMatcher(gherkin.DialectsBuiltin())
	scanner := gherkin.NewScanner(bytes.NewBuffer(content))
	parser := gherkin.NewParser(builder)
	parser.StopAtFirstError(true)

	if err := parser.Parse(scanner, matcher); err != nil {
		return section, fmt.Errorf("%w: %w", errParse, err)
	}

	return section, nil
}

func shouldResolveAccumulator(sec *section, accumulator []*gherkin.Token) bool {
	return len(accumulator) > 0 &&
		sec.kind == gherkin.TokenTypeTableRow &&
		(sec.nex != nil && sec.nex.kind != gherkin.TokenTypeComment) || sec.nex == nil
}

func shouldContinueAccumulating(sec *section, accumulator []*gherkin.Token) bool {
	return sec.kind == gherkin.TokenTypeTableRow &&
		sec.nex != nil &&
		sec.nex.kind == gherkin.TokenTypeComment &&
		sec.nex.nex != nil &&
		sec.nex.nex.kind == gherkin.TokenTypeTableRow ||
		len(accumulator) > 0 && sec.kind == gherkin.TokenTypeComment ||
		len(accumulator) > 0 && sec.kind == gherkin.TokenTypeTableRow
}

func processAccumulator(
	sec *section,
	accumulator []*gherkin.Token,
) (values []*gherkin.Token, newAccumulator []*gherkin.Token, skip bool) {
	values = sec.values
	newAccumulator = accumulator

	if shouldResolveAccumulator(sec, accumulator) {
		combined := make([]*gherkin.Token, 0, len(accumulator)+len(sec.values))
		combined = append(combined, accumulator...)
		combined = append(combined, sec.values...)
		values = combined
		newAccumulator = []*gherkin.Token{}
	}

	if shouldContinueAccumulating(sec, newAccumulator) {
		acc := make([]*gherkin.Token, 0, len(newAccumulator)+len(sec.values))
		acc = append(acc, newAccumulator...)
		acc = append(acc, sec.values...)
		newAccumulator = acc
		skip = true
	}

	return values, newAccumulator, skip
}

func isJSONDocString(sec *section) bool {
	return sec.kind == gherkin.TokenTypeDocStringSeparator &&
		len(sec.values) == 1 && sec.values[0].Text == "json"
}

// templateVarRegexp matches {{ ... }} template variables used in BDD feature files,
// optionally consuming surrounding double-quotes to avoid producing invalid
// double-quoted strings (e.g. `"{{ name }}"` → `"__GHOKIN_TPL_0__"`).
var templateVarRegexp = regexp.MustCompile(`"?\{\{[^}]*\}\}"?`)

type templatePlaceholder struct {
	original string
	quoted   bool
}

// replaceTemplateVars replaces {{ ... }} template variables with JSON-safe
// placeholder strings so that json.Indent can parse the content. It returns the
// modified string and an ordered slice of the original template expressions.
func replaceTemplateVars(s string) (string, []templatePlaceholder) {
	var placeholders []templatePlaceholder
	replaced := templateVarRegexp.ReplaceAllStringFunc(s, func(match string) string {
		idx := len(placeholders)
		quoted := strings.HasPrefix(match, `"`) && strings.HasSuffix(match, `"`)
		original := match
		if quoted {
			original = match[1 : len(match)-1]
		}
		placeholders = append(placeholders, templatePlaceholder{original: original, quoted: quoted})
		return fmt.Sprintf(`"__GHOKIN_TPL_%d__"`, idx)
	})
	return replaced, placeholders
}

// restoreTemplateVars reverses the placeholder substitution performed by
// replaceTemplateVars, putting the original {{ ... }} expressions back.
func restoreTemplateVars(s string, placeholders []templatePlaceholder) string {
	for i, p := range placeholders {
		placeholder := fmt.Sprintf(`"__GHOKIN_TPL_%d__"`, i)
		var replacement string
		if p.quoted {
			replacement = fmt.Sprintf("%q", p.original)
		} else {
			replacement = p.original
		}
		s = strings.Replace(s, placeholder, replacement, 1)
	}
	return s
}

func formatJSONDocString(
	sec *section,
	paddings map[gherkin.TokenType]int,
) (*section, []string, error) {
	document := make([]string, 0)
	document = append(
		document,
		trimExtraTrailingSpace(indentStrings(
			paddings[gherkin.TokenTypeOther], []string{`"""json`},
		))...,
	)

	var jsonLines strings.Builder
	for sec.nex.kind != gherkin.TokenTypeDocStringSeparator {
		sec = sec.nex
		for _, value := range sec.values {
			jsonLines.WriteString(value.Text)
		}
	}

	rawJSON := jsonLines.String()
	sanitized, placeholders := replaceTemplateVars(rawJSON)

	var prettyJSON bytes.Buffer
	if err := json.Indent(&prettyJSON, []byte(sanitized), "", "  "); err != nil {
		return sec, nil, fmt.Errorf("failed to format json: %w", err)
	}

	restored := restoreTemplateVars(prettyJSON.String(), placeholders)
	jsonFormatted := strings.Split(restored, "\n")
	document = append(
		document,
		trimExtraTrailingSpace(indentStrings(
			paddings[gherkin.TokenTypeOther], jsonFormatted,
		))...,
	)

	return sec, document, nil
}

func buildPaddings(indent int) map[gherkin.TokenType]int {
	return map[gherkin.TokenType]int{
		gherkin.TokenTypeFeatureLine:        0,
		gherkin.TokenTypeBackgroundLine:     indent,
		gherkin.TokenTypeScenarioLine:       indent,
		gherkin.TokenTypeDocStringSeparator: 3 * indent,
		gherkin.TokenTypeStepLine:           2 * indent,
		gherkin.TokenTypeExamplesLine:       2 * indent,
		gherkin.TokenTypeOther:              3 * indent,
		gherkin.TokenTypeTableRow:           3 * indent,
	}
}

var formats = map[gherkin.TokenType]func(values []*gherkin.Token) []string{
	gherkin.TokenTypeFeatureLine:        extractKeywordAndTextSeparatedWithAColon,
	gherkin.TokenTypeBackgroundLine:     extractKeywordAndTextSeparatedWithAColon,
	gherkin.TokenTypeScenarioLine:       extractKeywordAndTextSeparatedWithAColon,
	gherkin.TokenTypeExamplesLine:       extractKeywordAndTextSeparatedWithAColon,
	gherkin.TokenTypeComment:            extractTokensText,
	gherkin.TokenTypeTagLine:            extractTokensItemsText,
	gherkin.TokenTypeDocStringSeparator: extractKeyword,
	gherkin.TokenTypeRuleLine:           extractKeywordAndTextSeparatedWithAColon,
	gherkin.TokenTypeOther:              extractTokensText,
	gherkin.TokenTypeStepLine:           extractTokensKeywordAndText,
	gherkin.TokenTypeTableRow:           extractTableRowsAndComments,
	gherkin.TokenTypeEmpty:              extractTokensItemsText,
	gherkin.TokenTypeLanguage:           extractLanguage,
}

func transform(ctx context.Context, sec *section, indent int, aliases aliases) ([]byte, error) {
	paddings := buildPaddings(indent)

	var cmd *exec.Cmd
	document := []string{}
	optionalRulePadding := 0
	accumulator := []*gherkin.Token{}

	for ; sec != nil; sec = sec.nex {
		values, newAcc, skip := processAccumulator(sec, accumulator)
		accumulator = newAcc
		if skip {
			continue
		}

		if isJSONDocString(sec) {
			newSec, doc, err := formatJSONDocString(sec, paddings)
			if err != nil {
				return []byte{}, err
			}
			sec = newSec
			document = append(document, doc...)
			continue
		}

		if sec.kind == 0 {
			continue
		}
		lines := formats[sec.kind](values)

		var (
			padding int
			newCmd  *exec.Cmd
		)
		padding, lines, newCmd, optionalRulePadding = applySectionKind(
			ctx, sec, paddings, indent, optionalRulePadding, aliases, lines,
		)
		if newCmd != nil {
			cmd = newCmd
		}

		computed, lines, err := computeCommand(cmd, lines, sec)
		if err != nil {
			return []byte{}, err
		}
		if computed {
			cmd = nil
		}
		document = append(document, trimExtraTrailingSpace(indentStrings(padding, lines))...)
	}

	return []byte(fmt.Sprintf("%s\n", strings.Join(document, "\n"))), nil
}

func applySectionKind(
	ctx context.Context,
	sec *section,
	paddings map[gherkin.TokenType]int,
	indent int,
	optionalRulePadding int,
	aliases aliases,
	lines []string,
) (padding int, result []string, cmd *exec.Cmd, newOptionalRulePadding int) {
	padding = paddings[sec.kind] + optionalRulePadding
	result = lines
	newOptionalRulePadding = optionalRulePadding

	switch sec.kind {
	case gherkin.TokenTypeRuleLine:
		newOptionalRulePadding = indent
		padding = indent
	case gherkin.TokenTypeComment, gherkin.TokenTypeLanguage:
		cmd = extractCommand(ctx, sec.values, aliases)
		padding = getTagOrCommentPadding(paddings, indent, sec)
		result = trimLinesSpace(lines)
	case gherkin.TokenTypeTagLine:
		padding = getTagOrCommentPadding(paddings, indent, sec)
	case gherkin.TokenTypeDocStringSeparator:
		result = extractKeyword(sec.values)
	case gherkin.TokenTypeOther:
		if isDescriptionFeature(sec) {
			result = trimLinesSpace(lines)
			padding = indent
		} else if isDescriptionScenario(sec) {
			result = trimLinesSpace(lines)
			padding = paddings[gherkin.TokenTypeScenarioLine] + optionalRulePadding
		}
	default:
	}

	return padding, result, cmd, newOptionalRulePadding
}

func getTagOrCommentPadding(
	paddings map[gherkin.TokenType]int,
	indent int,
	sec *section,
) int {
	var kind gherkin.TokenType
	excluded := []gherkin.TokenType{
		gherkin.TokenTypeTagLine,
		gherkin.TokenTypeComment,
		gherkin.TokenTypeEmpty,
	}
	if sec.next(excluded) != nil {
		if s := sec.next(excluded); s != nil {
			kind = s.kind
		}
	}
	if kind == 0 && sec.previous(excluded) != nil {
		if s := sec.previous(excluded); s != nil {
			kind = s.kind
		}
	}
	// indent the last comment line at the same level than scenario and background.
	if sec.next([]gherkin.TokenType{gherkin.TokenTypeEmpty}) == nil {
		return indent
	}
	return paddings[kind]
}

func computeCommand(
	cmd *exec.Cmd,
	lines []string,
	sec *section,
) (computed bool, result []string, err error) {
	if sec.kind == gherkin.TokenTypeComment ||
		sec.kind == gherkin.TokenTypeDocStringSeparator || cmd == nil {
		return false, lines, nil
	}
	l, err := runCommand(cmd, lines)
	if err != nil {
		return true, []string{}, err
	}
	return true, l, err
}

func isDescriptionFeature(sec *section) bool {
	excluded := []gherkin.TokenType{gherkin.TokenTypeEmpty}
	if sec.previous(excluded) != nil {
		if s := sec.previous(excluded); s != nil && s.kind == gherkin.TokenTypeFeatureLine {
			return true
		}
	}
	return false
}

func isDescriptionScenario(sec *section) bool {
	excluded := []gherkin.TokenType{gherkin.TokenTypeEmpty}
	if sec.previous(excluded) != nil {
		if s := sec.previous(excluded); s != nil && s.kind == gherkin.TokenTypeScenarioLine {
			return true
		}
	}
	return false
}

func trimLinesSpace(lines []string) []string {
	content := make([]string, 0, len(lines))
	for _, line := range lines {
		content = append(content, strings.TrimSpace(line))
	}
	return content
}

func trimExtraTrailingSpace(lines []string) []string {
	content := make([]string, 0, len(lines))
	for _, line := range lines {
		content = append(content, strings.TrimRight(line, " \t"))
	}
	return content
}

func indentStrings(padding int, lines []string) []string {
	content := make([]string, 0, len(lines))
	for _, line := range lines {
		content = append(content, strings.Repeat(" ", padding)+line)
	}
	return content
}

func extractLanguage(tokens []*gherkin.Token) []string {
	return []string{fmt.Sprintf("# language: %s", tokens[0].Text)}
}

func extractTokensText(tokens []*gherkin.Token) []string {
	content := make([]string, 0, len(tokens))
	for _, token := range tokens {
		content = append(content, token.Text)
	}
	return content
}

func extractTokensItemsText(tokens []*gherkin.Token) []string {
	content := make([]string, 0, len(tokens))
	for _, token := range tokens {
		t := make([]string, 0, len(token.Items))
		for _, item := range token.Items {
			t = append(t, item.Text)
		}
		content = append(content, strings.Join(t, " "))
	}
	return content
}

func extractTokensKeywordAndText(tokens []*gherkin.Token) []string {
	content := make([]string, 0, len(tokens))
	for _, token := range tokens {
		content = append(content, fmt.Sprintf("%s%s", token.Keyword, token.Text))
	}
	return content
}

func extractKeywordAndTextSeparatedWithAColon(tokens []*gherkin.Token) []string {
	content := make([]string, 0, len(tokens))
	for _, token := range tokens {
		content = append(content, fmt.Sprintf("%s: %s", token.Keyword, token.Text))
	}
	return content
}

func extractKeyword(tokens []*gherkin.Token) []string {
	content := make([]string, 0, len(tokens))
	for _, t := range tokens {
		content = append(content, t.Keyword)
	}
	return content
}

func extractTableRowsAndComments(tokens []*gherkin.Token) []string {
	type tableElement struct {
		content []string
		kind    gherkin.TokenType
	}
	rows := [][]string{}
	tableElements := make([]tableElement, 0, len(tokens))
	for _, token := range tokens {
		element := tableElement{}
		if token.Type == gherkin.TokenTypeComment {
			element.kind = token.Type
			element.content = []string{token.Text}
		} else {
			row := []string{}
			for _, data := range token.Items {
				// A remaining pipe means it was escaped before to not be messed
				// with pipe column delimiter so here we introduce the escaping
				// sequence back.
				text := data.Text
				if strings.Contains(text, "\\\n") {
					text = strings.ReplaceAll(text, "\\\n", "\\\\\\n")
				}
				text = strings.ReplaceAll(text, "\n", "\\n")
				text = strings.ReplaceAll(text, "|", "\\|")
				row = append(row, text)
			}
			element.kind = token.Type
			element.content = row
			rows = append(rows, row)
		}
		tableElements = append(tableElements, element)
	}

	tableRows := make([]string, 0, len(tableElements))
	lengths := calculateLonguestLineLengthPerColumn(rows)
	for _, tableElement := range tableElements {
		inputs := []any{}
		fmtDirective := ""
		if tableElement.kind == gherkin.TokenTypeComment {
			inputs = append(inputs, trimLinesSpace(tableElement.content)[0])
			fmtDirective = "%s"
		} else {
			for i, str := range tableElement.content {
				inputs = append(inputs, str)
				fmtDirective += fmt.Sprintf("| %%-%ds ", lengths[i])
			}
			fmtDirective += "|"
		}
		tableRows = append(tableRows, fmt.Sprintf(fmtDirective, inputs...))
	}
	return tableRows
}

func calculateLonguestLineLengthPerColumn(rows [][]string) []int {
	lengths := []int{}
	for i, row := range rows {
		for j, str := range row {
			switch {
			case i == 0:
				lengths = append(lengths, utf8.RuneCountInString(str))
			case i != 0 && len(lengths) > j && lengths[j] < utf8.RuneCountInString(str):
				lengths[j] = utf8.RuneCountInString(str)
			default:
				lengths = append(lengths, 0)
			}
		}
	}
	return lengths
}

func extractCommand(ctx context.Context, tokens []*gherkin.Token, aliases map[string]string) *exec.Cmd {
	re := regexp.MustCompile(`(@[a-zA-Z0-9]+)`)
	matches := re.FindStringSubmatch(tokens[0].Text)
	if len(matches) == 0 {
		return nil
	}
	if cmd, ok := aliases[matches[0][1:]]; ok {
		return exec.CommandContext(ctx, "sh", "-c", cmd) //nolint:gosec // aliases are user-defined.
	}
	return nil
}

func runCommand(cmd *exec.Cmd, lines []string) ([]string, error) {
	if len(lines) == 0 {
		return lines, nil
	}

	cmd.Stdin = strings.NewReader(strings.Join(lines, "\n"))
	o, err := cmd.CombinedOutput()
	if err != nil {
		return []string{}, CmdError{strings.TrimRight(string(o), "\n")}
	}
	return strings.Split(strings.TrimRight(string(o), "\n"), "\n"), nil
}
