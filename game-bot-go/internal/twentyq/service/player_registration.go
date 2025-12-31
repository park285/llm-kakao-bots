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

// RegisterPlayerAsync: 비동기로 플레이어 등록 작업을 큐에 추가합니다. (병목 방지)
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

	s.startPlayerRegistrationWorkers()

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

	// stopped 채널 체크하여 종료 중이면 무시
	select {
	case <-s.playerRegistrationStopped:
		return
	default:
	}

	select {
	case s.playerRegistrationTasks <- task:
	case <-s.playerRegistrationStopped:
		// 종료 시그널 받음
	default:
		s.logger.Warn("playerset_queue_full", "chat_id", chatID, "user_id", userID)
	}
}

func (s *RiddleService) startPlayerRegistrationWorkers() {
	if s == nil {
		return
	}

	s.playerRegistrationOnce.Do(func() {
		s.playerRegistrationStopped = make(chan struct{})
		s.playerRegistrationTasks = make(chan playerRegistrationTask, playerRegistrationQueueSize)
		for i := 0; i < playerRegistrationParallelism; i++ {
			s.playerRegistrationWg.Add(1)
			go s.playerRegistrationWorker()
		}
		s.logger.Info("player_registration_workers_started", "count", playerRegistrationParallelism)
	})
}

func (s *RiddleService) playerRegistrationWorker() {
	defer s.playerRegistrationWg.Done()

	for task := range s.playerRegistrationTasks {
		timeoutCtx, cancel := context.WithTimeout(context.Background(), playerRegistrationTimeout)
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

// ShutdownPlayerRegistration: 플레이어 등록 워커들을 정상 종료합니다.
// 대기 중인 모든 작업이 완료될 때까지 블로킹됩니다.
func (s *RiddleService) ShutdownPlayerRegistration() {
	if s == nil || s.playerRegistrationTasks == nil {
		return
	}

	// 종료 시그널 전송 (새 작업 수신 차단)
	select {
	case <-s.playerRegistrationStopped:
		// 이미 닫힘
		return
	default:
		close(s.playerRegistrationStopped)
	}

	// 채널 닫기 (워커들이 range를 빠져나오도록)
	close(s.playerRegistrationTasks)

	// 모든 워커 종료 대기
	s.playerRegistrationWg.Wait()
	s.logger.Info("player_registration_workers_stopped")
}
