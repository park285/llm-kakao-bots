package guard

import (
	"strings"
	"unicode"

	"github.com/mtibben/confusables"
	"golang.org/x/text/unicode/norm"
)

func normalizeText(text string) string {
	// 1. Homoglyph 정규화 (Visual Skeleton 추출)
	// 예: "Sеcret" (Cyrillic e) -> "Secret"
	skeleton := confusables.Skeleton(text)

	// 2. NFKC 정규화
	normalized := norm.NFKC.String(skeleton)

	// 3. 제어 문자 제거
	return stripControlChars(normalized)
}

func stripControlChars(text string) string {
	var builder strings.Builder
	builder.Grow(len(text))
	for _, r := range text {
		if unicode.Is(unicode.Cf, r) || unicode.Is(unicode.Cc, r) {
			continue
		}
		builder.WriteRune(r)
	}
	return builder.String()
}

func containsEmoji(text string) bool {
	for _, r := range text {
		if r == 0x200D {
			return true
		}
		if isEmojiRune(r) {
			return true
		}
	}
	return false
}

func isEmojiRune(r rune) bool {
	for _, emojiRange := range emojiRanges {
		if r >= emojiRange.start && r <= emojiRange.end {
			return true
		}
	}
	return false
}

func isJamoOnly(text string) bool {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return false
	}

	hasJamo := false
	for _, r := range trimmed {
		if isJamoRune(r) {
			hasJamo = true
			continue
		}
		if unicode.IsSpace(r) || unicode.IsDigit(r) || unicode.IsPunct(r) {
			continue
		}
		return false
	}
	return hasJamo
}

func isJamoRune(r rune) bool {
	for _, block := range jamoRanges {
		if r >= block.start && r <= block.end {
			return true
		}
	}
	return false
}

type runeRange struct {
	start rune
	end   rune
}

var jamoRanges = []runeRange{
	{start: 0x1100, end: 0x11FF},
	{start: 0x3130, end: 0x318F},
	{start: 0xA960, end: 0xA97F},
	{start: 0xD7B0, end: 0xD7FF},
}

var emojiRanges = []runeRange{
	{start: 0x1F600, end: 0x1F64F},
	{start: 0x1F300, end: 0x1F5FF},
	{start: 0x1F680, end: 0x1F6FF},
	{start: 0x1F1E0, end: 0x1F1FF},
	{start: 0x2600, end: 0x26FF},
	{start: 0x2700, end: 0x27BF},
	{start: 0xFE00, end: 0xFE0F},
	{start: 0x1F900, end: 0x1F9FF},
	{start: 0x1FA00, end: 0x1FA6F},
	{start: 0x1FA70, end: 0x1FAFF},
}
