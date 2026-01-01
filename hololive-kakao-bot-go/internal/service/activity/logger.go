package activity

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"sync"
	"time"

	"github.com/goccy/go-json"
)

// LogEntry: 활동 로그의 한 항목을 나타내는 구조체
type LogEntry struct {
	Timestamp time.Time      `json:"timestamp"`
	Type      string         `json:"type"` // e.g., "command", "auth", "system"
	Summary   string         `json:"summary"`
	Details   map[string]any `json:"details,omitempty"`
}

// Logger: 파일 기반의 간단한 활동 로그 기록기
type Logger struct {
	filePath string
	logger   *slog.Logger
	mu       sync.RWMutex
}

// NewActivityLogger: 새로운 활동 로그 기록기를 생성합니다.
func NewActivityLogger(filePath string, logger *slog.Logger) *Logger {
	return &Logger{
		filePath: filePath,
		logger:   logger,
	}
}

// Log: 새로운 활동 로그를 파일에 추가한다. (Thread-safe)
func (l *Logger) Log(entryType, summary string, details map[string]any) {
	entry := LogEntry{
		Timestamp: time.Now(),
		Type:      entryType,
		Summary:   summary,
		Details:   details,
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	f, err := os.OpenFile(l.filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		l.logger.Error("Failed to open activity log file", slog.Any("error", err))
		return
	}
	defer f.Close()

	if err := json.NewEncoder(f).Encode(entry); err != nil {
		l.logger.Error("Failed to write activity log", slog.Any("error", err))
	}
}

// GetRecentLogs: 최근 활동 로그를 조회합니다.
func (l *Logger) GetRecentLogs(limit int) ([]LogEntry, error) {
	l.mu.RLock()
	defer l.mu.RUnlock()

	f, err := os.Open(l.filePath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return []LogEntry{}, nil
		}
		return nil, fmt.Errorf("failed to open activity log: %w", err)
	}
	defer f.Close()

	var logs []LogEntry
	decoder := json.NewDecoder(f)
	for decoder.More() {
		var entry LogEntry
		if err := decoder.Decode(&entry); err != nil {
			continue // 잘못된 형식의 줄은 건너뜀
		}
		logs = append(logs, entry)
	}

	// 전체 로그를 반환함. 너무 커지면 호출자 또는 UI가 잘라서 처리함.
	// 단, limit 적용은 좋은 관행임.
	if len(logs) > limit {
		return logs[len(logs)-limit:], nil
	}
	return logs, nil
}
