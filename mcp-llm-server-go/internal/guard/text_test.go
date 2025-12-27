package guard

import "testing"

func TestNormalizeText(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Normal text",
			input:    "Hello World",
			expected: "Hello World",
		},
		{
			name:     "Cyrillic Homoglyph (Sеcret)",
			input:    "Sеcret", // Cyrillic 'е' (U+0435)
			expected: "Secret", // Latin 'e'
		},
		{
			name:     "Fullwidth (Ｈｅｌｌｏ)",
			input:    "Ｈｅｌｌｏ",
			expected: "Hello",
		},
		{
			name:     "Control chars",
			input:    "Hello\u200BWorld", // Zero width space
			expected: "HelloWorld",
		},
		{
			name:     "Mixed Homoglyph + Fullwidth + Control",
			input:    "Ｓ\u0435cret\u200B", // Fullwidth S, Cyrillic e, Zero width
			expected: "Secret",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := normalizeText(tt.input)
			if got != tt.expected {
				t.Errorf("normalizeText(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}
