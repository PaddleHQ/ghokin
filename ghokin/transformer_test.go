package ghokin_test

import (
	"os"
	"os/exec"
	"testing"

	"github.com/PaddleHQ/ghokin/v4/ghokin"

	gherkin "github.com/cucumber/gherkin/go/v28"
	"github.com/stretchr/testify/assert"
)

func TestIndentStrings(t *testing.T) {
	datas := []string{
		"hello",
		"world",
	}

	expected := []string{
		"    hello",
		"    world",
	}

	assert.Equal(t, expected, ghokin.ExportIndentStrings(4, datas))
}

func TestExtractTokensText(t *testing.T) {
	tokens := []*gherkin.Token{
		{
			Text: "test1",
		},
		{
			Text: "test2",
		},
	}

	expected := []string{"test1", "test2"}

	assert.Equal(t, expected, ghokin.ExportExtractTokensText(tokens))
}

func TestExtractTokensItemsText(t *testing.T) {
	tokens := []*gherkin.Token{
		{
			Items: []*gherkin.LineSpan{
				{Text: "@test1"},
				{Text: "@test2"},
			},
		},
		{
			Items: []*gherkin.LineSpan{
				{Text: "@test3"},
				{Text: "@test4"},
			},
		},
	}

	expected := []string{"@test1 @test2", "@test3 @test4"}

	assert.Equal(t, expected, ghokin.ExportExtractTokensItemsText(tokens))
}

func TestExtractTokensKeywordAndText(t *testing.T) {
	tokens := []*gherkin.Token{
		{Keyword: "Then ", Text: "match some JSON properties"},
		{Keyword: "Then ", Text: "we do something"},
	}

	expected := []string{
		"Then match some JSON properties",
		"Then we do something",
	}

	assert.Equal(t, expected, ghokin.ExportExtractTokensKeywordAndText(tokens))
}

func TestExtractKeywordAndTextSeparatedWithAColon(t *testing.T) {
	tokens := []*gherkin.Token{{Keyword: "Feature", Text: "Set api"}}
	expected := []string{"Feature: Set api"}

	assert.Equal(t, expected, ghokin.ExportExtractKeywordAndTextSeparatedWithAColon(tokens))
}

func TestExtractKeyword(t *testing.T) {
	tokens := []*gherkin.Token{{Keyword: `"""`}}
	expected := []string{`"""`}

	assert.Equal(t, expected, ghokin.ExportExtractKeyword(tokens))
}

func TestExtractTableRows(t *testing.T) {
	type scenario struct {
		tokens []*gherkin.Token
		test   func([]string)
	}

	scenarios := []scenario{
		{
			[]*gherkin.Token{
				{
					Items: []*gherkin.LineSpan{
						{Text: "whatever"},
						{Text: "whatever whatever"},
					},
				},
				{
					Items: []*gherkin.LineSpan{
						{Text: "test"},
						{Text: "test"},
					},
				},
				{
					Items: []*gherkin.LineSpan{
						{Text: "t"},
						{Text: "t"},
					},
				},
			},
			func(output []string) {
				expected := []string{
					"| whatever | whatever whatever |",
					"| test     | test              |",
					"| t        | t                 |",
				}
				assert.Equal(t, expected, output)
			},
		},
	}

	for _, scenario := range scenarios {
		scenario.test(ghokin.ExportExtractTableRowsAndComments(scenario.tokens))
	}
}

func TestExtractCommand(t *testing.T) {
	type scenario struct {
		tokens []*gherkin.Token
		test   func(*exec.Cmd)
	}

	aliases := map[string]string{
		"cat": "cat",
		"jq":  "jq",
	}

	scenarios := []scenario{
		{
			[]*gherkin.Token{{
				Text: "",
			}},
			func(cmd *exec.Cmd) {
				assert.Nil(t, cmd)
			},
		},
		{
			[]*gherkin.Token{{
				Text: "# A comment",
			}},
			func(cmd *exec.Cmd) {
				assert.Nil(t, cmd)
			},
		},
		{
			[]*gherkin.Token{{
				Text: "# @jq",
			}},
			func(cmd *exec.Cmd) {
				assert.NotNil(t, cmd)
				assert.Equal(t, []string{"sh", "-c", "jq"}, cmd.Args)
			},
		},
	}

	for _, scenario := range scenarios {
		scenario.test(ghokin.ExportExtractCommand(scenario.tokens, aliases))
	}
}

