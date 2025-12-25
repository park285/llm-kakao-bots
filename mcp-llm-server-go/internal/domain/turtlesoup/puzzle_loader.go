package turtlesoup

import (
	"cmp"
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"math/rand/v2"
	"slices"
	"sync"
	"time"

	"github.com/goccy/go-json"
)

//go:embed puzzles/*.json
var puzzlesFS embed.FS

// PuzzlePreset 은 퍼즐 기본 데이터다.
type PuzzlePreset struct {
	ID         int    `json:"id"`
	Title      string `json:"title"`
	Question   string `json:"question"`
	Answer     string `json:"answer"`
	Difficulty int    `json:"difficulty"`
}

// PuzzleLoader 는 퍼즐 데이터를 로드하고 조회한다.
type PuzzleLoader struct {
	mu           sync.RWMutex
	all          []PuzzlePreset
	byDifficulty map[int][]PuzzlePreset
	byID         map[int]PuzzlePreset
	rnd          *rand.Rand
}

// NewPuzzleLoader 는 퍼즐 로더를 초기화한다.
func NewPuzzleLoader() (*PuzzleLoader, error) {
	loader := &PuzzleLoader{
		byDifficulty: make(map[int][]PuzzlePreset),
		byID:         make(map[int]PuzzlePreset),
		rnd:          rand.New(rand.NewPCG(uint64(time.Now().UnixNano()), 0)),
	}
	if _, err := loader.reloadLocked(); err != nil {
		return nil, err
	}
	return loader, nil
}

// Reload 는 퍼즐 데이터를 다시 로드한다.
func (l *PuzzleLoader) Reload() (int, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.reloadLocked()
}

// All 은 모든 퍼즐을 반환한다.
func (l *PuzzleLoader) All() []PuzzlePreset {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return append([]PuzzlePreset(nil), l.all...)
}

// CountByDifficulty 는 난이도별 퍼즐 개수를 반환한다.
func (l *PuzzleLoader) CountByDifficulty() map[int]int {
	l.mu.RLock()
	defer l.mu.RUnlock()
	counts := make(map[int]int, len(l.byDifficulty))
	for difficulty, puzzles := range l.byDifficulty {
		counts[difficulty] = len(puzzles)
	}
	return counts
}

// GetRandomPuzzle 는 랜덤 퍼즐을 반환한다.
func (l *PuzzleLoader) GetRandomPuzzle() (PuzzlePreset, error) {
	l.mu.RLock()
	defer l.mu.RUnlock()
	if len(l.all) == 0 {
		return PuzzlePreset{}, errors.New("no puzzles loaded")
	}
	return l.all[l.rnd.IntN(len(l.all))], nil
}

// GetRandomPuzzleByDifficulty 는 난이도 기준 랜덤 퍼즐을 반환한다.
func (l *PuzzleLoader) GetRandomPuzzleByDifficulty(difficulty int) (PuzzlePreset, error) {
	l.mu.RLock()
	defer l.mu.RUnlock()
	puzzles := l.byDifficulty[difficulty]
	if len(puzzles) == 0 {
		return PuzzlePreset{}, fmt.Errorf("no puzzle for difficulty %d", difficulty)
	}
	return puzzles[l.rnd.IntN(len(puzzles))], nil
}

// GetPuzzleByID 는 ID 로 퍼즐을 조회한다.
func (l *PuzzleLoader) GetPuzzleByID(id int) (PuzzlePreset, bool) {
	l.mu.RLock()
	defer l.mu.RUnlock()
	p, ok := l.byID[id]
	return p, ok
}

// GetExamples 는 난이도 기준 예시 퍼즐을 반환한다.
func (l *PuzzleLoader) GetExamples(difficulty int, maxExamples int) []PuzzlePreset {
	l.mu.RLock()
	defer l.mu.RUnlock()
	if maxExamples <= 0 {
		return nil
	}
	candidates := l.byDifficulty[difficulty]
	if len(candidates) == 0 {
		candidates = l.all
	}
	if len(candidates) == 0 {
		return nil
	}

	if len(candidates) <= maxExamples {
		return append([]PuzzlePreset(nil), candidates...)
	}

	indexes := l.rnd.Perm(len(candidates))[:maxExamples]
	slices.Sort(indexes)
	out := make([]PuzzlePreset, 0, len(indexes))
	for _, idx := range indexes {
		out = append(out, candidates[idx])
	}
	return out
}

func (l *PuzzleLoader) reloadLocked() (int, error) {
	paths, err := fs.Glob(puzzlesFS, "puzzles/*.json")
	if err != nil {
		return 0, fmt.Errorf("glob puzzles: %w", err)
	}
	slices.SortFunc(paths, cmp.Compare)

	combined := make([]PuzzlePreset, 0)
	for _, path := range paths {
		data, err := fs.ReadFile(puzzlesFS, path)
		if err != nil {
			return 0, fmt.Errorf("read puzzle file: %w", err)
		}
		var parsed []PuzzlePreset
		if err := json.Unmarshal(data, &parsed); err != nil {
			return 0, fmt.Errorf("decode puzzle file: %w", err)
		}
		combined = append(combined, parsed...)
	}

	byDifficulty := make(map[int][]PuzzlePreset)
	byID := make(map[int]PuzzlePreset)
	for _, puzzle := range combined {
		byDifficulty[puzzle.Difficulty] = append(byDifficulty[puzzle.Difficulty], puzzle)
		byID[puzzle.ID] = puzzle
	}

	l.all = combined
	l.byDifficulty = byDifficulty
	l.byID = byID
	return len(l.all), nil
}
