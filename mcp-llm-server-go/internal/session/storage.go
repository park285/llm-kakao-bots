package session

import (
	"context"

	"github.com/park285/llm-kakao-bots/mcp-llm-server-go/internal/llm"
)

// Storage 는 세션 저장소 인터페이스다.
// 테스트에서 mock 구현을 주입할 수 있도록 한다.
type Storage interface {
	// IsEnabled 저장소 활성화 여부
	IsEnabled() bool

	// CreateSession 세션 생성
	CreateSession(ctx context.Context, meta Meta) error

	// GetSession 세션 조회
	GetSession(ctx context.Context, sessionID string) (*Meta, error)

	// UpdateSession 세션 업데이트
	UpdateSession(ctx context.Context, meta Meta) error

	// DeleteSession 세션 삭제
	DeleteSession(ctx context.Context, sessionID string) error

	// GetHistory 히스토리 조회
	GetHistory(ctx context.Context, sessionID string) ([]llm.HistoryEntry, error)

	// AppendHistory 히스토리 추가
	AppendHistory(ctx context.Context, sessionID string, entries ...llm.HistoryEntry) error

	// SessionCount 세션 수
	SessionCount(ctx context.Context) (int, error)

	// Ping 연결 확인
	Ping(ctx context.Context) error

	// Close 리소스 정리
	Close()
}

// Store가 Storage 인터페이스를 구현하는지 컴파일 타임 확인
var _ Storage = (*Store)(nil)
