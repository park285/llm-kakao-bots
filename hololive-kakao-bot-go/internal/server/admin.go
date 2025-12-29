package server

import (
	"log/slog"
	"time"

	"github.com/kapu/hololive-kakao-bot-go/internal/config"
	"github.com/kapu/hololive-kakao-bot-go/internal/service/acl"
	"github.com/kapu/hololive-kakao-bot-go/internal/service/activity"
	"github.com/kapu/hololive-kakao-bot-go/internal/service/cache"
	"github.com/kapu/hololive-kakao-bot-go/internal/service/holodex"
	"github.com/kapu/hololive-kakao-bot-go/internal/service/member"
	"github.com/kapu/hololive-kakao-bot-go/internal/service/notification"
	"github.com/kapu/hololive-kakao-bot-go/internal/service/settings"
	"github.com/kapu/hololive-kakao-bot-go/internal/service/youtube"
)

// AdminHandler: 관리자 API 요청을 처리하는 핸들러입니다.
// 핸들러 메서드는 도메인별 파일로 분리됨:
//   - admin_auth.go: 인증 (로그인/로그아웃)
//   - admin_member.go: 멤버 관리
//   - admin_alarm.go: 알람 관리
//   - admin_room.go: 룸/ACL 관리
//   - admin_stream.go: 스트림/채널 통계
//   - admin_stats.go: 봇 통계
//   - admin_settings.go: 설정/로그/이름매핑
type AdminHandler struct {
	repo          *member.Repository
	memberCache   *member.Cache
	valkeyCache   *cache.Service
	alarm         *notification.AlarmService
	holodex       *holodex.Service
	youtube       *youtube.Service
	activity      *activity.Logger
	settings      *settings.Service
	acl           *acl.Service
	config        *config.Config
	sessions      SessionProvider
	rateLimiter   *LoginRateLimiter
	securityCfg   *SecurityConfig
	adminUser     string
	adminPassHash string
	logger        *slog.Logger
	startTime     time.Time
}

// NewAdminHandler: 새로운 관리자 핸들러를 생성합니다.
func NewAdminHandler(
	repo *member.Repository,
	memberCache *member.Cache,
	valkeyCache *cache.Service,
	alarm *notification.AlarmService,
	holodexSvc *holodex.Service,
	youtubeSvc *youtube.Service,
	activityLogger *activity.Logger,
	settingsSvc *settings.Service,
	aclSvc *acl.Service,
	cfg *config.Config,
	sessions SessionProvider,
	rateLimiter *LoginRateLimiter,
	securityCfg *SecurityConfig,
	adminUser, adminPassHash string,
	logger *slog.Logger,
) *AdminHandler {
	return &AdminHandler{
		repo:          repo,
		memberCache:   memberCache,
		valkeyCache:   valkeyCache,
		alarm:         alarm,
		holodex:       holodexSvc,
		youtube:       youtubeSvc,
		activity:      activityLogger,
		settings:      settingsSvc,
		acl:           aclSvc,
		config:        cfg,
		sessions:      sessions,
		rateLimiter:   rateLimiter,
		securityCfg:   securityCfg,
		adminUser:     adminUser,
		adminPassHash: adminPassHash,
		logger:        logger,
		startTime:     time.Now(),
	}
}
