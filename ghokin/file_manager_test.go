package ghokin_test

import (
	"fmt"
	"os"
	"testing"

	"github.com/PaddleHQ/ghokin/v4/ghokin"

	"github.com/stretchr/testify/assert"
)

func TestFileManagerTransform(t *testing.T) {
	type scenario struct {
		filename string
		test     func([]byte, error)
	}

	scenarios := []scenario{
		{
			"fixtures/file1.feature",
			func(buf []byte, _ error) {
				b, e := os.ReadFile("fixtures/file1.feature")
				assert.NoError(t, e)
				assert.Equal(t, string(b), string(buf))
			},
		},
		{
			"fixtures/utf8-with-bom.feature",
			func(buf []byte, _ error) {
				b, e := os.ReadFile("fixtures/utf8-with-bom.feature")
				assert.NoError(t, e)
				assert.Equal(t, string(b), string(buf))
			},
		},
		{
			"fixtures/file1-with-cr.feature",
			func(buf []byte, _ error) {
				b, e := os.ReadFile("fixtures/file1-with-cr.feature")
				assert.NoError(t, e)
				assert.Equal(t, string(b), string(buf))
			},
		},
		{
			"fixtures/file1-with-crlf.feature",
			func(buf []byte, _ error) {
				b, e := os.ReadFile("fixtures/file1-with-crlf.feature")
				assert.NoError(t, e)
				assert.Equal(t, string(b), string(buf))
			},
		},
		{
			"fixtures/iso-8859-1-encoding.input.feature",
			func(buf []byte, err error) {
				assert.NoError(t, err)
				b, e := os.ReadFile("fixtures/iso-8859-1-encoding.expected.feature")
				assert.NoError(t, e)
				assert.Equal(t, string(b), string(buf))
			},
		},
		{
			"fixtures/",
			func(_ []byte, err error) {
				assert.EqualError(t, err, "failed to read file: read fixtures: is a directory")
			},
		},
		{
			"fixtures/invalid.feature",
			func(_ []byte, err error) {
				assert.Error(t, err)
			},
		},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.filename, func(t *testing.T) {
			t.Parallel()
			f := ghokin.NewFileManager(
				2,
				map[string]string{
					"seq": "seq 1 3",
				},
			)
			scenario.test(f.Transform(t.Context(), scenario.filename))
		})
	}
}

