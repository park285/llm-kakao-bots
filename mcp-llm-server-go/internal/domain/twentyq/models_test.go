package twentyq

import "testing"

func TestParseAnswerScale(t *testing.T) {
	tests := []struct {
		input    string
		expected AnswerScale
		ok       bool
	}{
		// 정확한 매칭
		{"예", AnswerYes, true},
		{"아마도 예", AnswerProbablyYes, true},
		{"아마도 아니오", AnswerProbablyNo, true},
		{"아니오", AnswerNo, true},
		{"정책 위반", AnswerPolicyViolation, true},
		// prefix 포함 케이스
		{"prefix 예", AnswerYes, true},
		{"prefix 아마도 예", AnswerProbablyYes, true},
		{"prefix 아마도 아니오", AnswerProbablyNo, true},
		// 공백 처리
		{"  아마도 예  ", AnswerProbablyYes, true},
		// 매칭 실패
		{"unknown", "", false},
		{"", "", false},
	}

	for _, tc := range tests {
		value, ok := ParseAnswerScale(tc.input)
		if ok != tc.ok || value != tc.expected {
			t.Errorf("ParseAnswerScale(%q) = (%q, %v), want (%q, %v)",
				tc.input, value, ok, tc.expected, tc.ok)
		}
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
