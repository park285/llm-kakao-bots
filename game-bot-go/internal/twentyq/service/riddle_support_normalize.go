package service

import (
	"regexp"
	"strings"

	"golang.org/x/text/unicode/norm"
)

var explicitAnswerPattern = regexp.MustCompile(`(?i)^정답\s+(.+?)(?:인가요|입니까|이에요|이야)?\s*$`)

func matchExplicitAnswer(text string) (string, bool) {
	m := explicitAnswerPattern.FindStringSubmatch(strings.TrimSpace(text))
	if len(m) < 2 {
		return "", false
	}
	guess := strings.TrimSpace(m[1])
	if guess == "" {
		return "", false
	}
	return guess, true
}

var koreanEndingsPattern = regexp.MustCompile(`(?i)(?:야|이야|예요|이에요|입니까|인가요|니|죠|지|거야|거니|거죠|거지)\s*\??\s*$`)

var whitespacePunctPattern = regexp.MustCompile(`[\p{Z}\s\p{Punct}]`)

func normalizeForEquality(text string) string {
	normalized := norm.NFKC.String(text)
	normalized = strings.ToLower(normalized)
	normalized = koreanEndingsPattern.ReplaceAllString(normalized, "")
	normalized = strings.TrimSpace(normalized)
	normalized = whitespacePunctPattern.ReplaceAllString(normalized, "")
	return normalized
}
