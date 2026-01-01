package mq

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"

	"github.com/park285/llm-kakao-bots/game-bot-go/internal/common/mqmsg"
	"github.com/park285/llm-kakao-bots/game-bot-go/internal/common/pending"
	domainmodels "github.com/park285/llm-kakao-bots/game-bot-go/internal/domain/models"
)

type lockManagerStub struct {
	err         error
	calls       int
	holderName  string
	calledBlock bool
}

func (m *lockManagerStub) WithLock(ctx context.Context, _ string, holderName *string, block func(ctx context.Context) error) error {
	m.calls++
	if holderName != nil {
		m.holderName = *holderName
	}
	if m.err != nil {
		return m.err
	}
	m.calledBlock = true
	return block(ctx)
}

type processingLockServiceStub struct {
	startCalls  int
	finishCalls int

	startErr  error
	finishErr error
}

func (s *processingLockServiceStub) StartProcessing(_ context.Context, _ string) error {
	s.startCalls++
	return s.startErr
}

func (s *processingLockServiceStub) FinishProcessing(_ context.Context, _ string) error {
	s.finishCalls++
	return s.finishErr
}

type notifierStub struct {
	failedCalls int
	errorCalls  int

	lastError error
}

func (n *notifierStub) NotifyFailed(_ context.Context, _ string, _ domainmodels.PendingMessage, _ func(mqmsg.OutboundMessage) error) error {
	n.failedCalls++
	return nil
}

func (n *notifierStub) NotifyError(_ context.Context, _ string, _ domainmodels.PendingMessage, err error, _ func(mqmsg.OutboundMessage) error) error {
	n.errorCalls++
	n.lastError = err
	return nil
}

func newTestLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func strPtr(s string) *string {
	return &s
}

func TestProcessSingleQueuedMessage_lockSuccess_executesAndFinishesProcessing(t *testing.T) {
	ctx := context.Background()

	lockManager := &lockManagerStub{}
	processingLock := &processingLockServiceStub{}
	notifier := &notifierStub{}
	reEnqueueCalls := 0
	executorCalls := 0

	reEnqueue := func(_ context.Context, _ string, _ domainmodels.PendingMessage) (pending.EnqueueResult, error) {
		reEnqueueCalls++
		return pending.EnqueueSuccess, nil
	}
	executor := func(_ context.Context, _ string, _ domainmodels.PendingMessage, _ func(mqmsg.OutboundMessage) error) error {
		executorCalls++
		return nil
	}

	ok := ProcessSingleQueuedMessage(
		ctx,
		newTestLogger(),
		lockManager,
		processingLock,
		notifier,
		reEnqueue,
		executor,
		"chat1",
		domainmodels.PendingMessage{UserID: "user1", Sender: strPtr("Alice")},
		func(mqmsg.OutboundMessage) error { return nil },
	)

	if !ok {
		t.Fatalf("expected ok=true")
	}
	if lockManager.calls != 1 {
		t.Fatalf("expected lock calls=1, got %d", lockManager.calls)
	}
	if !lockManager.calledBlock {
		t.Fatalf("expected lock block to be called")
	}
	if lockManager.holderName != "Alice" {
		t.Fatalf("expected holderName=%q, got %q", "Alice", lockManager.holderName)
	}

	if processingLock.startCalls != 1 {
		t.Fatalf("expected StartProcessing calls=1, got %d", processingLock.startCalls)
	}
	if processingLock.finishCalls != 1 {
		t.Fatalf("expected FinishProcessing calls=1, got %d", processingLock.finishCalls)
	}
	if executorCalls != 1 {
		t.Fatalf("expected executor calls=1, got %d", executorCalls)
	}
	if reEnqueueCalls != 0 {
		t.Fatalf("expected reEnqueue calls=0, got %d", reEnqueueCalls)
	}
	if notifier.failedCalls != 0 {
		t.Fatalf("expected NotifyFailed calls=0, got %d", notifier.failedCalls)
	}
	if notifier.errorCalls != 0 {
		t.Fatalf("expected NotifyError calls=0, got %d", notifier.errorCalls)
	}
}

func TestProcessSingleQueuedMessage_executorError_triggersNotifyErrorButStillReturnsTrue(t *testing.T) {
	ctx := context.Background()

	lockManager := &lockManagerStub{}
	processingLock := &processingLockServiceStub{}
	notifier := &notifierStub{}
	reEnqueueCalls := 0

	execErr := errors.New("executor failed")
	executor := func(_ context.Context, _ string, _ domainmodels.PendingMessage, _ func(mqmsg.OutboundMessage) error) error {
		return execErr
	}
	reEnqueue := func(_ context.Context, _ string, _ domainmodels.PendingMessage) (pending.EnqueueResult, error) {
		reEnqueueCalls++
		return pending.EnqueueSuccess, nil
	}

	ok := ProcessSingleQueuedMessage(
		ctx,
		newTestLogger(),
		lockManager,
		processingLock,
		notifier,
		reEnqueue,
		executor,
		"chat1",
		domainmodels.PendingMessage{UserID: "user1"},
		func(mqmsg.OutboundMessage) error { return nil },
	)

	if !ok {
		t.Fatalf("expected ok=true")
	}
	if notifier.errorCalls != 1 {
		t.Fatalf("expected NotifyError calls=1, got %d", notifier.errorCalls)
	}
	if !errors.Is(notifier.lastError, execErr) {
		t.Fatalf("expected NotifyError to receive execErr")
	}
	if processingLock.startCalls != 1 || processingLock.finishCalls != 1 {
		t.Fatalf("expected StartProcessing/FinishProcessing to be called once, got start=%d finish=%d", processingLock.startCalls, processingLock.finishCalls)
	}
	if reEnqueueCalls != 0 {
		t.Fatalf("expected reEnqueue calls=0, got %d", reEnqueueCalls)
	}
}

