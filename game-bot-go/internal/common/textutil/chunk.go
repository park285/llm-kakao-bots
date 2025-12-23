package textutil

import (
	"strings"
	"unicode/utf8"
)

// ChunkByLines 는 동작을 수행한다.
func ChunkByLines(input string, maxLength int) []string {
	if maxLength <= 0 {
		return []string{input}
	}

	lines := strings.Split(input, "\n")
	chunks := make([]string, 0, len(lines))

	var current strings.Builder
	currentLength := 0

	truncateRunes := func(s string, limit int) string {
		if limit <= 0 {
			return ""
		}
		if utf8.RuneCountInString(s) <= limit {
			return s
		}
		count := 0
		for i := range s {
			if count == limit {
				return s[:i]
			}
			count++
		}
		return s
	}

	flush := func() {
		if currentLength == 0 {
			return
		}
		chunks = append(chunks, current.String())
		current.Reset()
		currentLength = 0
	}

	for _, raw := range lines {
		line := raw
		if utf8.RuneCountInString(line) > maxLength {
			line = truncateRunes(line, maxLength)
		}

		separator := 0
		if currentLength > 0 {
			separator = 1
		}

		lineLength := utf8.RuneCountInString(line)
		if currentLength+separator+lineLength <= maxLength {
			if separator == 1 {
				current.WriteByte('\n')
			}
			current.WriteString(line)
			currentLength += separator + lineLength
			continue
		}

		flush()
		current.WriteString(line)
		currentLength = lineLength
	}

	flush()
	return chunks
}
