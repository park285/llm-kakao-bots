package mq

// GameCommand: 게임 명령어의 공통 인터페이스입니다.
// 게임별 Command 구조체가 이 인터페이스를 구현합니다.
type GameCommand interface {
	// WaitingMessageKey: 명령어 처리 중 사용자에게 보여줄 '대기 중' 메시지 키를 반환합니다.
	// nil을 반환하면 별도의 대기 메시지를 보내지 않습니다.
	WaitingMessageKey() *string

	// RequiresLock: 이 명령어 실행에 분산 락이 필요한지 여부를 반환합니다.
	// 단순 조회나 도움말 등은 락이 필요 없습니다.
	RequiresLock() bool
}