func TestProcessSingleQueuedMessage_lockFailure_reEnqueueAndNotifyFailedOnlyOnQueueFull(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name              string
		reEnqueueResult   pending.EnqueueResult
		reEnqueueErr      error
		expectedFailed    int
		expectedReEnqueue int
	}{
		{name: "reEnqueue success", reEnqueueResult: pending.EnqueueSuccess, expectedFailed: 0, expectedReEnqueue: 1},
		{name: "reEnqueue duplicate", reEnqueueResult: pending.EnqueueDuplicate, expectedFailed: 0, expectedReEnqueue: 1},
		{name: "reEnqueue queue full", reEnqueueResult: pending.EnqueueQueueFull, expectedFailed: 1, expectedReEnqueue: 1},
		{name: "reEnqueue unknown", reEnqueueResult: pending.EnqueueResult(999), expectedFailed: 1, expectedReEnqueue: 1},
		{name: "reEnqueue error", reEnqueueErr: errors.New("reEnqueue failed"), expectedFailed: 0, expectedReEnqueue: 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lockManager := &lockManagerStub{err: errors.New("lock failed")}
			processingLock := &processingLockServiceStub{}
			notifier := &notifierStub{}
			reEnqueueCalls := 0
			executorCalls := 0

			executor := func(_ context.Context, _ string, _ domainmodels.PendingMessage, _ func(mqmsg.OutboundMessage) error) error {
				executorCalls++
				return nil
			}
			reEnqueue := func(_ context.Context, _ string, _ domainmodels.PendingMessage) (pending.EnqueueResult, error) {
				reEnqueueCalls++
				if tt.reEnqueueErr != nil {
					return pending.EnqueueSuccess, tt.reEnqueueErr
				}
				return tt.reEnqueueResult, nil
			}

			ok := ProcessSingleQueuedMessage(
				ctx,
				newTestLogger(),
				lockManager,
				processingLock,
				notifier,
				reEnqueue,
				executor,
				"chat1",
				domainmodels.PendingMessage{UserID: "user1", Sender: strPtr("")},
				func(mqmsg.OutboundMessage) error { return nil },
			)

			if ok {
				t.Fatalf("expected ok=false")
			}
			if executorCalls != 0 {
				t.Fatalf("expected executor calls=0, got %d", executorCalls)
			}
			if processingLock.startCalls != 0 || processingLock.finishCalls != 0 {
				t.Fatalf("expected processing lock calls=0, got start=%d finish=%d", processingLock.startCalls, processingLock.finishCalls)
			}
			if lockManager.holderName != "user1" {
				t.Fatalf("expected holderName=%q, got %q", "user1", lockManager.holderName)
			}
			if reEnqueueCalls != tt.expectedReEnqueue {
				t.Fatalf("expected reEnqueue calls=%d, got %d", tt.expectedReEnqueue, reEnqueueCalls)
			}
			if notifier.failedCalls != tt.expectedFailed {
				t.Fatalf("expected NotifyFailed calls=%d, got %d", tt.expectedFailed, notifier.failedCalls)
			}
		})
	}
}

func TestProcessSingleQueuedMessage_startProcessingError_triggersReEnqueue(t *testing.T) {
	ctx := context.Background()

	lockManager := &lockManagerStub{}
	processingLock := &processingLockServiceStub{startErr: errors.New("start failed")}
	notifier := &notifierStub{}
	reEnqueueCalls := 0
	executorCalls := 0

	reEnqueue := func(_ context.Context, _ string, _ domainmodels.PendingMessage) (pending.EnqueueResult, error) {
		reEnqueueCalls++
		return pending.EnqueueDuplicate, nil
	}
	executor := func(_ context.Context, _ string, _ domainmodels.PendingMessage, _ func(mqmsg.OutboundMessage) error) error {
		executorCalls++
		return nil
	}

	ok := ProcessSingleQueuedMessage(
		ctx,
		newTestLogger(),
		lockManager,
		processingLock,
		notifier,
		reEnqueue,
		executor,
		"chat1",
		domainmodels.PendingMessage{UserID: "user1"},
		func(mqmsg.OutboundMessage) error { return nil },
	)

	if ok {
		t.Fatalf("expected ok=false")
	}
	if processingLock.startCalls != 1 {
		t.Fatalf("expected StartProcessing calls=1, got %d", processingLock.startCalls)
	}
	if processingLock.finishCalls != 0 {
		t.Fatalf("expected FinishProcessing calls=0, got %d", processingLock.finishCalls)
	}
	if executorCalls != 0 {
		t.Fatalf("expected executor calls=0, got %d", executorCalls)
	}
	if reEnqueueCalls != 1 {
		t.Fatalf("expected reEnqueue calls=1, got %d", reEnqueueCalls)
	}
	if notifier.failedCalls != 0 || notifier.errorCalls != 0 {
		t.Fatalf("expected no notifier calls, got failed=%d error=%d", notifier.failedCalls, notifier.errorCalls)
	}
}

