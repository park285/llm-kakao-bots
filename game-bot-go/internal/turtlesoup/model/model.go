package model

import (
	"fmt"
	"slices"
	"strings"
	"time"

	domainmodels "github.com/park285/llm-kakao-bots/game-bot-go/internal/domain/models"
)

// PuzzleCategory: 퍼즐 카테고리 타입
type PuzzleCategory string

const (
	// PuzzleCategoryMystery: 기본 미스터리 카테고리
	PuzzleCategoryMystery PuzzleCategory = "MYSTERY"
)

// ParsePuzzleCategory: 문자열을 PuzzleCategory로 변환한다.
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

// Puzzle: 바다거북 스푸 게임의 문제(시나리오)와 정답(해설)을 담고 있는 구조체
type Puzzle struct {
	Title      string         `json:"title"`
	Scenario   string         `json:"scenario"`
	Solution   string         `json:"solution"`
	Category   PuzzleCategory `json:"category"`
	Difficulty int            `json:"difficulty"`
	Hints      []string       `json:"hints,omitempty"`
	CreatedAt  time.Time      `json:"createdAt"`
}

// HistoryEntry: 질문/답변 기록 항목
type HistoryEntry struct {
	Question string `json:"question"`
	Answer   string `json:"answer"`
}

// GameState: 특정 채팅방의 게임 진행 상황(퍼즐 정보, 질문 카운트, 이력, 플레이어 목록 등)을 저장하는 상태 객체
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

// NewInitialState: 새로운 게임 상태를 초기화한다.
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

// UseHint: 힌트를 사용하고 상태를 업데이트한다. (Immutable)
func (s GameState) UseHint(hintContent string) GameState {
	now := time.Now()
	nextHints := append(slices.Clone(s.HintContents), hintContent)
	return s.copyWith(func(next *GameState) {
		next.HintsUsed = s.HintsUsed + 1
		next.HintContents = nextHints
		next.LastActivityAt = now
	})
}

// AddPlayer: 참여자를 목록에 추가하고 상태를 업데이트한다. (Immutable)
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

// MarkSolved: 게임을 해결됨 상태로 변경한다. (Immutable)
func (s GameState) MarkSolved() GameState {
	now := time.Now()
	return s.copyWith(func(next *GameState) {
		next.IsSolved = true
		next.LastActivityAt = now
	})
}

// UpdateActivity: 마지막 활동 시간을 현재로 갱신한다. (Immutable)
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

// ValidationResult: 사용자의 정답 시도에 대한 AI의 판정 결과 (예, 아니오, 근접함 등)
type ValidationResult string

// ValidationResult 상수 목록.
const (
	// ValidationYes: 정답
	ValidationYes   ValidationResult = "YES"
	ValidationClose ValidationResult = "CLOSE"
	ValidationNo    ValidationResult = "NO"
)

// ParseValidationResult: 문자열을 ValidationResult로 변환한다.
func ParseValidationResult(input string) (ValidationResult, error) {
	upper := strings.ToUpper(strings.TrimSpace(input))
	switch ValidationResult(upper) {
	case ValidationYes, ValidationClose, ValidationNo:
		return ValidationResult(upper), nil
	default:
		return "", fmt.Errorf("unknown validation result: %q", input)
	}
}

// IsCorrect: 정답 여부를 반환한다.
func (r ValidationResult) IsCorrect() bool { return r == ValidationYes }

// IsClose: 정답에 근접했는지 여부를 반환한다.
func (r ValidationResult) IsClose() bool { return r == ValidationClose }

// AnswerResult: 정답 제출 후의 상세 결과 (판정, 힌트 사용 내역, 설명 포함)
type AnswerResult struct {
	Result        ValidationResult
	QuestionCount int
	HintCount     int
	MaxHints      int
	HintsUsed     []string
	Explanation   string
}

// IsCorrect: 정답 여부를 반환한다.
func (r AnswerResult) IsCorrect() bool { return r.Result.IsCorrect() }

// IsClose: 정답에 근접했는지 여부를 반환한다.
func (r AnswerResult) IsClose() bool { return r.Result.IsClose() }

// SurrenderResult: 게임 포기(항복) 시 공개되는 정답과 해석 정보
type SurrenderResult struct {
	Solution  string
	HintsUsed []string
}

// SurrenderVote: domainmodels.SurrenderVote alias
type SurrenderVote = domainmodels.SurrenderVote

// PendingMessage: domainmodels.PendingMessage alias
type PendingMessage = domainmodels.PendingMessage
