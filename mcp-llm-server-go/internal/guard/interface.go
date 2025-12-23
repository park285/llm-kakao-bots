package guard

// Guard 는 입력 검증 인터페이스다.
// 테스트에서 mock 구현을 주입할 수 있도록 한다.
type Guard interface {
	// Evaluate 입력 문자열 평가
	Evaluate(input string) Evaluation

	// EnsureSafe 위험 입력을 에러로 반환
	EnsureSafe(input string) error

	// IsMalicious 입력이 위험한지 여부
	IsMalicious(input string) bool
}

// InjectionGuard가 Guard 인터페이스를 구현하는지 컴파일 타임 확인
var _ Guard = (*InjectionGuard)(nil)
