package llmrest

// Ptr: 값의 포인터를 반환하는 제네릭 헬퍼 함수입니다.
// Go 문법상 리터럴(예: &int(5))의 주소를 직접 취할 수 없는 제약을 해결합니다.
//
// 사용 예:
//
//	req := TurtleSoupPuzzleGenerationRequest{
//	    Difficulty: Ptr(5),
//	}
func Ptr[T any](v T) *T {
	return &v
}
