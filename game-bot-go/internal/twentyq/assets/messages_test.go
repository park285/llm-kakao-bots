package assets

import (
	"testing"

	"github.com/park285/llm-kakao-bots/game-bot-go/internal/common/messageprovider"
	"github.com/park285/llm-kakao-bots/game-bot-go/internal/common/textutil"
	qconfig "github.com/park285/llm-kakao-bots/game-bot-go/internal/twentyq/config"
)

func TestGameMessagesYAML_Parses(t *testing.T) {
	provider, err := messageprovider.NewFromYAMLAtPath(GameMessagesYAML, "toon")
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if got := provider.Get("usage.fetch_failed"); got == "usage.fetch_failed" {
		t.Fatalf("expected usage.fetch_failed to exist")
	}
}

func TestHelpMessage_NotChunked(t *testing.T) {
	provider, err := messageprovider.NewFromYAMLAtPath(GameMessagesYAML, "toon")
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	help := provider.Get("help.message")
	chunks := textutil.ChunkByLines(help, qconfig.KakaoMessageMaxLength)
	if len(chunks) != 1 {
		t.Fatalf("expected help.message to be 1 chunk, got %d", len(chunks))
	}
}
