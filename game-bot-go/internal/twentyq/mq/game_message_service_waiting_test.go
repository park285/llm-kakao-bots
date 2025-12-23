package mq

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/park285/llm-kakao-bots/game-bot-go/internal/common/messageprovider"
	"github.com/park285/llm-kakao-bots/game-bot-go/internal/common/mqmsg"
)

type fakeHintAvailabilityRegistrar struct {
	can bool
	err error
}

func (f *fakeHintAvailabilityRegistrar) RegisterPlayerAsync(ctx context.Context, chatID string, userID string, sender *string) {
}

func (f *fakeHintAvailabilityRegistrar) HasSession(ctx context.Context, chatID string) (bool, error) {
	return true, nil
}

func (f *fakeHintAvailabilityRegistrar) CanGenerateHint(ctx context.Context, chatID string) (bool, error) {
	return f.can, f.err
}

type fakeSessionRegistrar struct {
	hasSession bool
	err        error
}

func (f *fakeSessionRegistrar) RegisterPlayerAsync(ctx context.Context, chatID string, userID string, sender *string) {
}

func (f *fakeSessionRegistrar) HasSession(ctx context.Context, chatID string) (bool, error) {
	return f.hasSession, f.err
}

func TestGameMessageService_shouldSendWaiting_CommandHints_NoBudget(t *testing.T) {
	svc := &GameMessageService{
		playerRegistrar: &fakeHintAvailabilityRegistrar{can: false},
	}

	if svc.shouldSendWaiting(context.Background(), "chat1", Command{Kind: CommandHints}) {
		t.Fatal("expected shouldSendWaiting=false when hint budget is exhausted")
	}
}

func TestGameMessageService_shouldSendWaiting_CommandHints_HasBudget(t *testing.T) {
	svc := &GameMessageService{
		playerRegistrar: &fakeHintAvailabilityRegistrar{can: true},
	}

	if !svc.shouldSendWaiting(context.Background(), "chat1", Command{Kind: CommandHints}) {
		t.Fatal("expected shouldSendWaiting=true when hint budget is available")
	}
}

func TestGameMessageService_shouldSendWaiting_CommandHints_CheckError(t *testing.T) {
	svc := &GameMessageService{
		playerRegistrar: &fakeHintAvailabilityRegistrar{err: errors.New("boom")},
	}

	if svc.shouldSendWaiting(context.Background(), "chat1", Command{Kind: CommandHints}) {
		t.Fatal("expected shouldSendWaiting=false when hint availability check fails")
	}
}

func TestGameMessageService_shouldSendWaiting_CommandAsk_AlwaysWaiting(t *testing.T) {
	svc := &GameMessageService{}

	if !svc.shouldSendWaiting(context.Background(), "chat1", Command{Kind: CommandAsk}) {
		t.Fatal("expected shouldSendWaiting=true for ask command")
	}
}

func TestGameMessageService_shouldSendWaiting_CommandStart_SessionExists_NoWaiting(t *testing.T) {
	svc := &GameMessageService{
		playerRegistrar: &fakeSessionRegistrar{hasSession: true},
	}

	if svc.shouldSendWaiting(context.Background(), "chat1", Command{Kind: CommandStart}) {
		t.Fatal("expected shouldSendWaiting=false when session already exists for start command")
	}
}

func TestGameMessageService_shouldSendWaiting_CommandStart_NoSession_Waiting(t *testing.T) {
	svc := &GameMessageService{
		playerRegistrar: &fakeSessionRegistrar{hasSession: false},
	}

	if !svc.shouldSendWaiting(context.Background(), "chat1", Command{Kind: CommandStart}) {
		t.Fatal("expected shouldSendWaiting=true when no session exists for start command")
	}
}

func TestGameMessageService_shouldSendWaiting_CommandStart_CheckError_NoWaiting(t *testing.T) {
	svc := &GameMessageService{
		playerRegistrar: &fakeSessionRegistrar{err: errors.New("boom")},
	}

	if svc.shouldSendWaiting(context.Background(), "chat1", Command{Kind: CommandStart}) {
		t.Fatal("expected shouldSendWaiting=false when session check fails for start command")
	}
}

func TestGameMessageService_executeCommand_CommandAsk_ProcessingWaitingDelayed_NoWaiting(t *testing.T) {
	msgProvider, err := messageprovider.NewFromYAML("processing:\n  waiting: \"WAITING\"\n")
	if err != nil {
		t.Fatalf("message provider init failed: %v", err)
	}

	var published []mqmsg.OutboundMessage
	sender := NewMessageSender(msgProvider, func(ctx context.Context, msg mqmsg.OutboundMessage) error {
		published = append(published, msg)
		return nil
	})

	commandHandler := &GameCommandHandler{
		handlers: map[CommandKind]commandHandlerFunc{
			CommandAsk: func(ctx context.Context, message mqmsg.InboundMessage, command Command) ([]string, error) {
				return []string{"OK"}, nil
			},
		},
	}

	svc := &GameMessageService{
		commandHandler:         commandHandler,
		messageSender:          sender,
		processingWaitingDelay: 500 * time.Millisecond,
	}

	svc.executeCommand(context.Background(), mqmsg.InboundMessage{ChatID: "chat1", UserID: "user1"}, Command{Kind: CommandAsk})

	if len(published) != 1 {
		t.Fatalf("expected 1 published message, got %d", len(published))
	}
	if published[0].Type != mqmsg.OutboundFinal {
		t.Fatalf("expected final, got %s", published[0].Type)
	}
	if published[0].Text != "OK" {
		t.Fatalf("unexpected final message: %q", published[0].Text)
	}
}

func TestGameMessageService_executeCommand_CommandAsk_ProcessingWaitingDelayed_SendsWaiting(t *testing.T) {
	msgProvider, err := messageprovider.NewFromYAML("processing:\n  waiting: \"WAITING\"\n")
	if err != nil {
		t.Fatalf("message provider init failed: %v", err)
	}

	var published []mqmsg.OutboundMessage
	sender := NewMessageSender(msgProvider, func(ctx context.Context, msg mqmsg.OutboundMessage) error {
		published = append(published, msg)
		return nil
	})

	commandHandler := &GameCommandHandler{
		handlers: map[CommandKind]commandHandlerFunc{
			CommandAsk: func(ctx context.Context, message mqmsg.InboundMessage, command Command) ([]string, error) {
				time.Sleep(50 * time.Millisecond)
				return []string{"OK"}, nil
			},
		},
	}

	svc := &GameMessageService{
		commandHandler:         commandHandler,
		messageSender:          sender,
		processingWaitingDelay: 10 * time.Millisecond,
	}

	svc.executeCommand(context.Background(), mqmsg.InboundMessage{ChatID: "chat1", UserID: "user1"}, Command{Kind: CommandAsk})

	if len(published) != 2 {
		t.Fatalf("expected 2 published messages, got %d", len(published))
	}
	if published[0].Type != mqmsg.OutboundWaiting {
		t.Fatalf("expected waiting, got %s", published[0].Type)
	}
	if published[0].Text != "WAITING" {
		t.Fatalf("unexpected waiting message: %q", published[0].Text)
	}
	if published[1].Type != mqmsg.OutboundFinal {
		t.Fatalf("expected final, got %s", published[1].Type)
	}
	if published[1].Text != "OK" {
		t.Fatalf("unexpected final message: %q", published[1].Text)
	}
}
