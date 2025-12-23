package activity

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"

	"go.uber.org/zap"
)

// LogEntry 는 타입이다.
type LogEntry struct {
	Timestamp time.Time              `json:"timestamp"`
	Type      string                 `json:"type"` // e.g., "command", "auth", "system"
	Summary   string                 `json:"summary"`
	Details   map[string]interface{} `json:"details,omitempty"`
}

// Logger 는 타입이다.
type Logger struct {
	filePath string
	logger   *zap.Logger
	mu       sync.RWMutex
}

// NewActivityLogger 는 동작을 수행한다.
func NewActivityLogger(filePath string, logger *zap.Logger) *Logger {
	return &Logger{
		filePath: filePath,
		logger:   logger,
	}
}

// Log 는 동작을 수행한다.
func (l *Logger) Log(entryType, summary string, details map[string]interface{}) {
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
		l.logger.Error("Failed to open activity log file", zap.Error(err))
		return
	}
	defer f.Close()

	if err := json.NewEncoder(f).Encode(entry); err != nil {
		l.logger.Error("Failed to write activity log", zap.Error(err))
	}
}

// GetRecentLogs 는 동작을 수행한다.
func (l *Logger) GetRecentLogs(limit int) ([]LogEntry, error) {
	l.mu.RLock()
	defer l.mu.RUnlock()

	f, err := os.Open(l.filePath)
	if err != nil {
		if os.IsNotExist(err) {
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
			continue // Skip malformed lines
		}
		logs = append(logs, entry)
	}

	// Just return all logs, let the caller or UI handle truncation/pagination if it gets too large.
	// But enforcing a limit is good practice.
	if len(logs) > limit {
		return logs[len(logs)-limit:], nil
	}
	return logs, nil
}
