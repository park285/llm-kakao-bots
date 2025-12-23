package guard

import "testing"

func TestIsJamoOnly(t *testing.T) {
	jamo := string([]rune{0x1100, 0x1161})
	if !isJamoOnly(jamo) {
		t.Fatalf("expected jamo-only to be true")
	}
	if isJamoOnly(jamo + "a") {
		t.Fatalf("expected mixed text to be false")
	}
	if isJamoOnly("123") {
		t.Fatalf("expected digits-only to be false")
	}
}

func TestContainsEmoji(t *testing.T) {
	if !containsEmoji("hello \U0001F600") {
		t.Fatalf("expected emoji detection")
	}
	if containsEmoji("hello") {
		t.Fatalf("did not expect emoji detection")
	}
}

func TestNormalizeText(t *testing.T) {
	input := "a\u200bb"
	output := normalizeText(input)
	if output != "ab" {
		t.Fatalf("unexpected normalized output: %q", output)
	}
}
