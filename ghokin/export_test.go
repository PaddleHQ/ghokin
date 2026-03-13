package ghokin

import (
	"context"
	"os/exec"

	gherkin "github.com/cucumber/gherkin/go/v28"
)

// ExportSection is an alias for section, exported for testing.
type ExportSection = section

// Exported function references for testing.
var (
	ExportIndentStrings                            = indentStrings
	ExportExtractTokensText                        = extractTokensText
	ExportExtractTokensItemsText                   = extractTokensItemsText
	ExportExtractTokensKeywordAndText              = extractTokensKeywordAndText
	ExportExtractKeywordAndTextSeparatedWithAColon = extractKeywordAndTextSeparatedWithAColon
	ExportExtractKeyword                           = extractKeyword
	ExportExtractTableRowsAndComments              = extractTableRowsAndComments
	ExportExtractCommand                           = extractCommand
	ExportTrimLinesSpace                           = trimLinesSpace
	ExportRunCommand                               = runCommand
	ExportExtractSections                          = extractSections
	ExportTransform                                = transform
)

// SectionKindName returns the name of the section's kind.
func SectionKindName(s *section) string {
	if s == nil {
		return ""
	}
	return s.kind.Name()
}

// SectionValues returns the section's values.
func SectionValues(s *section) []*gherkin.Token {
	return s.values
}

// SectionNext returns the next section excluding given types.
func SectionNext(s *section, excluded []gherkin.TokenType) *section {
	return s.next(excluded)
}

// SectionPrevious returns the previous section excluding given types.
func SectionPrevious(s *section, excluded []gherkin.TokenType) *section {
	return s.previous(excluded)
}

// NewCommandForTest creates exec.Cmd for testing.
func NewCommandForTest(ctx context.Context, name string, args ...string) *exec.Cmd {
	return exec.CommandContext(ctx, name, args...)
}
