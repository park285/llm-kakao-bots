package service

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/park285/llm-kakao-bots/game-bot-go/internal/common/llmrest"
	tsconfig "github.com/park285/llm-kakao-bots/game-bot-go/internal/turtlesoup/config"
	tserrors "github.com/park285/llm-kakao-bots/game-bot-go/internal/turtlesoup/errors"
	tsmodel "github.com/park285/llm-kakao-bots/game-bot-go/internal/turtlesoup/model"
)

// GameSetupService 는 타입이다.
type GameSetupService struct {
	restClient     *llmrest.Client
	puzzleService  *PuzzleService
	sessionManager *GameSessionManager
	logger         *slog.Logger
}

// NewGameSetupService 는 동작을 수행한다.
func NewGameSetupService(
	restClient *llmrest.Client,
	puzzleService *PuzzleService,
	sessionManager *GameSessionManager,
	logger *slog.Logger,
) *GameSetupService {
	return &GameSetupService{
		restClient:     restClient,
		puzzleService:  puzzleService,
		sessionManager: sessionManager,
		logger:         logger,
	}
}

// GameSetupResult 는 타입이다.
type GameSetupResult struct {
	State  tsmodel.GameState
	Puzzle tsmodel.Puzzle
}

// PrepareNewGame 는 동작을 수행한다.
func (s *GameSetupService) PrepareNewGame(
	ctx context.Context,
	sessionID string,
	userID string,
	chatID string,
	difficulty *int,
	category *tsmodel.PuzzleCategory,
	theme *string,
) (GameSetupResult, error) {
	existing, err := s.sessionManager.Load(ctx, sessionID)
	if err != nil {
		return GameSetupResult{}, err
	}
	if existing != nil {
		if existing.IsSolved {
			if deleteErr := s.sessionManager.Delete(ctx, sessionID); deleteErr != nil {
				return GameSetupResult{}, deleteErr
			}
		} else {
			return GameSetupResult{}, tserrors.GameAlreadyStartedError{SessionID: sessionID}
		}
	}

	puzzle, err := s.puzzleService.GeneratePuzzle(ctx, PuzzleGenerationRequest{
		Category:   category,
		Difficulty: difficulty,
		Theme:      theme,
	}, chatID)
	if err != nil {
		return GameSetupResult{}, err
	}

	if _, createErr := s.restClient.CreateSession(ctx, chatID, tsconfig.LlmNamespace); createErr != nil {
		return GameSetupResult{}, fmt.Errorf("create llm session failed: %w", createErr)
	}
	s.logger.Info("llm_session_created", "chat_id", chatID)

	state := tsmodel.NewInitialState(sessionID, userID, chatID, puzzle)
	if err := s.sessionManager.Save(ctx, state); err != nil {
		return GameSetupResult{}, err
	}

	return GameSetupResult{State: state, Puzzle: puzzle}, nil
}
