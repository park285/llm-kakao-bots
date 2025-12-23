package mq

import (
	"context"
	"io"
	"log/slog"
	"testing"

	"github.com/park285/llm-kakao-bots/game-bot-go/internal/common/messageprovider"
	"github.com/park285/llm-kakao-bots/game-bot-go/internal/common/mqmsg"
	domainmodels "github.com/park285/llm-kakao-bots/game-bot-go/internal/domain/models"
	qmessages "github.com/park285/llm-kakao-bots/game-bot-go/internal/twentyq/messages"
	qmodel "github.com/park285/llm-kakao-bots/game-bot-go/internal/twentyq/model"
	qredis "github.com/park285/llm-kakao-bots/game-bot-go/internal/twentyq/redis"
	qsvc "github.com/park285/llm-kakao-bots/game-bot-go/internal/twentyq/service"
)

type fakeChainedQuestionRiddleService struct {
	lastQuestion string
	lastIsChain  bool
	answerScale  qmodel.FiveScaleKo
	statusMain   string
	statusHint   string
}

func (f *fakeChainedQuestionRiddleService) AnswerWithOutcome(
	ctx context.Context,
	chatID string,
	userID string,
	sender *string,
	question string,
	isChain bool,
) (qsvc.AnswerOutcome, error) {
	f.lastQuestion = question
	f.lastIsChain = isChain
	return qsvc.AnswerOutcome{
		Message:         "OK",
		Scale:           f.answerScale,
		IsAnswerAttempt: false,
	}, nil
}

func (f *fakeChainedQuestionRiddleService) StatusSeparated(ctx context.Context, chatID string) (string, string, error) {
	return f.statusMain, f.statusHint, nil
}

type fakeChainedQuestionQueueCoordinator struct {
	enqueued  []qmodel.PendingMessage
	skipFlags map[string]bool
}

func newFakeChainedQuestionQueueCoordinator() *fakeChainedQuestionQueueCoordinator {
	return &fakeChainedQuestionQueueCoordinator{
		skipFlags: make(map[string]bool),
	}
}

func (f *fakeChainedQuestionQueueCoordinator) Enqueue(ctx context.Context, chatID string, msg qmodel.PendingMessage) (qredis.EnqueueResult, error) {
	f.enqueued = append(f.enqueued, msg)
	return qredis.EnqueueSuccess, nil
}

func (f *fakeChainedQuestionQueueCoordinator) SetChainSkipFlag(ctx context.Context, chatID string, userID string) error {
	f.skipFlags[chatID+":"+userID] = true
	return nil
}

func (f *fakeChainedQuestionQueueCoordinator) CheckAndClearChainSkipFlag(ctx context.Context, chatID string, userID string) (bool, error) {
	key := chatID + ":" + userID
	has := f.skipFlags[key]
	delete(f.skipFlags, key)
	return has, nil
}

func TestChainedQuestionHandler_Handle_IF_TRUE_SetsSkipFlag(t *testing.T) {
	msgProvider, err := messageprovider.NewFromYAML("user:\n  anonymous: \"anon\"\nchain:\n  condition_not_met: \"SKIP:{questions}\"\n")
	if err != nil {
		t.Fatalf("message provider init failed: %v", err)
	}

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	queueCoordinator := newFakeChainedQuestionQueueCoordinator()
	riddleService := &fakeChainedQuestionRiddleService{answerScale: qmodel.FiveScaleAlwaysNo}

	handler := NewChainedQuestionHandler(riddleService, queueCoordinator, msgProvider, logger)

	res, err := handler.Handle(
		context.Background(),
		"chat1",
		"user1",
		nil,
		[]string{"사람이면", "직업인가요"},
		qmodel.ChainConditionIfTrue,
	)
	if err != nil {
		t.Fatalf("handle failed: %v", err)
	}
	// 조건 불만족 시 스킵 알림이 포함됨
	expectedResponse := "OK\n\nSKIP:직업인가요"
	if res != expectedResponse {
		t.Fatalf("unexpected response: %q, expected: %q", res, expectedResponse)
	}
	if riddleService.lastQuestion != "사람이면" {
		t.Fatalf("expected original first question, got %q", riddleService.lastQuestion)
	}
	if !queueCoordinator.skipFlags["chat1:user1"] {
		t.Fatal("expected skip flag set")
	}
}

func TestChainedQuestionHandler_ProcessChainBatch_Skipped_EmitsSkipNotification(t *testing.T) {
	msgProvider, err := messageprovider.NewFromYAML("chain:\n  condition_not_met: \"SKIP:{questions}\"\n")
	if err != nil {
		t.Fatalf("message provider init failed: %v", err)
	}

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	queueCoordinator := newFakeChainedQuestionQueueCoordinator()
	queueCoordinator.skipFlags["chat1:user1"] = true

	riddleService := &fakeChainedQuestionRiddleService{answerScale: qmodel.FiveScaleAlwaysNo, statusMain: "STATUS"}
	handler := NewChainedQuestionHandler(riddleService, queueCoordinator, msgProvider, logger)

	var emitted []mqmsg.OutboundMessage
	emit := func(out mqmsg.OutboundMessage) error {
		emitted = append(emitted, out)
		return nil
	}

	err = handler.ProcessChainBatch(
		context.Background(),
		"chat1",
		qmodel.PendingMessage{
			UserID:         "user1",
			IsChainBatch:   true,
			BatchQuestions: []string{"Q1", "Q2"},
		},
		emit,
	)
	if err != nil {
		t.Fatalf("process chain batch failed: %v", err)
	}

	if len(emitted) != 1 {
		t.Fatalf("expected 1 emitted message, got %d", len(emitted))
	}
	if emitted[0].Type != mqmsg.OutboundFinal {
		t.Fatalf("expected final message, got %s", emitted[0].Type)
	}
	if emitted[0].Text != "SKIP:Q1, Q2" {
		t.Fatalf("unexpected skip message: %q", emitted[0].Text)
	}
}

