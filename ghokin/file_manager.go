// Package ghokin provides formatting and transformation for Gherkin feature files.
package ghokin

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	mpath "path"
	"path/filepath"
	"sync"

	"github.com/PaddleHQ/ghokin/v4/ghokin/internal/transformer"

	"github.com/saintfish/chardet"
	"golang.org/x/net/html/charset"
)

// sentinel errors for file operations.
var (
	errReadFile      = fmt.Errorf("failed to read file")
	errDetectCharset = fmt.Errorf("failed to detect charset")
	errNewReader     = fmt.Errorf("failed to create charset reader")
	errReadContent   = fmt.Errorf("failed to read content")
	errWalkPath      = fmt.Errorf("failed to walk path")
)

// ProcessFileError is emitted when processing a file trigger an error.
type ProcessFileError struct {
	Message string
	File    string
}

// Error dumps a string error.
func (p ProcessFileError) Error() string {
	return fmt.Sprintf("an error occurred with file %q : %s", p.File, p.Message)
}

type aliases map[string]string

// FileManager handles transformation on feature files.
type FileManager struct {
	indent  int
	aliases aliases
}

// NewFileManager creates a brand new FileManager, it requires indentation values and aliases defined
// as a shell commands in comments.
func NewFileManager(indent int, aliases map[string]string) FileManager {
	return FileManager{
		indent,
		aliases,
	}
}

// Transform formats and applies shell commands on feature file.
func (f FileManager) Transform(ctx context.Context, filename string) ([]byte, error) {
	content, err := os.ReadFile(filepath.Clean(filename))
	if err != nil {
		return []byte{}, fmt.Errorf("%w: %w", errReadFile, err)
	}
	detector := chardet.NewTextDetector()
	result, err := detector.DetectBest(content)
	if err != nil {
		return []byte{}, fmt.Errorf("%w: %w", errDetectCharset, err)
	}
	if result.Charset != "UTF-8" {
		r, err := charset.NewReaderLabel(result.Charset, bytes.NewBuffer(content))
		if err != nil {
			return []byte{}, fmt.Errorf("%w: %w", errNewReader, err)
		}
		content, err = io.ReadAll(r)
		if err != nil {
			return []byte{}, fmt.Errorf("%w: %w", errReadContent, err)
		}
	}
	contentTransformer := &transformer.ContentTransformer{}
	contentTransformer.DetectSettings(content)
	content = contentTransformer.Prepare(content)
	section, err := extractSections(content)
	if err != nil {
		return []byte{}, err
	}
	content, err = transform(ctx, section, f.indent, f.aliases)
	if err != nil {
		return []byte{}, err
	}
	return contentTransformer.Restore(content), nil
}

// TransformAndReplace formats and applies shell commands on file or folder
// and replace the content of files.
func (f FileManager) TransformAndReplace(ctx context.Context, path string, extensions []string) []error {
	return f.process(ctx, path, extensions, replaceFileWithContent)
}

// Check ensures file or folder is well formatted.
func (f FileManager) Check(ctx context.Context, path string, extensions []string) []error {
	return f.process(ctx, path, extensions, check)
}

func (f FileManager) process(
	ctx context.Context,
	path string,
	extensions []string,
	processFile func(file string, content []byte) error,
) []error {
	errors := []error{}
	fi, err := os.Stat(path)
	if err != nil {
		return append(errors, err)
	}

	switch mode := fi.Mode(); {
	case mode.IsDir():
		errors = append(errors, f.processPath(ctx, path, extensions, processFile)...)
	case mode.IsRegular():
		b, err := f.Transform(ctx, path)
		if err != nil {
			return append(errors, err)
		}
		if err := processFile(path, b); err != nil {
			errors = append(errors, err)
		}
	}
	return errors
}

func (f FileManager) processPath(
	ctx context.Context,
	path string,
	extensions []string,
	processFile func(file string, content []byte) error,
) []error {
	errors := []error{}
	fc := make(chan string)
	wg := sync.WaitGroup{}
	var mu sync.Mutex

	files, err := findFeatureFiles(path, extensions)
	if err != nil {
		return []error{err}
	}
	if len(files) == 0 {
		return []error{}
	}

	for range 10 {
		wg.Go(func() {
			for file := range fc {
				b, err := f.Transform(ctx, file)
				if err != nil {
					mu.Lock()
					errors = append(errors, ProcessFileError{Message: err.Error(), File: file})
					mu.Unlock()
					continue
				}
				if err := processFile(file, b); err != nil {
					mu.Lock()
					errors = append(errors, err)
					mu.Unlock()
				}
			}
		})
	}

	for _, file := range files {
		fc <- file
	}

	close(fc)
	wg.Wait()

	return errors
}

func replaceFileWithContent(file string, content []byte) error {
	f, err := os.Create(filepath.Clean(file))
	if err != nil {
		return ProcessFileError{Message: err.Error(), File: file}
	}
	_, err = f.Write(content)
	if err != nil {
		return ProcessFileError{Message: err.Error(), File: file}
	}
	return nil
}

func check(file string, content []byte) error {
	currentContent, err := os.ReadFile(file) // #nosec
	if err != nil {
		return ProcessFileError{Message: err.Error(), File: file}
	}

	if !bytes.Equal(currentContent, content) {
		return ProcessFileError{Message: "file is not properly formatted", File: file}
	}

	return nil
}

func findFeatureFiles(rootPath string, extensions []string) ([]string, error) {
	files := []string{}

	if err := filepath.Walk(rootPath, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		for _, extension := range extensions {
			if !info.IsDir() && mpath.Ext(p) == "."+extension {
				files = append(files, p)
				break
			}
		}

		return nil
	}); err != nil {
		return []string{}, fmt.Errorf("%w: %w", errWalkPath, err)
	}

	return files, nil
}
