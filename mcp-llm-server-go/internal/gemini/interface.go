package gemini

import (
	"context"

	"github.com/park285/llm-kakao-bots/mcp-llm-server-go/internal/llm"
)

// LLM: LLM 클라이언트 인터페이스입니다.
// 테스트에서 mock 구현을 주입할 수 있습니다.
type LLM interface {
	// Chat 텍스트 채팅 요청
	Chat(ctx context.Context, req Request) (string, string, error)

	// ChatWithUsage 채팅 + 사용량 반환
	ChatWithUsage(ctx context.Context, req Request) (llm.ChatResult, string, error)

	// Structured JSON 스키마 기반 응답
	Structured(ctx context.Context, req Request, schema map[string]any) (map[string]any, string, error)
}

// Client가 LLM 인터페이스를 구현하는지 컴파일 타임 확인
var _ LLM = (*Client)(nil)
