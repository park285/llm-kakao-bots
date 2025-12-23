package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/park285/llm-kakao-bots/game-bot-go/internal/common/llmrest"
	"github.com/park285/llm-kakao-bots/game-bot-go/internal/common/ptr"
	tsconfig "github.com/park285/llm-kakao-bots/game-bot-go/internal/turtlesoup/config"
	tserrors "github.com/park285/llm-kakao-bots/game-bot-go/internal/turtlesoup/errors"
	tsmodel "github.com/park285/llm-kakao-bots/game-bot-go/internal/turtlesoup/model"
	tsredis "github.com/park285/llm-kakao-bots/game-bot-go/internal/turtlesoup/redis"
)

// PuzzleService 는 타입이다.
type PuzzleService struct {
	restClient *llmrest.Client
	cfg        tsconfig.PuzzleConfig
	dedupStore *tsredis.PuzzleDedupStore
	logger     *slog.Logger
}

// NewPuzzleService 는 동작을 수행한다.
func NewPuzzleService(restClient *llmrest.Client, cfg tsconfig.PuzzleConfig, dedupStore *tsredis.PuzzleDedupStore, logger *slog.Logger) *PuzzleService {
	return &PuzzleService{
		restClient: restClient,
		cfg:        cfg,
		dedupStore: dedupStore,
		logger:     logger,
	}
}

// PuzzleGenerationRequest 는 타입이다.
type PuzzleGenerationRequest struct {
	Category   *tsmodel.PuzzleCategory
	Difficulty *int
	Theme      *string
}

// GeneratePuzzle 는 동작을 수행한다.
func (s *PuzzleService) GeneratePuzzle(ctx context.Context, req PuzzleGenerationRequest, chatID string) (tsmodel.Puzzle, error) {
	category := tsmodel.PuzzleCategoryMystery
	if req.Category != nil {
		category = *req.Category
	}

	difficulty := tsconfig.PuzzleDefaultDifficulty
	if req.Difficulty != nil {
		difficulty = *req.Difficulty
	}
	difficulty = clampInt(difficulty, tsconfig.PuzzleMinDifficulty, tsconfig.PuzzleMaxDifficulty)

	theme := ""
	if req.Theme != nil {
		theme = strings.TrimSpace(*req.Theme)
	}

	var lastErr error

	for attempt := 0; attempt < tsconfig.PuzzleDedupMaxGenerationRetries; attempt++ {
		puzzle, err := s.tryGeneratePuzzle(ctx, chatID, category, difficulty, theme, attempt)
		if err != nil {
			lastErr = err
			continue
		}
		return puzzle, nil
	}

	fallback, err := s.getPresetPuzzleByDifficulty(ctx, difficulty)
	if err != nil {
		if lastErr != nil {
			return tsmodel.Puzzle{}, tserrors.PuzzleGenerationError{Err: lastErr}
		}
		return tsmodel.Puzzle{}, err
	}

	signature := computeSignature(fallback)
	_ = s.dedupStore.MarkUsed(ctx, signature, chatID)

	if lastErr != nil {
		s.logger.Info("puzzle_fallback_preset", "reason", "generate_failed", "chat_id", chatID)
	} else {
		s.logger.Info("puzzle_fallback_preset", "reason", "duplicate_exhausted", "chat_id", chatID)
	}

	return fallback, nil
}

func (s *PuzzleService) tryGeneratePuzzle(
	ctx context.Context,
	chatID string,
	category tsmodel.PuzzleCategory,
	difficulty int,
	theme string,
	attempt int,
) (tsmodel.Puzzle, error) {
	req := llmrest.TurtleSoupPuzzleGenerationRequest{
		Category:   ptr.String(string(category)),
		Difficulty: &difficulty,
	}
	if theme != "" {
		req.Theme = &theme
	}

	res, err := s.restClient.TurtleSoupGeneratePuzzle(ctx, req)
	if err != nil {
		s.logger.Warn("puzzle_generate_failed", "attempt", attempt+1, "chat_id", chatID, "err", err)
		return tsmodel.Puzzle{}, tserrors.PuzzleGenerationError{Err: err}
	}

	puzzle := puzzleFromGeneration(res)
	if !hasRequiredContent(puzzle) {
		s.logger.Warn("puzzle_invalid_empty_fields", "attempt", attempt+1, "chat_id", chatID)
		return tsmodel.Puzzle{}, tserrors.PuzzleGenerationError{Err: fmt.Errorf("empty fields")}
	}

	signature := computeSignature(puzzle)
	dup, err := s.dedupStore.IsDuplicate(ctx, signature, chatID)
	if err != nil {
		s.logger.Warn("puzzle_dedup_check_failed", "attempt", attempt+1, "chat_id", chatID, "err", err)
		return tsmodel.Puzzle{}, fmt.Errorf("puzzle dedup check failed: %w", err)
	}
	if dup {
		s.logger.Warn("puzzle_duplicate_detected", "attempt", attempt+1, "chat_id", chatID)
		return tsmodel.Puzzle{}, tserrors.PuzzleGenerationError{Err: fmt.Errorf("duplicate")}
	}

	if err := s.dedupStore.MarkUsed(ctx, signature, chatID); err != nil {
		s.logger.Warn("puzzle_dedup_mark_failed", "attempt", attempt+1, "chat_id", chatID, "err", err)
		return tsmodel.Puzzle{}, fmt.Errorf("puzzle dedup mark failed: %w", err)
	}

	s.logger.Info("puzzle_generated", "chat_id", chatID, "difficulty", puzzle.Difficulty)
	return puzzle, nil
}

