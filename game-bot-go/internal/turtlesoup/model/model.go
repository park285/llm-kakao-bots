package model

import (
	"fmt"
	"slices"
	"strings"
	"time"

	domainmodels "github.com/park285/llm-kakao-bots/game-bot-go/internal/domain/models"
)

// PuzzleCategory 는 타입이다.
type PuzzleCategory string

const (
	// PuzzleCategoryMystery 는 상수다.
	PuzzleCategoryMystery PuzzleCategory = "MYSTERY"
)

// ParsePuzzleCategory 는 동작을 수행한다.
func ParsePuzzleCategory(input string) PuzzleCategory {
	normalized := strings.TrimSpace(input)
	if normalized == "" {
		return PuzzleCategoryMystery
	}
	upper := strings.ToUpper(normalized)
	switch PuzzleCategory(upper) {
	case PuzzleCategoryMystery:
		return PuzzleCategoryMystery
	default:
		return PuzzleCategoryMystery
	}
}

// Puzzle 는 타입이다.
type Puzzle struct {
	Title      string         `json:"title"`
	Scenario   string         `json:"scenario"`
	Solution   string         `json:"solution"`
	Category   PuzzleCategory `json:"category"`
	Difficulty int            `json:"difficulty"`
	Hints      []string       `json:"hints,omitempty"`
	CreatedAt  time.Time      `json:"createdAt"`
}

// HistoryEntry 는 타입이다.
type HistoryEntry struct {
	Question string `json:"question"`
	Answer   string `json:"answer"`
}

// GameState 는 타입이다.
type GameState struct {
	SessionID string `json:"sessionId"`
	UserID    string `json:"userId"`
	ChatID    string `json:"chatId"`

	Puzzle        *Puzzle        `json:"puzzle,omitempty"`
	QuestionCount int            `json:"questionCount"`
	History       []HistoryEntry `json:"history,omitempty"`

	HintsUsed      int       `json:"hintsUsed"`
	HintContents   []string  `json:"hintContents,omitempty"`
	Players        []string  `json:"players,omitempty"`
	IsSolved       bool      `json:"isSolved"`
	StartedAt      time.Time `json:"startedAt"`
	LastActivityAt time.Time `json:"lastActivityAt"`
}

// NewInitialState 는 동작을 수행한다.
func NewInitialState(sessionID string, userID string, chatID string, puzzle Puzzle) GameState {
	now := time.Now()
	return GameState{
		SessionID:      sessionID,
		UserID:         userID,
		ChatID:         chatID,
		Puzzle:         &puzzle,
		Players:        []string{userID},
		StartedAt:      now,
		LastActivityAt: now,
	}
}

// UseHint 는 동작을 수행한다.
func (s GameState) UseHint(hintContent string) GameState {
	now := time.Now()
	nextHints := append(slices.Clone(s.HintContents), hintContent)
	return s.copyWith(func(next *GameState) {
		next.HintsUsed = s.HintsUsed + 1
		next.HintContents = nextHints
		next.LastActivityAt = now
	})
}

// AddPlayer 는 동작을 수행한다.
func (s GameState) AddPlayer(playerID string) GameState {
	now := time.Now()
	if slices.Contains(s.Players, playerID) {
		return s
	}
	nextPlayers := append(slices.Clone(s.Players), playerID)
	return s.copyWith(func(next *GameState) {
		next.Players = nextPlayers
		next.LastActivityAt = now
	})
}

// MarkSolved 는 동작을 수행한다.
func (s GameState) MarkSolved() GameState {
	now := time.Now()
	return s.copyWith(func(next *GameState) {
		next.IsSolved = true
		next.LastActivityAt = now
	})
}

// UpdateActivity 는 동작을 수행한다.
func (s GameState) UpdateActivity() GameState {
	now := time.Now()
	return s.copyWith(func(next *GameState) {
		next.LastActivityAt = now
	})
}

func (s GameState) copyWith(mut func(*GameState)) GameState {
	next := s
	mut(&next)
	return next
}

// ValidationResult 는 타입이다.
type ValidationResult string

// ValidationResult 상수 목록.
const (
	// ValidationYes 는 상수다.
	ValidationYes   ValidationResult = "YES"
	ValidationClose ValidationResult = "CLOSE"
	ValidationNo    ValidationResult = "NO"
)

// ParseValidationResult 는 동작을 수행한다.
func ParseValidationResult(input string) (ValidationResult, error) {
	upper := strings.ToUpper(strings.TrimSpace(input))
	switch ValidationResult(upper) {
	case ValidationYes, ValidationClose, ValidationNo:
		return ValidationResult(upper), nil
	default:
		return "", fmt.Errorf("unknown validation result: %q", input)
	}
}

// IsCorrect 는 동작을 수행한다.
func (r ValidationResult) IsCorrect() bool { return r == ValidationYes }

// IsClose 는 동작을 수행한다.
func (r ValidationResult) IsClose() bool { return r == ValidationClose }

// AnswerResult 는 타입이다.
type AnswerResult struct {
	Result        ValidationResult
	QuestionCount int
	HintCount     int
	MaxHints      int
	HintsUsed     []string
	Explanation   string
}

// IsCorrect 는 동작을 수행한다.
func (r AnswerResult) IsCorrect() bool { return r.Result.IsCorrect() }

// IsClose 는 동작을 수행한다.
func (r AnswerResult) IsClose() bool { return r.Result.IsClose() }

// SurrenderResult 는 타입이다.
type SurrenderResult struct {
	Solution  string
	HintsUsed []string
}

// SurrenderVote 는 타입이다.
type SurrenderVote = domainmodels.SurrenderVote

// PendingMessage 는 타입이다.
type PendingMessage = domainmodels.PendingMessage
