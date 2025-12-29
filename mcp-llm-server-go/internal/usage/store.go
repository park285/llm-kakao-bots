package usage

import (
	"context"
	"time"
)

// Store: 사용량 저장소 인터페이스입니다.
// 테스트에서 mock 구현을 주입할 수 있도록 합니다.
type Store interface {
	// RecordUsage 토큰 사용량 기록
	RecordUsage(
		ctx context.Context,
		inputTokens int64,
		outputTokens int64,
		reasoningTokens int64,
		requestCount int64,
		usageDate time.Time,
	) error

	// GetDailyUsage 일별 사용량 조회
	GetDailyUsage(ctx context.Context, usageDate time.Time) (*DailyUsage, error)

	// GetRecentUsage 최근 N일 사용량 조회
	GetRecentUsage(ctx context.Context, days int) ([]DailyUsage, error)

	// GetTotalUsage 최근 N일 합계 조회
	GetTotalUsage(ctx context.Context, days int) (DailyUsage, error)

	// Close 리소스 정리
	Close()
}

// Repository가 Store 인터페이스를 구현하는지 컴파일 타임 확인
var _ Store = (*Repository)(nil)