func TestTrimLinesSpace(t *testing.T) {
	datas := []string{
		"                        hello                          ",
		`		world


		`,
	}

	expected := []string{
		"hello",
		"world",
	}

	assert.Equal(t, expected, ghokin.ExportTrimLinesSpace(datas))
}

func TestRunCommand(t *testing.T) {
	type scenario struct {
		cmd   *exec.Cmd
		lines []string
		test  func([]string, error)
	}

	scenarios := []scenario{
		{
			nil,
			[]string{},
			func(lines []string, err error) {
				assert.Empty(t, lines)
				assert.NoError(t, err)
			},
		},
		{
			ghokin.NewCommandForTest("sh", "-c", "cat"),
			[]string{"hello world !", "hello universe !"},
			func(lines []string, err error) {
				assert.Equal(t, []string{"hello world !", "hello universe !"}, lines)
				assert.NoError(t, err)
			},
		},
		{
			ghokin.NewCommandForTest("sh", "-c", "catttttt"),
			[]string{"hello world !", "hello universe !"},
			func(lines []string, err error) {
				assert.Equal(t, []string{}, lines)
				assert.Regexp(t, ".*catttttt.*(not found|introuvable).*", err.Error())
			},
		},
	}

	for _, scenario := range scenarios {
		scenario.test(ghokin.ExportRunCommand(scenario.cmd, scenario.lines))
	}
}

func TestExtractSections(t *testing.T) {
	type scenario struct {
		filename string
		test     func(*ghokin.ExportSection, error)
	}

	scenarios := []scenario{
		{
			"fixtures/file.txt",
			func(_ *ghokin.ExportSection, err error) {
				assert.ErrorContains(
					t, err,
					"Parser errors:\n(1:1): expected: #EOF, #Language, #TagLine, #FeatureLine, #Comment, #Empty, got 'whatever'",
				)
			},
		},
		{
			"fixtures/feature.feature",
			func(sec *ghokin.ExportSection, err error) {
				type test struct {
					previousName string
					currentName  string
					nextName     string
					values       []map[string]string
				}

				assert.NoError(t, err)
				assert.Empty(t, ghokin.SectionKindName(sec))

				ts := []test{
					{
						"",
						"FeatureLine",
						"Other",
						[]map[string]string{
							{
								"keyword": "Feature",
								"text":    "Test",
							},
						},
					},
					{
						"FeatureLine",
						"Other",
						"BackgroundLine",
						[]map[string]string{
							{
								"keyword": "",
								"text":    "  This is a description",
							},
							{
								"keyword": "",
								"text":    "",
							},
						},
					},
					{
						"Other",
						"BackgroundLine",
						"StepLine",
						[]map[string]string{
							{
								"keyword": "Background",
								"text":    "",
							},
						},
					},
					{
						"BackgroundLine",
						"StepLine",
						"ScenarioLine",
						[]map[string]string{
							{
								"keyword": "Given ",
								"text":    "something",
							},
						},
					},
					{
						"StepLine",
						"ScenarioLine",
						"StepLine",
						[]map[string]string{
							{
								"keyword": "Scenario",
								"text":    "A scenario to test",
							},
						},
					},
					{
						"ScenarioLine",
						"StepLine",
						"ScenarioLine",
						[]map[string]string{
							{
								"keyword": "Given ",
								"text":    "a thing",
							},
							{
								"keyword": "Given ",
								"text":    "something else",
							},
							{
								"keyword": "Then ",
								"text":    "something happened",
							},
						},
					},
					{
						"StepLine",
						"ScenarioLine",
						"StepLine",
						[]map[string]string{
							{
								"keyword": "Scenario",
								"text":    "Another scenario to test",
							},
						},
					},
					{
						"ScenarioLine",
						"StepLine",
						"",
						[]map[string]string{
							{
								"keyword": "Given ",
								"text":    "a second thing",
							},
							{
								"keyword": "Given ",
								"text":    "another second thing",
							},
							{
								"keyword": "Then ",
								"text":    "another thing happened",
							},
						},
					},
				}

				emptyExcl := []gherkin.TokenType{gherkin.TokenTypeEmpty}
				sec = ghokin.SectionNext(sec, emptyExcl)

				for i := range ts {
					prev := ghokin.SectionPrevious(sec, emptyExcl)
					assert.Equal(t, ghokin.SectionKindName(prev), ts[i].previousName)
					assert.Equal(t, ghokin.SectionKindName(sec), ts[i].currentName)

					if i == len(ts)-1 {
						assert.Equal(
							t,
							(*ghokin.ExportSection)(nil),
							ghokin.SectionNext(sec, emptyExcl),
						)
					} else {
						nxt := ghokin.SectionNext(sec, emptyExcl)
						assert.Equal(t, ghokin.SectionKindName(nxt), ts[i].nextName)
					}

					for j, v := range ghokin.SectionValues(sec) {
						assert.Equal(t, ts[i].values[j]["keyword"], v.Keyword)
						assert.Equal(t, ts[i].values[j]["text"], v.Text)
					}

					sec = ghokin.SectionNext(sec, emptyExcl)
				}
			},
		},
	}

	for _, scenario := range scenarios {
		content, err := os.ReadFile(scenario.filename)
		assert.NoError(t, err)
		scenario.test(ghokin.ExportExtractSections(content))
	}
}