func TestChainedQuestionHandler_ProcessChainBatch_UsesIsChainFlag(t *testing.T) {
	msgProvider, err := messageprovider.NewFromYAML(
		"error:\n  session_not_found: \"NOSESSION\"\n",
	)
	if err != nil {
		t.Fatalf("message provider init failed: %v", err)
	}

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	queueCoordinator := newFakeChainedQuestionQueueCoordinator()

	riddleService := &fakeChainedQuestionRiddleService{
		answerScale: qmodel.FiveScaleAlwaysYes,
		statusMain:  "STATUS",
	}
	handler := NewChainedQuestionHandler(riddleService, queueCoordinator, msgProvider, logger)

	var emitted []mqmsg.OutboundMessage
	emit := func(out mqmsg.OutboundMessage) error {
		emitted = append(emitted, out)
		return nil
	}

	err = handler.ProcessChainBatch(
		context.Background(),
		"chat1",
		qmodel.PendingMessage{
			UserID:         "user1",
			IsChainBatch:   true,
			BatchQuestions: []string{"Q1"},
		},
		emit,
	)
	if err != nil {
		t.Fatalf("process chain batch failed: %v", err)
	}
	if !riddleService.lastIsChain {
		t.Fatal("expected isChain=true for batch questions")
	}
	if len(emitted) != 1 || emitted[0].Text != "STATUS" {
		t.Fatalf("unexpected emitted messages: %+v", emitted)
	}
}

func TestChainedQuestionHandler_ProcessChainBatch_StatusWithHintLine_EmitsSeparatedMessages(t *testing.T) {
	msgProvider, err := messageprovider.NewFromYAML(
		"error:\n  session_not_found: \"NOSESSION\"\n",
	)
	if err != nil {
		t.Fatalf("message provider init failed: %v", err)
	}

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	queueCoordinator := newFakeChainedQuestionQueueCoordinator()

	riddleService := &fakeChainedQuestionRiddleService{
		answerScale: qmodel.FiveScaleAlwaysYes,
		statusMain:  "STATUS",
		statusHint:  "\U0001F4A1#1 HINT",
	}
	handler := NewChainedQuestionHandler(riddleService, queueCoordinator, msgProvider, logger)

	var emitted []mqmsg.OutboundMessage
	emit := func(out mqmsg.OutboundMessage) error {
		emitted = append(emitted, out)
		return nil
	}

	err = handler.ProcessChainBatch(
		context.Background(),
		"chat1",
		qmodel.PendingMessage{
			UserID:         "user1",
			IsChainBatch:   true,
			BatchQuestions: []string{"Q1"},
		},
		emit,
	)
	if err != nil {
		t.Fatalf("process chain batch failed: %v", err)
	}

	if len(emitted) != 2 {
		t.Fatalf("expected 2 emitted messages, got %d", len(emitted))
	}
	if emitted[0].Type != mqmsg.OutboundFinal || emitted[1].Type != mqmsg.OutboundFinal {
		t.Fatalf("expected final messages, got %+v", emitted)
	}
	if emitted[0].Text != "STATUS" {
		t.Fatalf("unexpected main message: %q", emitted[0].Text)
	}
	if emitted[1].Text != "\U0001F4A1#1 HINT" {
		t.Fatalf("unexpected hint message: %q", emitted[1].Text)
	}
}

func TestIsAccessBypassAdminCommand_ExcludesCaching(t *testing.T) {
	if !isAccessBypassAdminCommand(Command{Kind: CommandAdminForceEnd}) {
		t.Fatal("expected admin force-end bypass")
	}
	if !isAccessBypassAdminCommand(Command{Kind: CommandAdminClearAll}) {
		t.Fatal("expected admin clear-all bypass")
	}
	if isAccessBypassAdminCommand(Command{Kind: CommandAdminUsage}) {
		t.Fatal("expected admin usage not bypassed")
	}
	if isAccessBypassAdminCommand(Command{Kind: CommandStart}) {
		t.Fatal("expected start not bypassed")
	}
	if isAccessBypassAdminCommand(Command{Kind: CommandHelp}) {
		t.Fatal("expected help not bypassed")
	}
}

func TestIsAnswerCommand(t *testing.T) {
	if !isAnswerCommand("정답 고양이") {
		t.Fatal("expected answer command")
	}
	if !isAnswerCommand("  정답 고양이  ") {
		t.Fatal("expected answer command with spaces")
	}
	if isAnswerCommand("질문 고양이인가요") {
		t.Fatal("expected non-answer command")
	}
	if isAnswerCommand("") {
		t.Fatal("expected non-answer command for empty string")
	}
}

func TestErrorUserBlockedUsesNicknameParam(t *testing.T) {
	msgProvider, err := messageprovider.NewFromYAML("user:\n  anonymous: \"anon\"\n")
	if err != nil {
		t.Fatalf("message provider init failed: %v", err)
	}

	service := &GameMessageService{
		msgProvider: msgProvider,
	}

	nickname := domainmodels.DisplayName("chat1", "user1", nil, service.msgProvider.Get(qmessages.UserAnonymous))
	if nickname != "user1" {
		t.Fatalf("unexpected nickname: %q", nickname)
	}
}
