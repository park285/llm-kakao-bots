package toon

import (
	"strings"
	"testing"
)

func TestEncodeMapOrder(t *testing.T) {
	value := map[string]any{"b": 1, "a": 2}
	got := Encode(value)
	if got != "a: 2\nb: 1" {
		t.Fatalf("unexpected encoding: %s", got)
	}
}

func TestEncodeStringEscapes(t *testing.T) {
	got := Encode("a,b")
	if got != "\"a,b\"" {
		t.Fatalf("unexpected encoding: %s", got)
	}
}

func TestEncodeSecret(t *testing.T) {
	got := EncodeSecret("target", "category", map[string]any{"foo": "bar"})
	if !strings.Contains(got, "target:") || !strings.Contains(got, "details:") {
		t.Fatalf("unexpected secret encoding: %s", got)
	}
}
