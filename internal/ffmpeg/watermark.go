package ffmpeg

import (
	"fmt"
	"math"
	"strings"
)

// SanitizeWatermarkText escapes characters that have special meaning in FFmpeg filtergraphs
func SanitizeWatermarkText(text string) string {
	var sb strings.Builder
	for _, r := range text {
		switch r {
		case '\\':
			sb.WriteString(`\\`)
		case '\'':
			sb.WriteString(`\'`)
		case ':':
			sb.WriteString(`\:`)
		case '%':
			sb.WriteString(`\%`)
		case ',':
			sb.WriteString(`\,`)
		case '[':
			sb.WriteString(`\[`)
		case ']':
			sb.WriteString(`\]`)
		default:
			sb.WriteRune(r)
		}
	}
	return sb.String()
}

// GetWatermarkCoordinates returns FFmpeg drawtext x, y expressions for the chosen position
func GetWatermarkCoordinates(posIndex int) (string, string) {
	switch posIndex {
	case 0: // Top Left
		return "w*0.02", "h*0.02"
	case 1: // Top Center
		return "(w-text_w)/2", "h*0.02"
	case 2: // Top Right
		return "w-text_w-w*0.02", "h*0.02"
	case 3: // Center Left
		return "w*0.02", "(h-text_h)/2"
	case 4: // Centered
		return "(w-text_w)/2", "(h-text_h)/2"
	case 5: // Center Right
		return "w-text_w-w*0.02", "(h-text_h)/2"
	case 6: // Bottom Left
		return "w*0.02", "h-text_h-h*0.02"
	case 7: // Bottom Center
		return "(w-text_w)/2", "h-text_h-h*0.02"
	case 8: // Bottom Right
		return "w-text_w-w*0.02", "h-text_h-h*0.02"
	default:
		return "(w-text_w)/2", "(h-text_h)/2"
	}
}

// BuildWatermarkFilter constructs the complete FFmpeg video filter string
func BuildWatermarkFilter(text, color string, posIndex, fontSizeTier, width, height int) string {
	if strings.TrimSpace(text) == "" {
		return "scale=1280:-1"
	}

	sanitized := SanitizeWatermarkText(text)
	baseSize := 40
	switch fontSizeTier {
	case 0:
		baseSize = 30
	case 1:
		baseSize = 40
	case 2:
		baseSize = 50
	}

	diag := math.Sqrt(float64(width*width + height*height))
	calcSize := int((diag / 1388.0) * float64(baseSize))
	if calcSize < 12 {
		calcSize = 12
	}

	xPos, yPos := GetWatermarkCoordinates(posIndex)

	return fmt.Sprintf("drawtext=text='%s':fontcolor=%s:fontsize=%d:x=%s:y=%s,scale=1280:-1",
		sanitized, color, calcSize, xPos, yPos)
}
