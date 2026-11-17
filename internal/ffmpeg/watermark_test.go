package ffmpeg

import (
	"strings"
	"testing"
)

func TestSanitizeWatermarkText(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "plain text",
			input:    "My Watermark",
			expected: "My Watermark",
		},
		{
			name:     "colons and single quotes",
			input:    "Movie: 'Title'",
			expected: `Movie\: \'Title\'`,
		},
		{
			name:     "percentages and commas",
			input:    "100%, 4K",
			expected: `100\%\, 4K`,
		},
		{
			name:     "brackets and backslashes",
			input:    `[test]\path`,
			expected: `\[test\]\\path`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SanitizeWatermarkText(tt.input)
			if got != tt.expected {
				t.Errorf("SanitizeWatermarkText(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestBuildWatermarkFilter(t *testing.T) {
	t.Run("empty text returns scale only", func(t *testing.T) {
		filter := BuildWatermarkFilter("", "white", 0, 1, 1920, 1080)
		if filter != "scale=1280:-1" {
			t.Errorf("expected scale=1280:-1, got %s", filter)
		}
	})

	t.Run("custom text formats properly", func(t *testing.T) {
		filter := BuildWatermarkFilter("Hello", "white", 4, 1, 1920, 1080)
		if !strings.Contains(filter, "text='Hello'") {
			t.Errorf("expected text='Hello' in filter, got %s", filter)
		}
		if !strings.Contains(filter, "fontcolor=white") {
			t.Errorf("expected fontcolor=white in filter, got %s", filter)
		}
		if !strings.Contains(filter, "scale=1280:-1") {
			t.Errorf("expected scale=1280:-1 in filter, got %s", filter)
		}
	})
}
