package middleware

import (
	"crypto/subtle"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/park285/llm-kakao-bots/mcp-llm-server-go/internal/config"
	"github.com/park285/llm-kakao-bots/mcp-llm-server-go/internal/httperror"
)

// APIKeyAuth 는 API 키 인증 미들웨어다.
func APIKeyAuth(cfg *config.Config) gin.HandlerFunc {
	expected := ""
	if cfg != nil {
		expected = strings.TrimSpace(cfg.HTTPAuth.APIKey)
	}

	return func(c *gin.Context) {
		if expected == "" {
			c.Next()
			return
		}

		if !shouldProtectPath(c.Request.URL.Path) {
			c.Next()
			return
		}

		provided := extractAPIKey(c)
		if provided == "" || subtle.ConstantTimeCompare([]byte(provided), []byte(expected)) != 1 {
			details := map[string]any{"path": c.Request.URL.Path}
			status, payload := httperror.Response(httperror.NewUnauthorized(details), GetRequestID(c))
			c.AbortWithStatusJSON(status, payload)
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
		token := strings.TrimSpace(authValue[7:])
		return token
	}

	return ""
}

func shouldProtectPath(path string) bool {
	return strings.HasPrefix(path, "/api/")
}
