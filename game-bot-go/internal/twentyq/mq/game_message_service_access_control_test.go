package mq

import (
	"context"
	"io"
	"log/slog"
	"testing"

	"github.com/park285/llm-kakao-bots/game-bot-go/internal/common/messageprovider"
	"github.com/park285/llm-kakao-bots/game-bot-go/internal/common/mqmsg"
	qconfig "github.com/park285/llm-kakao-bots/game-bot-go/internal/twentyq/config"
	qsecurity "github.com/park285/llm-kakao-bots/game-bot-go/internal/twentyq/security"
)

func TestGameMessageService_isAccessAllowed_UserBlockedSendsNickname(t *testing.T) {
	msgProvider, err := messageprovider.NewFromYAML("error:\n  user_blocked: \"BLOCK:{nickname}\"\nuser:\n  anonymous: \"anon\"\n")
	if err != nil {
		t.Fatalf("message provider init failed: %v", err)
	}

	accessControl := qsecurity.NewAccessControl(qconfig.AccessConfig{
		Passthrough:    false,
		Enabled:        true,
		BlockedUserIDs: []string{"user1"},
		BlockedChatIDs: nil,
		AllowedChatIDs: nil,
	})

	var published []mqmsg.OutboundMessage
	sender := NewMessageSender(msgProvider, func(ctx context.Context, msg mqmsg.OutboundMessage) error {
		published = append(published, msg)
		return nil
	})

	svc := &GameMessageService{
		messageSender: sender,
		msgProvider:   msgProvider,
		accessControl: accessControl,
		logger:        slog.New(slog.NewTextHandler(io.Discard, nil)),
	}

	nick := "Nick"
	in := mqmsg.InboundMessage{
		ChatID:  "chat1",
		UserID:  "user1",
		Sender:  &nick,
		Content: "/스자 시작",
	}

	allowed := svc.isAccessAllowed(context.Background(), in, Command{Kind: CommandStart})
	if allowed {
		t.Fatal("expected access denied")
	}
	if len(published) != 1 {
		t.Fatalf("expected 1 published message, got %d", len(published))
	}
	// SendError는 NewFinal을 사용하여 Iris의 에러 이모지 추가를 방지한다
	if published[0].Type != mqmsg.OutboundFinal {
		t.Fatalf("expected final outbound (SendError uses NewFinal), got %s", published[0].Type)
	}
	if published[0].Text != "BLOCK:Nick" {
		t.Fatalf("unexpected error message: %q", published[0].Text)
	}
}

func TestGameMessageService_isAccessAllowed_AdminBypass(t *testing.T) {
	msgProvider, err := messageprovider.NewFromYAML("error:\n  user_blocked: \"BLOCK:{nickname}\"\nuser:\n  anonymous: \"anon\"\n")
	if err != nil {
		t.Fatalf("message provider init failed: %v", err)
	}

	accessControl := qsecurity.NewAccessControl(qconfig.AccessConfig{
		Passthrough:    false,
		Enabled:        true,
		BlockedUserIDs: []string{"user1"},
	})

	var published []mqmsg.OutboundMessage
	sender := NewMessageSender(msgProvider, func(ctx context.Context, msg mqmsg.OutboundMessage) error {
		published = append(published, msg)
		return nil
	})

	svc := &GameMessageService{
		messageSender: sender,
		msgProvider:   msgProvider,
		accessControl: accessControl,
		logger:        slog.New(slog.NewTextHandler(io.Discard, nil)),
	}

	in := mqmsg.InboundMessage{
		ChatID:  "chat1",
		UserID:  "user1",
		Content: "/스자 admin force-end",
	}

	allowed := svc.isAccessAllowed(context.Background(), in, Command{Kind: CommandAdminForceEnd})
	if !allowed {
		t.Fatal("expected admin bypass allowed")
	}
	if len(published) != 0 {
		t.Fatalf("expected no published messages, got %d", len(published))
	}
}

func TestGameMessageService_isAccessAllowed_AccessDeniedIsSilent(t *testing.T) {
	msgProvider, err := messageprovider.NewFromYAML("error:\n  access_denied: \"DENIED\"\n")
	if err != nil {
		t.Fatalf("message provider init failed: %v", err)
	}

	accessControl := qsecurity.NewAccessControl(qconfig.AccessConfig{
		Passthrough:    false,
		Enabled:        true,
		AllowedChatIDs: []string{"allowedChat"},
	})

	var published []mqmsg.OutboundMessage
	sender := NewMessageSender(msgProvider, func(ctx context.Context, msg mqmsg.OutboundMessage) error {
		published = append(published, msg)
		return nil
	})

	svc := &GameMessageService{
		messageSender: sender,
		msgProvider:   msgProvider,
		accessControl: accessControl,
		logger:        slog.New(slog.NewTextHandler(io.Discard, nil)),
	}

	in := mqmsg.InboundMessage{
		ChatID:  "otherChat",
		UserID:  "user1",
		Content: "/스자 시작",
	}

	allowed := svc.isAccessAllowed(context.Background(), in, Command{Kind: CommandStart})
	if allowed {
		t.Fatal("expected access denied")
	}
	if len(published) != 0 {
		t.Fatalf("expected 0 published messages, got %d", len(published))
	}
}
