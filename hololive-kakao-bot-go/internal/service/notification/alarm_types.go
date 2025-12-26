package notification

import (
	"log/slog"
	"sync"

	"github.com/kapu/hololive-kakao-bot-go/internal/service/cache"
	"github.com/kapu/hololive-kakao-bot-go/internal/service/holodex"
)

// 알람 키 상수 목록.
const (
	// AlarmKeyPrefix: 알림 데이터 Redis 키 접두사
	AlarmKeyPrefix = "alarm:"
	// AlarmRegistryKey: 알림이 설정된 모든 사용자/채팅방 목록을 추적하는 Set 키
	AlarmRegistryKey            = "alarm:registry"
	AlarmChannelRegistryKey     = "alarm:channel_registry"
	ChannelSubscribersKeyPrefix = "alarm:channel_subscribers:"
	MemberNameKey               = "member_names"
	RoomNamesCacheKey           = "alarm:room_names"
	UserNamesCacheKey           = "alarm:user_names"
	NotifiedKeyPrefix           = "notified:"
	NextStreamKeyPrefix         = "alarm:next_stream:"
)

// NotifiedData: 알림 중복 발송 방지를 위해 기록하는 알림 이력 정보
type NotifiedData struct {
	StartScheduled string `json:"start_scheduled"`
	NotifiedAt     string `json:"notified_at"`
	MinutesUntil   int    `json:"minutes_until"`
}

// AlarmService: 방송 알림(Alarm)을 관리하고, 예정된 방송을 주기적으로 체크하여 알림을 발송하는 서비스
type AlarmService struct {
	cache           *cache.Service
	holodex         *holodex.Service
	logger          *slog.Logger
	targetMinutes   []int
	baseConcurrency int  // 기본 동시성
	maxConcurrency  int  // 최대 동시성
	autoscale       bool // 자동 스케일링 활성화
	cacheMutex      sync.RWMutex
}
