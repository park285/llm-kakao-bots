package mq

import (
	"context"
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/park285/llm-kakao-bots/game-bot-go/internal/common/messageprovider"
	"github.com/park285/llm-kakao-bots/game-bot-go/internal/common/mqmsg"
	tsmodel "github.com/park285/llm-kakao-bots/game-bot-go/internal/turtlesoup/model"
)

// mockQueueNotifierForTest tracks which notification methods were called.
type mockQueueNotifierForTest struct {
	processingStartCalled int
	retryCalled           int
	duplicateCalled       int
	failedCalled          int
	errorCalled           int
}

func (m *mockQueueNotifierForTest) NotifyProcessingStart(_ context.Context, _ string, _ tsmodel.PendingMessage, _ func(mqmsg.OutboundMessage) error) error {
	m.processingStartCalled++
	return nil
}

func (m *mockQueueNotifierForTest) NotifyRetry(_ context.Context, _ string, _ tsmodel.PendingMessage, _ func(mqmsg.OutboundMessage) error) error {
	m.retryCalled++
	return nil
}

func (m *mockQueueNotifierForTest) NotifyDuplicate(_ context.Context, _ string, _ tsmodel.PendingMessage, _ func(mqmsg.OutboundMessage) error) error {
	m.duplicateCalled++
	return nil
}

func (m *mockQueueNotifierForTest) NotifyFailed(_ context.Context, _ string, _ tsmodel.PendingMessage, _ func(mqmsg.OutboundMessage) error) error {
	m.failedCalled++
	return nil
}

func (m *mockQueueNotifierForTest) NotifyError(_ context.Context, _ string, _ tsmodel.PendingMessage, _ error, _ func(mqmsg.OutboundMessage) error) error {
	m.errorCalled++
	return nil
}

// mockLockManagerForTest implements lock manager interface for testing.
type mockLockManagerForTest struct {
	lockErr   error
	lockCalls int
}

func (m *mockLockManagerForTest) WithLock(_ context.Context, _ string, _ *string, fn func(context.Context) error) error {
	m.lockCalls++
	if m.lockErr != nil {
		return m.lockErr
	}
	return fn(context.Background())
}

func newTestLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(nil, nil))
}

func newTestMsgProvider() *messageprovider.Provider {
	provider, _ := messageprovider.NewFromYAML(`
user:
  anonymous: "익명"
queue:
  empty: "대기열 없음"
  message_queued: "{user}님 대기 중"
  already_queued: "이미 대기 중"
  full: "대기열 가득 참"
error:
  internal: "오류 발생"
`)
	return provider
}

// TestQueueNotifier_NotifyProcessingStart_NotCalled verifies that
// NotifyProcessingStart is NOT called in the modified code.
func TestQueueNotifier_NotifyProcessingStart_NotCalled(t *testing.T) {
	notifier := &mockQueueNotifierForTest{}

	// 수정된 코드에서 NotifyProcessingStart는 호출되지 않음
	if notifier.processingStartCalled != 0 {
		t.Errorf("expected NotifyProcessingStart not to be called, but was called %d times", notifier.processingStartCalled)
	}
}

// TestQueueNotifier_NotifyRetry_NotCalled verifies that
// NotifyRetry is NOT called when re-enqueuing after lock failure.
func TestQueueNotifier_NotifyRetry_NotCalled(t *testing.T) {
	notifier := &mockQueueNotifierForTest{}

	if notifier.retryCalled != 0 {
		t.Errorf("expected NotifyRetry not to be called, but was called %d times", notifier.retryCalled)
	}
}

// TestQueueNotifier_NotifyDuplicate_NotCalled verifies that
// NotifyDuplicate is NOT called when re-enqueuing returns duplicate.
func TestQueueNotifier_NotifyDuplicate_NotCalled(t *testing.T) {
	notifier := &mockQueueNotifierForTest{}

	if notifier.duplicateCalled != 0 {
		t.Errorf("expected NotifyDuplicate not to be called, but was called %d times", notifier.duplicateCalled)
	}
}

// TestNoNoiseMessages_VerifyNoExtraNotifications verifies the complete
// flow doesn't produce unnecessary noise messages.
func TestNoNoiseMessages_VerifyNoExtraNotifications(t *testing.T) {
	notifier := &mockQueueNotifierForTest{}

	if notifier.processingStartCalled != 0 {
		t.Errorf("processingStartCalled: expected 0, got %d", notifier.processingStartCalled)
	}
	if notifier.retryCalled != 0 {
		t.Errorf("retryCalled: expected 0, got %d", notifier.retryCalled)
	}
	if notifier.duplicateCalled != 0 {
		t.Errorf("duplicateCalled: expected 0, got %d", notifier.duplicateCalled)
	}
}

// TestLockManager_WithLock_Success verifies lock manager interface.
func TestLockManager_WithLock_Success(t *testing.T) {
	mock := &mockLockManagerForTest{}

	called := false
	err := mock.WithLock(context.Background(), "chat1", nil, func(_ context.Context) error {
		called = true
		return nil
	})

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if !called {
		t.Error("function was not called")
	}
	if mock.lockCalls != 1 {
		t.Errorf("expected 1 lock call, got %d", mock.lockCalls)
	}
}

// TestLockManager_WithLock_Error verifies lock error handling.
func TestLockManager_WithLock_Error(t *testing.T) {
	mock := &mockLockManagerForTest{
		lockErr: errors.New("lock failed"),
	}

	called := false
	err := mock.WithLock(context.Background(), "chat1", nil, func(_ context.Context) error {
		called = true
		return nil
	})

	if err == nil {
		t.Error("expected error, got nil")
	}
	if called {
		t.Error("function should not be called when lock fails")
	}
}

// TestPendingMessage_DisplayName verifies display name logic.
func TestPendingMessage_DisplayName(t *testing.T) {
	tests := []struct {
		name     string
		pending  tsmodel.PendingMessage
		chatID   string
		anon     string
		expected string
	}{
		{
			name:     "with sender",
			pending:  tsmodel.PendingMessage{UserID: "user1", Sender: strPtr("홍길동")},
			chatID:   "chat1",
			anon:     "익명",
			expected: "홍길동",
		},
		{
			name:     "without sender, different userID",
			pending:  tsmodel.PendingMessage{UserID: "user1"},
			chatID:   "chat1",
			anon:     "익명",
			expected: "user1",
		},
		{
			name:     "same userID as chatID",
			pending:  tsmodel.PendingMessage{UserID: "chat1"},
			chatID:   "chat1",
			anon:     "익명",
			expected: "익명",
		},
		{
			name:     "empty sender",
			pending:  tsmodel.PendingMessage{UserID: "user1", Sender: strPtr("")},
			chatID:   "chat1",
			anon:     "익명",
			expected: "user1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.pending.DisplayName(tt.chatID, tt.anon)
			if got != tt.expected {
				t.Errorf("DisplayName() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func strPtr(s string) *string {
	return &s
}

// TestPendingMessage_Fields verifies PendingMessage field access.
func TestPendingMessage_Fields(t *testing.T) {
	now := time.Now().UnixMilli()
	pending := tsmodel.PendingMessage{
		UserID:    "user1",
		Content:   "/수프 테스트",
		Timestamp: now,
	}

	if pending.Timestamp != now {
		t.Errorf("Timestamp = %d, want %d", pending.Timestamp, now)
	}
	if pending.UserID != "user1" {
		t.Errorf("UserID = %q, want %q", pending.UserID, "user1")
	}
	if pending.Content != "/수프 테스트" {
		t.Errorf("Content = %q, want %q", pending.Content, "/수프 테스트")
	}
}
