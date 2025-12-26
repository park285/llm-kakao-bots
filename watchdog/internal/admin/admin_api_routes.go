package admin

import (
	"log/slog"

	"github.com/gin-gonic/gin"
	watchdog "llm-watchdog/internal/core"
)

// 요청 타입 정의
type restartRequest struct {
	Reason string `json:"reason"`
	Force  bool   `json:"force"`
}

type stopRequest struct {
	TimeoutSeconds int    `json:"timeoutSeconds"`
	Reason         string `json:"reason"`
}

type startRequest struct {
	Reason string `json:"reason"`
}

type managedRequest struct {
	Managed *bool  `json:"managed"`
	Reason  string `json:"reason"`
}

// registerAdminAPIRoutes Admin API 라우트 등록
// 핸들러는 도메인별 파일로 분리됨:
//   - handler_watchdog.go: watchdog 상태/설정
//   - handler_targets.go: 컨테이너 타겟 관리
//   - handler_docker.go: Docker 컨테이너, 로그, 이벤트
func registerAdminAPIRoutes(router *gin.Engine, w *watchdog.Watchdog, logger *slog.Logger, middlewares ...gin.HandlerFunc) {
	api := router.Group("/admin/api/v1", middlewares...)
	api.Use(noCacheHeaders)

	// 도메인별 핸들러 등록
	registerWatchdogHandlers(api, w, logger)
	registerTargetHandlers(api, w, logger)
	registerDockerHandlers(api, w, logger)
}