func TestTransform(t *testing.T) {
	type scenario struct {
		input    string
		expected string
	}

	scenarios := []scenario{
		{
			"fixtures/file1.feature",
			"fixtures/file1.feature",
		},
		{
			"fixtures/json.input.feature",
			"fixtures/json.expected.feature",
		},

		{
			"fixtures/cmd.input.feature",
			"fixtures/cmd.expected.feature",
		},
		{
			"fixtures/multisize-table.input.feature",
			"fixtures/multisize-table.expected.feature",
		},
		{
			"fixtures/docstring-empty.input.feature",
			"fixtures/docstring-empty.expected.feature",
		},
		{
			"fixtures/comment-after-scenario.feature",
			"fixtures/comment-after-scenario.feature",
		},
		{
			"fixtures/comment-after-background.feature",
			"fixtures/comment-after-background.feature",
		},
		{
			"fixtures/comment-with-newline.feature",
			"fixtures/comment-with-newline.feature",
		},
		{
			"fixtures/escape-pipe.feature",
			"fixtures/escape-pipe.feature",
		},
		{
			"fixtures/escape-new-line.feature",
			"fixtures/escape-new-line.feature",
		},
		{
			"fixtures/several-scenario-following.feature",
			"fixtures/several-scenario-following.feature",
		},
		{
			"fixtures/rule.feature",
			"fixtures/rule.feature",
		},
		{
			"fixtures/non-ascii-characters-formatting.feature",
			"fixtures/non-ascii-characters-formatting.feature",
		},
		{
			"fixtures/double-escaping.feature",
			"fixtures/double-escaping.feature",
		},
		{
			"fixtures/comment-in-a-midst-of-row.feature",
			"fixtures/comment-in-a-midst-of-row.feature",
		},
		{
			"fixtures/scenario-description.feature",
			"fixtures/scenario-description.feature",
		},
		{
			"fixtures/escaping-in-examples.feature",
			"fixtures/escaping-in-examples.feature",
		},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.input, func(t *testing.T) {
			t.Parallel()
			content, err := os.ReadFile(scenario.input)
			assert.NoError(t, err)
			s, err := ghokin.ExportExtractSections(content)
			assert.NoError(t, err)

			aliases := map[string]string{
				"seq": "seq 1 3",
			}

			buf, err := ghokin.ExportTransform(s, 2, aliases)
			assert.NoError(t, err)

			b, e := os.ReadFile(scenario.expected)
			assert.NoError(t, e)
			assert.Equal(t, string(b), string(buf))
		})
	}
}