func (s *PuzzleService) getPresetPuzzleByDifficulty(ctx context.Context, difficulty int) (tsmodel.Puzzle, error) {
	difficulty = clampInt(difficulty, tsconfig.PuzzleMinDifficulty, tsconfig.PuzzleMaxDifficulty)

	preset, err := s.restClient.TurtleSoupGetRandomPuzzle(ctx, &difficulty)
	if err != nil {
		return tsmodel.Puzzle{}, tserrors.PuzzleGenerationError{Err: err}
	}

	basePuzzle := puzzleFromPreset(preset)
	return s.applyRewriteIfEnabled(ctx, basePuzzle), nil
}

func (s *PuzzleService) applyRewriteIfEnabled(ctx context.Context, puzzle tsmodel.Puzzle) tsmodel.Puzzle {
	if !s.cfg.RewriteEnabled {
		s.logger.Info("preset_puzzle_selected", "title", puzzle.Title, "rewrite", false)
		return puzzle
	}

	s.logger.Info("rewriting_puzzle", "title", puzzle.Title)

	rewritten, err := s.restClient.TurtleSoupRewriteScenario(ctx, puzzle.Title, puzzle.Scenario, puzzle.Solution, puzzle.Difficulty)
	if err != nil {
		s.logger.Warn("rewrite_failed_using_original", "title", puzzle.Title, "err", err)
		return puzzle
	}

	puzzle.Scenario = rewritten.Scenario
	puzzle.Solution = rewritten.Solution
	s.logger.Info("puzzle_rewritten", "title", puzzle.Title)
	return puzzle
}

func puzzleFromGeneration(res *llmrest.TurtleSoupPuzzleGenerationResponse) tsmodel.Puzzle {
	category := tsmodel.ParsePuzzleCategory(res.Category)
	difficulty := clampInt(res.Difficulty, tsconfig.PuzzleMinDifficulty, tsconfig.PuzzleMaxDifficulty)

	return tsmodel.Puzzle{
		Title:      strings.TrimSpace(res.Title),
		Scenario:   strings.TrimSpace(res.Scenario),
		Solution:   strings.TrimSpace(res.Solution),
		Category:   category,
		Difficulty: difficulty,
		Hints:      res.Hints,
		CreatedAt:  timeNow(),
	}
}

func puzzleFromPreset(res *llmrest.TurtleSoupPuzzlePresetResponse) tsmodel.Puzzle {
	title := ""
	if res.Title != nil {
		title = strings.TrimSpace(*res.Title)
	}
	question := ""
	if res.Question != nil {
		question = strings.TrimSpace(*res.Question)
	}
	answer := ""
	if res.Answer != nil {
		answer = strings.TrimSpace(*res.Answer)
	}
	difficulty := tsconfig.PuzzleDefaultDifficulty
	if res.Difficulty != nil {
		difficulty = *res.Difficulty
	}
	difficulty = clampInt(difficulty, tsconfig.PuzzleMinDifficulty, tsconfig.PuzzleMaxDifficulty)

	return tsmodel.Puzzle{
		Title:      title,
		Scenario:   question,
		Solution:   answer,
		Category:   tsmodel.PuzzleCategoryMystery,
		Difficulty: difficulty,
		CreatedAt:  timeNow(),
	}
}

func hasRequiredContent(puzzle tsmodel.Puzzle) bool {
	return strings.TrimSpace(puzzle.Title) != "" && strings.TrimSpace(puzzle.Scenario) != "" && strings.TrimSpace(puzzle.Solution) != ""
}

func computeSignature(puzzle tsmodel.Puzzle) string {
	normalizedParts := []string{
		strings.ToLower(strings.TrimSpace(puzzle.Title)),
		strings.ToLower(strings.TrimSpace(puzzle.Scenario)),
		strings.ToLower(strings.TrimSpace(puzzle.Solution)),
		strings.ToLower(strings.TrimSpace(strings.Join(puzzle.Hints, "|"))),
	}
	normalized := strings.Join(normalizedParts, "|")

	sum := sha256.Sum256([]byte(normalized))
	return hex.EncodeToString(sum[:])
}

func clampInt(v int, minValue int, maxValue int) int {
	if v < minValue {
		return minValue
	}
	if v > maxValue {
		return maxValue
	}
	return v
}

var timeNow = time.Now
