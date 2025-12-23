package textutil

import (
	"strings"
	"testing"
	"unicode/utf8"
)

func TestChunkByLines_Empty(t *testing.T) {
	result := ChunkByLines("", 100)
	if len(result) != 0 {
		t.Errorf("expected 0 chunks for empty input, got %d", len(result))
	}
}

func TestChunkByLines_SingleLine(t *testing.T) {
	result := ChunkByLines("hello world", 100)
	if len(result) != 1 {
		t.Fatalf("expected 1 chunk, got %d", len(result))
	}
	if result[0] != "hello world" {
		t.Errorf("expected 'hello world', got '%s'", result[0])
	}
}

func TestChunkByLines_MultipleLinesFitInOne(t *testing.T) {
	input := "line1\nline2\nline3"
	result := ChunkByLines(input, 100)
	if len(result) != 1 {
		t.Fatalf("expected 1 chunk, got %d", len(result))
	}
	if result[0] != input {
		t.Errorf("expected '%s', got '%s'", input, result[0])
	}
}

func TestChunkByLines_MultipleChunks(t *testing.T) {
	input := "aaa\nbbb\nccc\nddd"
	// "aaa\nbbb"의 길이 = 7, "ccc\nddd"의 길이 = 7
	result := ChunkByLines(input, 7)
	if len(result) != 2 {
		t.Fatalf("expected 2 chunks, got %d", len(result))
	}
	if result[0] != "aaa\nbbb" {
		t.Errorf("expected 'aaa\\nbbb', got '%s'", result[0])
	}
	if result[1] != "ccc\nddd" {
		t.Errorf("expected 'ccc\\nddd', got '%s'", result[1])
	}
}

func TestChunkByLines_LineTruncation(t *testing.T) {
	// 개별 라인이 maxLength보다 긴 경우 자름
	input := "verylongline"
	result := ChunkByLines(input, 4)
	if len(result) != 1 {
		t.Fatalf("expected 1 chunk, got %d", len(result))
	}
	if result[0] != "very" {
		t.Errorf("expected 'very', got '%s'", result[0])
	}
}

func TestChunkByLines_ZeroMaxLength(t *testing.T) {
	input := "hello"
	result := ChunkByLines(input, 0)
	if len(result) != 1 {
		t.Fatalf("expected 1 chunk, got %d", len(result))
	}
	if result[0] != input {
		t.Errorf("expected '%s', got '%s'", input, result[0])
	}
}

func TestChunkByLines_NegativeMaxLength(t *testing.T) {
	input := "hello"
	result := ChunkByLines(input, -10)
	if len(result) != 1 {
		t.Fatalf("expected 1 chunk, got %d", len(result))
	}
	if result[0] != input {
		t.Errorf("expected '%s', got '%s'", input, result[0])
	}
}

func TestChunkByLines_ExactFit(t *testing.T) {
	// "ab\ncd"의 길이 = 5
	input := "ab\ncd"
	result := ChunkByLines(input, 5)
	if len(result) != 1 {
		t.Fatalf("expected 1 chunk, got %d", len(result))
	}
	if result[0] != input {
		t.Errorf("expected '%s', got '%s'", input, result[0])
	}
}

func TestChunkByLines_OffByOne(t *testing.T) {
	// 정확히 맞지 않는 경우 분리
	input := "ab\ncd\nef"
	// maxLength=5: "ab\ncd"(5) → 분리 → "ef"(2)
	result := ChunkByLines(input, 5)
	if len(result) != 2 {
		t.Fatalf("expected 2 chunks, got %d", len(result))
	}
	if result[0] != "ab\ncd" {
		t.Errorf("expected 'ab\\ncd', got '%s'", result[0])
	}
	if result[1] != "ef" {
		t.Errorf("expected 'ef', got '%s'", result[1])
	}
}

func TestChunkByLines_OnlyNewlines(t *testing.T) {
	input := "\n\n\n"
	result := ChunkByLines(input, 10)
	// 빈 라인만 있으면 결과도 비어야 함
	if len(result) != 0 {
		t.Errorf("expected 0 chunks for newlines-only input, got %d: %v", len(result), result)
	}
}

func TestChunkByLines_KoreanContent(t *testing.T) {
	input := "가나다\n라마바"
	result := ChunkByLines(input, 20)
	if len(result) != 1 {
		t.Fatalf("expected 1 chunk, got %d", len(result))
	}
	if result[0] != input {
		t.Errorf("expected '%s', got '%s'", input, result[0])
	}
}

func TestChunkByLines_KoreanNotSplitByBytes(t *testing.T) {
	// UTF-8 바이트 길이 기준으로는 500을 넘지만, 문자 수 기준으로는 500을 넘지 않는 경우.
	input := strings.Repeat("가", 200) + "\n" + strings.Repeat("나", 200) // 401 chars
	result := ChunkByLines(input, 500)
	if len(result) != 1 {
		t.Fatalf("expected 1 chunk, got %d", len(result))
	}
	if result[0] != input {
		t.Errorf("expected chunk equals input, got '%s'", result[0])
	}
}

func TestChunkByLines_TruncationKeepsValidUTF8(t *testing.T) {
	input := strings.Repeat("가", 600)
	result := ChunkByLines(input, 500)
	if len(result) != 1 {
		t.Fatalf("expected 1 chunk, got %d", len(result))
	}
	if !utf8.ValidString(result[0]) {
		t.Fatalf("expected valid UTF-8 string")
	}
	if utf8.RuneCountInString(result[0]) != 500 {
		t.Fatalf("expected 500 runes, got %d", utf8.RuneCountInString(result[0]))
	}
}

func TestChunkByLines_LargeInput(t *testing.T) {
	// 100줄의 입력, 각 줄 = "line N"
	var builder strings.Builder
	for i := 0; i < 100; i++ {
		if i > 0 {
			builder.WriteByte('\n')
		}
		builder.WriteString("line")
	}
	input := builder.String()
	result := ChunkByLines(input, 50)

	// 최소 2개 이상의 청크가 있어야 함
	if len(result) < 2 {
		t.Errorf("expected at least 2 chunks for large input, got %d", len(result))
	}

	// 각 청크가 maxLength를 초과하지 않아야 함
	for i, chunk := range result {
		if utf8.RuneCountInString(chunk) > 50 {
			t.Errorf("chunk %d exceeds maxLength: %d > 50", i, utf8.RuneCountInString(chunk))
		}
	}
}
