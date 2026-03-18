package json_test

import (
	"testing"

	jsonct "github.com/PaddleHQ/ghokin/v4/ghokin/internal/contenttypes/json"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFormatter_Format(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
		wantErr  bool
	}{
		{
			name:     "compact json is pretty printed",
			input:    `{"name":"test","value":1}`,
			expected: "{\n  \"name\": \"test\",\n  \"value\": 1\n}",
		},
		{
			name:     "already formatted json is unchanged",
			input:    "{\n  \"name\": \"test\"\n}",
			expected: "{\n  \"name\": \"test\"\n}",
		},
		{
			name:     "json array",
			input:    `[1,2,3]`,
			expected: "[\n  1,\n  2,\n  3\n]",
		},
		{
			name:    "invalid json returns error",
			input:   `{invalid}`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := jsonct.Formatter{}
			result, err := f.Format(tt.input)

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}
