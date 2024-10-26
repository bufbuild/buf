package buflsp

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_formatComment(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Single-line comment",
			input:    "// This is a single-line comment",
			expected: " This is a single-line comment",
		},
		{
			name:     "Multi-line comment",
			input:    "/*\n This is a\n multi-line comment\n */",
			expected: "This is a\nmulti-line comment",
		},
		{
			name:     "Multi-line comment with mixed indentation",
			input:    "/*\n  * First line\n  * - Second line\n  *   - Third line\n */",
			expected: "First line\n- Second line\n  - Third line",
		},
		{
			name:     "Multi-line comment with JavaDoc convention",
			input:    "/** This is a\n   * multi-line comment\n   * with multi-asterisks */",
			expected: "This is a\nmulti-line comment\nwith multi-asterisks",
		},
		{
			name:     "Single-line multi-line comment",
			input:    "/* Single-line multi-line comment */",
			expected: "Single-line multi-line comment",
		},
		{
			name:     "Empty comment",
			input:    "/**/",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := formatComment(tt.input)
			if result != tt.expected {
				assert.Equal(t, tt.input, result)
			}
		})
	}
}
