package service

import (
	"context"
	"strings"
	"time"
)

const (
	playerRegistrationParallelism = 32
	playerRegistrationQueueSize   = 1024
	playerRegistrationTimeout     = 5 * time.Second
)

type playerRegistrationTask struct {
	chatID string
	userID string
	sender string
}

// RegisterPlayerAsync: 비동기로 플레이어 등록 작업을 큐에 추가한다. (병목 방지)
func (s *RiddleService) RegisterPlayerAsync(ctx context.Context, chatID string, userID string, sender *string) {
	if s == nil {
		return
	}

	if ctx == nil {
		ctx = context.Background()
	}

	chatID = strings.TrimSpace(chatID)
	userID = strings.TrimSpace(userID)
	if chatID == "" || userID == "" {
		return
	}

	s.startPlayerRegistrationWorkers(ctx)

	senderText := ""
	if sender != nil {
		senderText = strings.TrimSpace(*sender)
	}

	task := playerRegistrationTask{
		chatID: chatID,
		userID: userID,
		sender: senderText,
	}

	if s.playerRegistrationTasks == nil {
		timeoutCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), playerRegistrationTimeout)
		defer cancel()

		if err := s.RegisterPlayer(timeoutCtx, chatID, userID, sender); err != nil {
			s.logger.Error("playerset_add_failed", "chat_id", chatID, "user_id", userID, "err", err)
		}
		return
	}

	select {
	case s.playerRegistrationTasks <- task:
	default:
		s.logger.Warn("playerset_queue_full", "chat_id", chatID, "user_id", userID)
	}
}

func (s *RiddleService) startPlayerRegistrationWorkers(ctx context.Context) {
	if s == nil {
		return
	}

	if ctx == nil {
		ctx = context.Background()
	}

	baseCtx := context.WithoutCancel(ctx)
	s.playerRegistrationOnce.Do(func() {
		s.playerRegistrationTasks = make(chan playerRegistrationTask, playerRegistrationQueueSize)
		for i := 0; i < playerRegistrationParallelism; i++ {
			go s.playerRegistrationWorker(baseCtx)
		}
	})
}

func (s *RiddleService) playerRegistrationWorker(ctx context.Context) {
	for task := range s.playerRegistrationTasks {
		timeoutCtx, cancel := context.WithTimeout(ctx, playerRegistrationTimeout)
		sender := strings.TrimSpace(task.sender)
		var senderPtr *string
		if sender != "" {
			senderPtr = &sender
		}

		if err := s.RegisterPlayer(timeoutCtx, task.chatID, task.userID, senderPtr); err != nil {
			s.logger.Error("playerset_add_failed", "chat_id", task.chatID, "user_id", task.userID, "err", err)
		}
		cancel()
	}
}
