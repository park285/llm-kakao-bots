package settings

import (
	"fmt"
	"log/slog"
	"os"
	"sync"

	"github.com/goccy/go-json"
)

// Settings: 봇의 동적 설정을 담는 구조체 (예: 알림 전송 시점)
type Settings struct {
	AlarmAdvanceMinutes int `json:"alarmAdvanceMinutes"`
}

// Service: 설정 파일을 로드하고 관리하며, 변경 시 파일에 실시간으로 반영하는 서비스
type Service struct {
	filePath string
	logger   *slog.Logger
	mu       sync.RWMutex
	cache    *Settings
}

// NewSettingsService: 지정된 파일 경로에서 설정을 로드하여 서비스 인스턴스를 생성합니다.
func NewSettingsService(filePath string, logger *slog.Logger) *Service {
	s := &Service{
		filePath: filePath,
		logger:   logger,
		cache: &Settings{
			AlarmAdvanceMinutes: 5, // 기본값
		},
	}
	s.load()
	return s
}

func (s *Service) load() {
	f, err := os.Open(s.filePath)
	if err != nil {
		return // 파일이 없으면 기본값 사용함
	}
	defer f.Close()

	_ = json.NewDecoder(f).Decode(s.cache)
}

// Get: 현재 메모리에 캐시된 설정 값을 조회한다. (Thread-safe)
func (s *Service) Get() Settings {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return *s.cache
}

// Update: 설정을 업데이트하고 파일에 즉시 영구 저장합니다.
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
