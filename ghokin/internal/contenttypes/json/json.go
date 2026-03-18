// Package json provides a Formatter for JSON doc string content.
package json

import (
	"bytes"
	"encoding/json"
	"fmt"
)

// Formatter formats JSON content with standard indentation.
type Formatter struct{}

// NewFormatter creates a new instance of the JSON Formatter.
func NewFormatter() *Formatter {
	return &Formatter{}
}

// SupportedTypes returns the list of content types supported by this formatter.
func (f *Formatter) SupportedTypes() []string {
	return []string{"json"}
}

// Format pretty-prints JSON content with 2-space indentation.
func (f *Formatter) Format(content string) (string, error) {
	var buf bytes.Buffer
	if err := json.Indent(&buf, []byte(content), "", "  "); err != nil {
		return "", fmt.Errorf("failed to format json: %w", err)
	}

	return buf.String(), nil
}
