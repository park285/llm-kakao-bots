// Package metrics: Prometheus 메트릭 엔드포인트 보호 유틸리티
package metrics

import (
	"crypto/subtle"
	"strings"

	"github.com/gin-gonic/gin"
)

// APIKeyAuth: /metrics 보호를 위한 간단한 API 키 인증 미들웨어입니다.
// - METRICS_API_KEY가 비어있으면 보호하지 않습니다(하위 호환/내부망 전제).
// - 헤더는 Authorization: Bearer <token> 또는 X-API-Key: <token> 를 지원합니다.
func APIKeyAuth(expected string) gin.HandlerFunc {
	expected = strings.TrimSpace(expected)

	return func(c *gin.Context) {
		if expected == "" {
			c.Next()
			return
		}

		provided := extractAPIKey(c)
		if provided == "" || subtle.ConstantTimeCompare([]byte(provided), []byte(expected)) != 1 {
			c.AbortWithStatusJSON(401, gin.H{"error": "Unauthorized"})
			return
		}

		c.Next()
	}
}

func extractAPIKey(c *gin.Context) string {
	if c == nil {
		return ""
	}

	value := strings.TrimSpace(c.GetHeader("X-API-Key"))
	if value != "" {
		return value
	}

	authValue := strings.TrimSpace(c.GetHeader("Authorization"))
	if authValue == "" {
		return ""
	}

	if strings.HasPrefix(strings.ToLower(authValue), "bearer ") {
		return strings.TrimSpace(authValue[7:])
	}

	return ""
}
