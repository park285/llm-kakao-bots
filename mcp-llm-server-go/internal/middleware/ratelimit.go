package middleware

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/park285/llm-kakao-bots/mcp-llm-server-go/internal/cache"
	"github.com/park285/llm-kakao-bots/mcp-llm-server-go/internal/config"
	"github.com/park285/llm-kakao-bots/mcp-llm-server-go/internal/httperror"
)

// RateLimit 는 요청 제한 미들웨어다.
func RateLimit(cfg *config.Config) gin.HandlerFunc {
	limit := 0
	cacheSize := 0
	cacheTTL := time.Duration(0)
	if cfg != nil {
		limit = cfg.HTTPRateLimit.RequestsPerMinute
		cacheSize = cfg.HTTPRateLimit.CacheSize
		cacheTTL = time.Duration(cfg.HTTPRateLimit.CacheTTLSeconds) * time.Second
	}

	counter := cache.NewTTLCache[string, int](cacheSize, cacheTTL)

	return func(c *gin.Context) {
		if limit <= 0 {
			c.Next()
			return
		}

		if c.Request.Method == http.MethodOptions || !shouldProtectPath(c.Request.URL.Path) {
			c.Next()
			return
		}

		identity := rateLimitIdentity(c)
		window := time.Now().Unix() / 60
		key := fmt.Sprintf("%s:%d", identity, window)

		count, ok := counter.Modify(key, func(current int, _ bool) int { return current + 1 })
		if !ok {
			c.Next()
			return
		}

		if count > limit {
			details := map[string]any{
				"path":             c.Request.URL.Path,
				"identity":         identity,
				"limit_per_minute": limit,
			}
			status, payload := httperror.Response(httperror.NewRateLimitExceeded(details), GetRequestID(c))
			c.AbortWithStatusJSON(status, payload)
			return
		}

		c.Next()
	}
}

func rateLimitIdentity(c *gin.Context) string {
	if key := extractAPIKey(c); key != "" {
		return "key:" + hashKey(key)
	}

	forwarded := strings.TrimSpace(c.GetHeader("X-Forwarded-For"))
	if forwarded != "" {
		ip := strings.TrimSpace(strings.Split(forwarded, ",")[0])
		if ip != "" {
			return "ip:" + ip
		}
	}

	if c.ClientIP() != "" {
		return "ip:" + c.ClientIP()
	}

	return "ip:unknown"
}

func hashKey(value string) string {
	sum := sha256.Sum256([]byte(value))
	encoded := hex.EncodeToString(sum[:])
	if len(encoded) <= 16 {
		return encoded
	}
	return encoded[:16]
}
