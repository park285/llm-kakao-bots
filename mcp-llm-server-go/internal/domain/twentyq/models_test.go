package twentyq

import "testing"

func TestParseAnswerScale(t *testing.T) {
	input := "prefix " + string(AnswerYes)
	value, ok := ParseAnswerScale(input)
	if !ok || value != AnswerYes {
		t.Fatalf("unexpected parse result")
	}
}

func TestResultNameMappings(t *testing.T) {
	if value, ok := VerifyResultName(string(VerifyAccept)); !ok || value != "ACCEPT" {
		t.Fatalf("unexpected verify result")
	}
	if value, ok := SynonymResultName(string(SynonymEquivalent)); !ok || value != "EQUIVALENT" {
		t.Fatalf("unexpected synonym result")
	}
}

func TestSchemas(t *testing.T) {
	schema := HintsSchema()
	required, ok := schema["required"].([]string)
	if !ok || len(required) == 0 {
		t.Fatalf("expected required fields")
	}
}
