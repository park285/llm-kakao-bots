package settings

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"

	"go.uber.org/zap"
)

// Settings 는 타입이다.
type Settings struct {
	AlarmAdvanceMinutes int `json:"alarmAdvanceMinutes"`
}

// Service 는 타입이다.
type Service struct {
	filePath string
	logger   *zap.Logger
	mu       sync.RWMutex
	cache    *Settings
}

// NewSettingsService 는 동작을 수행한다.
func NewSettingsService(filePath string, logger *zap.Logger) *Service {
	s := &Service{
		filePath: filePath,
		logger:   logger,
		cache: &Settings{
			AlarmAdvanceMinutes: 5, // Default
		},
	}
	s.load()
	return s
}

func (s *Service) load() {
	f, err := os.Open(s.filePath)
	if err != nil {
		return // Use defaults if file missing
	}
	defer f.Close()

	_ = json.NewDecoder(f).Decode(s.cache)
}

// Get 는 동작을 수행한다.
func (s *Service) Get() Settings {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return *s.cache
}

// Update 는 동작을 수행한다.
func (s *Service) Update(newSettings Settings) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.cache = &newSettings

	f, err := os.Create(s.filePath)
	if err != nil {
		return fmt.Errorf("failed to create settings file: %w", err)
	}
	defer f.Close()

	if err := json.NewEncoder(f).Encode(s.cache); err != nil {
		return fmt.Errorf("failed to write settings: %w", err)
	}
	return nil
}
