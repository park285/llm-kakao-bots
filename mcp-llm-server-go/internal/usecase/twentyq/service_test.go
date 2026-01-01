package twentyq

import (
	"testing"
)

func TestNormalizeForCompare(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		// 기본 케이스
		{
			name:     "exact_match",
			input:    "사과",
			expected: "사과",
		},
		{
			name:     "with_spaces",
			input:    "사 과",
			expected: "사과",
		},
		{
			name:     "leading_trailing_spaces",
			input:    "  사과  ",
			expected: "사과",
		},
		{
			name:     "mixed_spaces_tabs",
			input:    "사\t과",
			expected: "사과",
		},
		{
			name:     "uppercase_english",
			input:    "Apple",
			expected: "apple",
		},
		{
			name:     "mixed_case_korean_english",
			input:    "아이 Phone",
			expected: "아이phone",
		},
		{
			name:     "empty_string",
			input:    "",
			expected: "",
		},
		{
			name:     "spaces_only",
			input:    "   ",
			expected: "",
		},
		// 확장된 케이스
		{
			name:     "multiple_spaces",
			input:    "스 마 트 폰",
			expected: "스마트폰",
		},
		{
			name:     "mixed_whitespace",
			input:    " \t iPhone \t ",
			expected: "iphone",
		},
		{
			name:     "numbers",
			input:    "아이폰 15",
			expected: "아이폰15",
		},
		{
			name:     "special_characters_preserved",
			input:    "C++",
			expected: "c++",
		},
		{
			name:     "korean_with_numbers",
			input:    "갤럭시 S24",
			expected: "갤럭시s24",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := normalizeForCompare(tc.input)
			if result != tc.expected {
				t.Fatalf("normalizeForCompare(%q) = %q, expected %q", tc.input, result, tc.expected)
			}
		})
	}
}

func TestNormalizeForCompare_Equality(t *testing.T) {
	// Fast-path에서 사용되는 실제 비교 시나리오
	tests := []struct {
		name     string
		target   string
		guess    string
		expected bool
	}{
		// 정확 일치 케이스 (Fast-path 적용)
		{
			name:     "exact_match",
			target:   "사과",
			guess:    "사과",
			expected: true,
		},
		{
			name:     "case_insensitive",
			target:   "Apple",
			guess:    "apple",
			expected: true,
		},
		{
			name:     "space_difference",
			target:   "핸드폰",
			guess:    "핸드 폰",
			expected: true,
		},
		{
			name:     "mixed_korean_english",
			target:   "iPhone",
			guess:    "iphone",
			expected: true,
		},
		{
			name:     "multiple_space_difference",
			target:   "스마트폰",
			guess:    "스 마 트 폰",
			expected: true,
		},
		{
			name:     "leading_trailing_space",
			target:   "노트북",
			guess:    "  노트북  ",
			expected: true,
		},
		{
			name:     "tab_vs_space",
			target:   "컴퓨터",
			guess:    "컴\t퓨터",
			expected: true,
		},
		// 불일치 케이스 (LLM 호출 필요)
		{
			name:     "different_words",
			target:   "사과",
			guess:    "배",
			expected: false,
		},
		{
			name:     "synonym_not_detected",
			target:   "휴대폰",
			guess:    "핸드폰",
			expected: false, // 동의어는 Fast-path에서 감지 안 됨 → LLM 호출 필요
		},
		{
			name:     "hypernym_not_detected",
			target:   "진돗개",
			guess:    "개",
			expected: false, // 상위어는 Fast-path에서 감지 안 됨 → LLM 호출 필요
		},
		{
			name:     "hyponym_not_detected",
			target:   "개",
			guess:    "진돗개",
			expected: false, // 하위어는 Fast-path에서 감지 안 됨 → LLM 호출 필요
		},
		{
			name:     "similar_but_different",
			target:   "사과",
			guess:    "사과나무",
			expected: false,
		},
		{
			name:     "partial_match",
			target:   "스마트폰",
			guess:    "스마트",
			expected: false,
		},
		// 숫자 포함 케이스
		{
			name:     "with_numbers_match",
			target:   "아이폰15",
			guess:    "아이폰 15",
			expected: true,
		},
		{
			name:     "different_numbers",
			target:   "아이폰15",
			guess:    "아이폰16",
			expected: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := normalizeForCompare(tc.target) == normalizeForCompare(tc.guess)
			if result != tc.expected {
				t.Fatalf("normalizeForCompare(%q) == normalizeForCompare(%q) = %v, expected %v",
					tc.target, tc.guess, result, tc.expected)
			}
		})
	}
}

func TestNormalizeForCompare_RealGameScenarios(t *testing.T) {
	// 실제 게임에서 발생할 수 있는 시나리오
	tests := []struct {
		name        string
		target      string
		guess       string
		shouldMatch bool
		description string
	}{
		// Fast-path 적용되어야 하는 케이스 (비용 절감)
		{
			name:        "exact_food",
			target:      "김치찌개",
			guess:       "김치찌개",
			shouldMatch: true,
			description: "정확히 같은 음식 이름",
		},
		{
			name:        "space_in_food",
			target:      "김치찌개",
			guess:       "김치 찌개",
			shouldMatch: true,
			description: "공백 차이만 있는 음식 이름",
		},
		{
			name:        "brand_case",
			target:      "MacBook",
			guess:       "macbook",
			shouldMatch: true,
			description: "브랜드명 대소문자 차이",
		},
		{
			name:        "korean_brand",
			target:      "삼성갤럭시",
			guess:       "삼성 갤럭시",
			shouldMatch: true,
			description: "한글 브랜드명 공백 차이",
		},
		// Fast-path 적용 안 됨 (LLM 필요)
		{
			name:        "synonym_phone",
			target:      "스마트폰",
			guess:       "핸드폰",
			shouldMatch: false,
			description: "동의어 - LLM 판단 필요",
		},
		{
			name:        "category_vs_specific",
			target:      "자동차",
			guess:       "테슬라",
			shouldMatch: false,
			description: "카테고리 vs 특정 브랜드 - LLM 판단 필요",
		},
		{
			name:        "ingredient_vs_product",
			target:      "쌀",
			guess:       "밥",
			shouldMatch: false,
			description: "재료 vs 제품 - LLM 판단 필요",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := normalizeForCompare(tc.target) == normalizeForCompare(tc.guess)
			if result != tc.shouldMatch {
				t.Fatalf("[%s] normalizeForCompare(%q) == normalizeForCompare(%q) = %v, expected %v",
					tc.description, tc.target, tc.guess, result, tc.shouldMatch)
			}
		})
	}
}
