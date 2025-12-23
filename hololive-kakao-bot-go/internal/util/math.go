package util

import "fmt"

// Max 는 동작을 수행한다.
func Max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// Min 는 동작을 수행한다.
func Min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// Unique 는 동작을 수행한다.
func Unique(nums []int) []int {
	seen := make(map[int]struct{})
	result := make([]int, 0, len(nums))

	for _, n := range nums {
		if _, exists := seen[n]; !exists {
			seen[n] = struct{}{}
			result = append(result, n)
		}
	}

	return result
}

// FormatKoreanNumber 는 한국어 단위(만)로 숫자를 포맷팅한다.
// 예: 10000 -> "1만", 12345 -> "1만 2345", 500 -> "500"
func FormatKoreanNumber(n int64) string {
	if n >= 10000 {
		man := n / 10000
		remainder := n % 10000
		if remainder == 0 {
			return fmt.Sprintf("%d만", man)
		}
		return fmt.Sprintf("%d만 %d", man, remainder)
	}
	return fmt.Sprintf("%d", n)
}