func TestFileManagerTransformAndReplace(t *testing.T) {
	tmpDir := t.TempDir()

	type scenario struct {
		testName   string
		path       string
		extensions []string
		setup      func()
		test       func([]error)
	}

	scenarios := []scenario{
		{
			"Format a file",
			fmt.Sprintf("%s/file1.feature", tmpDir),
			[]string{"feature"},
			func() {
				content := []byte(`Feature: test
   test

Scenario:            scenario1
   Given       whatever
   Then                  whatever
"""
hello world
"""
`)

				assert.NoError(t, os.RemoveAll(fmt.Sprintf("%s/file1.feature", tmpDir)))
				assert.NoError(t, os.WriteFile(fmt.Sprintf("%s/file1.feature", tmpDir), content, 0o777))
			},
			func(errs []error) {
				assert.Empty(t, errs)

				content := `Feature: test
  test

  Scenario: scenario1
    Given whatever
    Then whatever
      """
      hello world
      """
`

				b, e := os.ReadFile(fmt.Sprintf("%s/file1.feature", tmpDir))
				assert.NoError(t, e)
				assert.Equal(t, content, string(b))
			},
		},
		{
			"Format a folder",
			fmt.Sprintf("%s/", tmpDir),
			[]string{"feature"},
			func() {
				content := []byte(`Feature: test
        test

Scenario:   scenario%d
   Given           whatever
   Then      whatever
"""
hello world
"""
`)

				// Clean up any files from previous scenarios
				entries, _ := os.ReadDir(tmpDir)
				for _, e := range entries {
					assert.NoError(t, os.RemoveAll(fmt.Sprintf("%s/%s", tmpDir, e.Name())))
				}

				assert.NoError(t, os.MkdirAll(fmt.Sprintf("%s/test1", tmpDir), 0o777))
				assert.NoError(t, os.MkdirAll(fmt.Sprintf("%s/test2/test3", tmpDir), 0o777))

				for i, f := range []string{
					fmt.Sprintf("%s/file1.feature", tmpDir),
					fmt.Sprintf("%s/file2.feature", tmpDir),
					fmt.Sprintf("%s/test1/file3.feature", tmpDir),
					fmt.Sprintf("%s/test1/file4.feature", tmpDir),
					fmt.Sprintf("%s/test2/test3/file5.feature", tmpDir),
					fmt.Sprintf("%s/test2/test3/file6.feature", tmpDir),
				} {
					assert.NoError(t, os.WriteFile(f, fmt.Appendf(nil, string(content), i), 0o777))
				}
			},
			func(errs []error) {
				assert.Empty(t, errs)

				content := `Feature: test
  test

  Scenario: scenario%d
    Given whatever
    Then whatever
      """
      hello world
      """
`

				for i, f := range []string{
					fmt.Sprintf("%s/file1.feature", tmpDir),
					fmt.Sprintf("%s/file2.feature", tmpDir),
					fmt.Sprintf("%s/test1/file3.feature", tmpDir),
					fmt.Sprintf("%s/test1/file4.feature", tmpDir),
					fmt.Sprintf("%s/test2/test3/file5.feature", tmpDir),
					fmt.Sprintf("%s/test2/test3/file6.feature", tmpDir),
				} {
					b, e := os.ReadFile(f)
					assert.NoError(t, e)
					assert.Equal(t, fmt.Sprintf(content, i), string(b))
				}
			},
		},
		{
			"Format a folder with parsing errors",
			fmt.Sprintf("%s/", tmpDir),
			[]string{"feature"},
			func() {
				content := []byte(`Feature: test
      test

Scenario:   scenario
   Given           whatever
   Then      whatever
"""
hello world
"""
`)

				entries, _ := os.ReadDir(tmpDir)
				for _, e := range entries {
					assert.NoError(t, os.RemoveAll(fmt.Sprintf("%s/%s", tmpDir, e.Name())))
				}

				assert.NoError(t, os.MkdirAll(fmt.Sprintf("%s/test1", tmpDir), 0o777))

				assert.NoError(t, os.WriteFile(fmt.Sprintf("%s/file1.feature", tmpDir), content, 0o777))
				assert.NoError(t, os.WriteFile(fmt.Sprintf("%s/file2.feature", tmpDir), append([]byte("whatever"), content...), 0o777))
				assert.NoError(t, os.WriteFile(fmt.Sprintf("%s/test1/file3.feature", tmpDir), content, 0o777))
				assert.NoError(t, os.WriteFile(fmt.Sprintf("%s/test1/file4.feature", tmpDir), content, 0o777))
				assert.NoError(t, os.WriteFile(fmt.Sprintf("%s/test1/file5.feature", tmpDir), append([]byte("whatever"), content...), 0o777))
			},
			func(errs []error) {
				assert.Len(t, errs, 2)

				msgs := []string{
					fmt.Sprintf("an error occurred with file \"%s/file2.feature\" : failed to parse gherkin: Parser errors:\n(1:1): expected: #EOF, #Language, #TagLine, #FeatureLine, #Comment, #Empty, got 'whateverFeature: test'", tmpDir),
					fmt.Sprintf("an error occurred with file \"%s/test1/file5.feature\" : failed to parse gherkin: Parser errors:\n(1:1): expected: #EOF, #Language, #TagLine, #FeatureLine, #Comment, #Empty, got 'whateverFeature: test'", tmpDir),
				}

				for _, e := range errs {
					var match bool
					for _, msg := range msgs {
						if msg == e.Error() {
							match = true
						}
					}

					if !match {
						assert.Fail(t, "Must fail with 2 files when formatting folder")
					}
				}
			},
		},
		{
			"Format a folder and set various extensions for feature files",
			fmt.Sprintf("%s/", tmpDir),
			[]string{"txt", "feat"},
			func() {
				content := []byte(`Feature: test
   test

Scenario:   scenario
   Given           whatever
   Then      whatever
"""
hello world
"""
`)

				entries, _ := os.ReadDir(tmpDir)
				for _, e := range entries {
					assert.NoError(t, os.RemoveAll(fmt.Sprintf("%s/%s", tmpDir, e.Name())))
				}

				assert.NoError(t, os.WriteFile(fmt.Sprintf("%s/file1.feature", tmpDir), content, 0o777))
				assert.NoError(t, os.WriteFile(fmt.Sprintf("%s/file2.txt", tmpDir), content, 0o777))
				assert.NoError(t, os.WriteFile(fmt.Sprintf("%s/file3.feat", tmpDir), content, 0o777))
			},
			func(errs []error) {
				assert.Empty(t, errs)

				contentFormatted := `Feature: test
  test

  Scenario: scenario
    Given whatever
    Then whatever
      """
      hello world
      """
`

				contentUnformatted := `Feature: test
   test

Scenario:   scenario
   Given           whatever
   Then      whatever
"""
hello world
"""
`

				for _, s := range []struct {
					filename string
					expected string
				}{
					{
						fmt.Sprintf("%s/file1.feature", tmpDir),
						contentUnformatted,
					},
					{
						fmt.Sprintf("%s/file2.txt", tmpDir),
						contentFormatted,
					},
					{
						fmt.Sprintf("%s/file3.feat", tmpDir),
						contentFormatted,
					},
				} {
					b, e := os.ReadFile(s.filename)
					assert.NoError(t, e)
					assert.Equal(t, s.expected, string(b))
				}
			},
		},
		{
			"Format folder with no feature files",
			tmpDir,
			[]string{"feature"},
			func() {
				entries, _ := os.ReadDir(tmpDir)
				for _, e := range entries {
					assert.NoError(t, os.RemoveAll(fmt.Sprintf("%s/%s", tmpDir, e.Name())))
				}

				assert.NoError(t, os.WriteFile(fmt.Sprintf("%s/file1.txt", tmpDir), []byte("file1"), 0o777))
				assert.NoError(t, os.WriteFile(fmt.Sprintf("%s/file2.txt", tmpDir), []byte("file2"), 0o777))
			},
			func(errs []error) {
				assert.Empty(t, errs)
			},
		},
		{
			"Format a file with different extension and an error",
			"fixtures/file.txt",
			[]string{"txt"},
			func() {},
			func(errs []error) {
				assert.Len(t, errs, 1)
				assert.EqualError(
					t, errs[0],
					"failed to parse gherkin: Parser errors:\n(1:1): expected: #EOF, #Language, #TagLine, #FeatureLine, #Comment, #Empty, got 'whatever'",
				)
			},
		},
		{
			"Format an unexisting folder",
			"whatever/whatever",
			[]string{"feature"},
			func() {},
			func(errs []error) {
				assert.Len(t, errs, 1)
				assert.EqualError(t, errs[0], "stat whatever/whatever: no such file or directory")
			},
		},
		{
			"Format an invalid file",
			"fixtures/invalid.feature",
			[]string{"feature"},
			func() {},
			func(errs []error) {
				assert.Len(t, errs, 1)
			},
		},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.testName, func(_ *testing.T) {
			scenario.setup()
			f := ghokin.NewFileManager(
				2,
				map[string]string{
					"seq": "seq 1 3",
				},
			)
			scenario.test(f.TransformAndReplace(t.Context(), scenario.path, scenario.extensions))
		})
	}
}

