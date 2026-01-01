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

// GameSetupService: 게임 시작 전 퍼즐 생성 및 세션 초기화를 담당하는 서비스입니다.
type GameSetupService struct {
	restClient     *llmrest.Client
	puzzleService  *PuzzleService
	sessionManager *GameSessionManager
	logger         *slog.Logger
}

// NewGameSetupService: GameSetupService 인스턴스를 생성합니다.
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

// GameSetupResult: 게임 셋업 결과(GameState 및 Puzzle)를 담는 구조체입니다.
type GameSetupResult struct {
	State  tsmodel.GameState
	Puzzle tsmodel.Puzzle
}

// PrepareNewGame: 새 게임을 준비하고 퍼즐을 생성합니다.
// 기존 세션이 있으면 에러를 반환하고, 해결된 세션은 삭제 후 새로 생성합니다.
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
		return GameSetupResult{}, fmt.Errorf("load session: %w", err)
	}
	if existing != nil {
		if existing.IsSolved {
			if deleteErr := s.sessionManager.Delete(ctx, sessionID); deleteErr != nil {
				return GameSetupResult{}, fmt.Errorf("delete solved session: %w", deleteErr)
			}
		} else {
			return GameSetupResult{}, tserrors.GameAlreadyStartedError{SessionID: sessionID}
		}
	}

	validatedDifficulty := difficulty
	if difficulty == nil {
		defaultDifficulty := tsconfig.PuzzleDefaultDifficulty
		validatedDifficulty = &defaultDifficulty
	} else if *difficulty < tsconfig.PuzzleMinDifficulty || *difficulty > tsconfig.PuzzleMaxDifficulty {
		return GameSetupResult{}, fmt.Errorf(
			"difficulty out of range (%d..%d): %d",
			tsconfig.PuzzleMinDifficulty,
			tsconfig.PuzzleMaxDifficulty,
			*difficulty,
		)
	}

	puzzle, err := s.puzzleService.GeneratePuzzle(ctx, PuzzleGenerationRequest{
		Category:   category,
		Difficulty: validatedDifficulty,
		Theme:      theme,
	}, chatID)
	if err != nil {
		return GameSetupResult{}, fmt.Errorf("generate puzzle: %w", err)
	}

	state := tsmodel.NewInitialState(sessionID, userID, chatID, puzzle)
	if err := s.sessionManager.Save(ctx, state); err != nil {
		return GameSetupResult{}, fmt.Errorf("save session: %w", err)
	}

	return GameSetupResult{State: state, Puzzle: puzzle}, nil
}
