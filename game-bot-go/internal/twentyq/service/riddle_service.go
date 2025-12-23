package service

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync"

	"github.com/park285/llm-kakao-bots/game-bot-go/internal/common/llmrest"
	"github.com/park285/llm-kakao-bots/game-bot-go/internal/common/messageprovider"
	qredis "github.com/park285/llm-kakao-bots/game-bot-go/internal/twentyq/redis"
)

// RiddleService 는 타입이다.
type RiddleService struct {
	restClient    *llmrest.Client
	commandPrefix string
	msgProvider   *messageprovider.Provider

	lockManager *qredis.LockManager

	sessionStore      *qredis.SessionStore
	categoryStore     *qredis.CategoryStore
	historyStore      *qredis.HistoryStore
	hintCountStore    *qredis.HintCountStore
	playerStore       *qredis.PlayerStore
	wrongGuessStore   *qredis.WrongGuessStore
	topicHistoryStore *qredis.TopicHistoryStore
	voteStore         *qredis.SurrenderVoteStore

	topicSelector *TopicSelector
	statsRecorder *StatsRecorder
	logger        *slog.Logger

	playerRegistrationOnce  sync.Once
	playerRegistrationTasks chan playerRegistrationTask
}

// NewRiddleService 는 동작을 수행한다.
func NewRiddleService(
	restClient *llmrest.Client,
	commandPrefix string,
	msgProvider *messageprovider.Provider,
	lockManager *qredis.LockManager,
	sessionStore *qredis.SessionStore,
	categoryStore *qredis.CategoryStore,
	historyStore *qredis.HistoryStore,
	hintCountStore *qredis.HintCountStore,
	playerStore *qredis.PlayerStore,
	wrongGuessStore *qredis.WrongGuessStore,
	topicHistoryStore *qredis.TopicHistoryStore,
	voteStore *qredis.SurrenderVoteStore,
	topicSelector *TopicSelector,
	statsRecorder *StatsRecorder,
	logger *slog.Logger,
) *RiddleService {
	svc := &RiddleService{
		restClient:        restClient,
		commandPrefix:     strings.TrimSpace(commandPrefix),
		msgProvider:       msgProvider,
		lockManager:       lockManager,
		sessionStore:      sessionStore,
		categoryStore:     categoryStore,
		historyStore:      historyStore,
		hintCountStore:    hintCountStore,
		playerStore:       playerStore,
		wrongGuessStore:   wrongGuessStore,
		topicHistoryStore: topicHistoryStore,
		voteStore:         voteStore,
		topicSelector:     topicSelector,
		statsRecorder:     statsRecorder,
		logger:            logger,
	}
	return svc
}

// HasSession 세션 존재 여부 확인.
func (s *RiddleService) HasSession(ctx context.Context, chatID string) (bool, error) {
	chatID = strings.TrimSpace(chatID)
	if chatID == "" {
		return false, nil
	}
	exists, err := s.sessionStore.Exists(ctx, chatID)
	if err != nil {
		return false, fmt.Errorf("session exists check failed: %w", err)
	}
	return exists, nil
}

// RegisterPlayer 는 동작을 수행한다.
func (s *RiddleService) RegisterPlayer(ctx context.Context, chatID string, userID string, sender *string) error {
	chatID = strings.TrimSpace(chatID)
	userID = strings.TrimSpace(userID)
	if chatID == "" || userID == "" {
		return nil
	}

	senderText := ""
	if sender != nil {
		senderText = strings.TrimSpace(*sender)
	}

	isNew, err := s.playerStore.Add(ctx, chatID, userID, senderText)
	if err != nil {
		return fmt.Errorf("player store add failed: %w", err)
	}

	if isNew && s.statsRecorder != nil {
		s.statsRecorder.RecordGameStart(ctx, chatID, userID)
	}
	return nil
}