func TestFileManagerCheck(t *testing.T) {
	tmpDir := t.TempDir()

	type scenario struct {
		testName   string
		path       string
		extensions []string
		setup      func()
		test       func([]error)
	}

	scenarios := []scenario{
		{
			"Check a file wrongly formatted",
			fmt.Sprintf("%s/file1.feature", tmpDir),
			[]string{"feature"},
			func() {
				content := []byte(`Feature: test
   test

Scenario:            scenario1
   Given       whatever
   Then                  whatever
"""
hello world
"""
`)

				assert.NoError(t, os.WriteFile(fmt.Sprintf("%s/file1.feature", tmpDir), content, 0o777))
			},
			func(errs []error) {
				assert.Len(t, errs, 1)
				assert.EqualError(
					t, errs[0],
					fmt.Sprintf("an error occurred with file \"%s/file1.feature\" : file is not properly formatted", tmpDir),
				)
			},
		},
		{
			"Check a file correctly formatted",
			fmt.Sprintf("%s/file1.feature", tmpDir),
			[]string{"feature"},
			func() {
				content := []byte(`Feature: test

  Scenario: scenario
    Given whatever
    Then whatever
      """
      hello world
      """
`)

				assert.NoError(t, os.WriteFile(fmt.Sprintf("%s/file1.feature", tmpDir), content, 0o777))
			},
			func(errs []error) {
				assert.Empty(t, errs)
			},
		},
		{
			"Check a folder is wrongly formatted",
			fmt.Sprintf("%s/", tmpDir),
			[]string{"feature"},
			func() {
				content := []byte(`Feature: test
   test

Scenario:   scenario%d
   Given           whatever
   Then      whatever
"""
hello world
"""
`)

				entries, _ := os.ReadDir(tmpDir)
				for _, e := range entries {
					assert.NoError(t, os.RemoveAll(fmt.Sprintf("%s/%s", tmpDir, e.Name())))
				}

				assert.NoError(t, os.MkdirAll(fmt.Sprintf("%s/test1", tmpDir), 0o777))
				assert.NoError(t, os.MkdirAll(fmt.Sprintf("%s/test2/test3", tmpDir), 0o777))

				for i, f := range []string{
					fmt.Sprintf("%s/file1.feature", tmpDir),
					fmt.Sprintf("%s/file2.feature", tmpDir),
					fmt.Sprintf("%s/test1/file3.feature", tmpDir),
					fmt.Sprintf("%s/test1/file4.feature", tmpDir),
					fmt.Sprintf("%s/test2/test3/file5.feature", tmpDir),
					fmt.Sprintf("%s/test2/test3/file6.feature", tmpDir),
				} {
					assert.NoError(t, os.WriteFile(f, fmt.Appendf(nil, string(content), i), 0o777))
				}
			},
			func(errs []error) {
				assert.Len(t, errs, 6)

				errors := map[string]bool{
					fmt.Sprintf("an error occurred with file \"%s/file1.feature\" : file is not properly formatted", tmpDir):             true,
					fmt.Sprintf("an error occurred with file \"%s/file2.feature\" : file is not properly formatted", tmpDir):             true,
					fmt.Sprintf("an error occurred with file \"%s/test1/file3.feature\" : file is not properly formatted", tmpDir):       true,
					fmt.Sprintf("an error occurred with file \"%s/test1/file4.feature\" : file is not properly formatted", tmpDir):       true,
					fmt.Sprintf("an error occurred with file \"%s/test2/test3/file5.feature\" : file is not properly formatted", tmpDir): true,
					fmt.Sprintf("an error occurred with file \"%s/test2/test3/file6.feature\" : file is not properly formatted", tmpDir): true,
				}

				for _, err := range errs {
					if _, ok := errors[err.Error()]; !ok {
						assert.Fail(t, fmt.Sprintf("error %s doesn't exists", err.Error()))
					}
				}
			},
		},
		{
			"Check a folder is correctly formatted",
			fmt.Sprintf("%s/", tmpDir),
			[]string{"feature"},
			func() {
				content := []byte(`Feature: test

  Scenario: scenario%d
    Given whatever
    Then whatever
      """
      hello world
      """
`)

				entries, _ := os.ReadDir(tmpDir)
				for _, e := range entries {
					assert.NoError(t, os.RemoveAll(fmt.Sprintf("%s/%s", tmpDir, e.Name())))
				}

				assert.NoError(t, os.MkdirAll(fmt.Sprintf("%s/test1", tmpDir), 0o777))
				assert.NoError(t, os.MkdirAll(fmt.Sprintf("%s/test2/test3", tmpDir), 0o777))

				for i, f := range []string{
					fmt.Sprintf("%s/file1.feature", tmpDir),
					fmt.Sprintf("%s/file2.feature", tmpDir),
					fmt.Sprintf("%s/test1/file3.feature", tmpDir),
					fmt.Sprintf("%s/test1/file4.feature", tmpDir),
					fmt.Sprintf("%s/test2/test3/file5.feature", tmpDir),
					fmt.Sprintf("%s/test2/test3/file6.feature", tmpDir),
				} {
					assert.NoError(t, os.WriteFile(f, fmt.Appendf(nil, string(content), i), 0o777))
				}
			},
			func(errs []error) {
				assert.Empty(t, errs)
			},
		},
		{
			"Check a folder with parsing errors",
			fmt.Sprintf("%s/", tmpDir),
			[]string{"feature"},
			func() {
				content := []byte(`Feature: test

  Scenario: scenario
    Given whatever
    Then whatever
      """
      hello world
      """
`)

				entries, _ := os.ReadDir(tmpDir)
				for _, e := range entries {
					assert.NoError(t, os.RemoveAll(fmt.Sprintf("%s/%s", tmpDir, e.Name())))
				}

				assert.NoError(t, os.MkdirAll(fmt.Sprintf("%s/test1", tmpDir), 0o777))

				assert.NoError(t, os.WriteFile(fmt.Sprintf("%s/file1.feature", tmpDir), content, 0o777))
				assert.NoError(t, os.WriteFile(fmt.Sprintf("%s/file2.feature", tmpDir), append([]byte("whatever"), content...), 0o777))
				assert.NoError(t, os.WriteFile(fmt.Sprintf("%s/test1/file3.feature", tmpDir), content, 0o777))
				assert.NoError(t, os.WriteFile(fmt.Sprintf("%s/test1/file4.feature", tmpDir), content, 0o777))
				assert.NoError(t, os.WriteFile(fmt.Sprintf("%s/test1/file5.feature", tmpDir), append([]byte("whatever"), content...), 0o777))
			},
			func(errs []error) {
				assert.Len(t, errs, 2)

				msgs := []string{
					fmt.Sprintf("an error occurred with file \"%s/file2.feature\" : failed to parse gherkin: Parser errors:\n(1:1): expected: #EOF, #Language, #TagLine, #FeatureLine, #Comment, #Empty, got 'whateverFeature: test'", tmpDir),
					fmt.Sprintf("an error occurred with file \"%s/test1/file5.feature\" : failed to parse gherkin: Parser errors:\n(1:1): expected: #EOF, #Language, #TagLine, #FeatureLine, #Comment, #Empty, got 'whateverFeature: test'", tmpDir),
				}

				for _, e := range errs {
					var match bool
					for _, msg := range msgs {
						if msg == e.Error() {
							match = true
						}
					}

					if !match {
						assert.Fail(t, "Must fail with 2 files when formatting folder")
					}
				}
			},
		},
		{
			"Check a folder and set various extensions for feature files",
			fmt.Sprintf("%s/", tmpDir),
			[]string{"txt", "feat"},
			func() {
				content := []byte(`Feature: test
   test

Scenario:   scenario
   Given           whatever
   Then      whatever
"""
hello world
"""
`)

				entries, _ := os.ReadDir(tmpDir)
				for _, e := range entries {
					assert.NoError(t, os.RemoveAll(fmt.Sprintf("%s/%s", tmpDir, e.Name())))
				}

				assert.NoError(t, os.WriteFile(fmt.Sprintf("%s/file1.feature", tmpDir), content, 0o777))
				assert.NoError(t, os.WriteFile(fmt.Sprintf("%s/file2.txt", tmpDir), content, 0o777))
				assert.NoError(t, os.WriteFile(fmt.Sprintf("%s/file3.feat", tmpDir), content, 0o777))
			},
			func(errs []error) {
				assert.Len(t, errs, 2)

				errors := map[string]bool{
					fmt.Sprintf("an error occurred with file \"%s/file2.txt\" : file is not properly formatted", tmpDir):  true,
					fmt.Sprintf("an error occurred with file \"%s/file3.feat\" : file is not properly formatted", tmpDir): true,
				}

				for _, err := range errs {
					if _, ok := errors[err.Error()]; !ok {
						assert.Fail(t, fmt.Sprintf("error %s doesn't exists", err.Error()))
					}
				}
			},
		},
		{
			"Check folder with no feature files",
			tmpDir,
			[]string{"feature"},
			func() {
				entries, _ := os.ReadDir(tmpDir)
				for _, e := range entries {
					assert.NoError(t, os.RemoveAll(fmt.Sprintf("%s/%s", tmpDir, e.Name())))
				}

				assert.NoError(t, os.WriteFile(fmt.Sprintf("%s/file1.txt", tmpDir), []byte("file1"), 0o777))
				assert.NoError(t, os.WriteFile(fmt.Sprintf("%s/file2.txt", tmpDir), []byte("file2"), 0o777))
			},
			func(errs []error) {
				assert.Empty(t, errs)
			},
		},
		{
			"Check a file with different extension and an error",
			"fixtures/file.txt",
			[]string{"txt"},
			func() {},
			func(errs []error) {
				assert.Len(t, errs, 1)
				assert.EqualError(
					t, errs[0],
					"failed to parse gherkin: Parser errors:\n(1:1): expected: #EOF, #Language, #TagLine, #FeatureLine, #Comment, #Empty, got 'whatever'",
				)
			},
		},
		{
			"Check an unexisting folder",
			"whatever/whatever",
			[]string{"feature"},
			func() {},
			func(errs []error) {
				assert.Len(t, errs, 1)
				assert.EqualError(t, errs[0], "stat whatever/whatever: no such file or directory")
			},
		},
		{
			"Check an invalid file",
			"fixtures/invalid.feature",
			[]string{"feature"},
			func() {},
			func(errs []error) {
				assert.Len(t, errs, 1)
			},
		},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.testName, func(_ *testing.T) {
			scenario.setup()

			f := ghokin.NewFileManager(
				2,
				map[string]string{
					"seq": "seq 1 3",
				},
			)

			scenario.test(f.Check(t.Context(), scenario.path, scenario.extensions))
		})
	}
}
