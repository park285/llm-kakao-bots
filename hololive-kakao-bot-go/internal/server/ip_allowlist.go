package server

import (
	"fmt"
	"net"
	"strings"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/kapu/hololive-kakao-bot-go/internal/util"
)

// NewIPAllowList 는 동작을 수행한다.
func NewIPAllowList(allowed []string) ([]*net.IPNet, error) {
	nets := make([]*net.IPNet, 0, len(allowed))
	for _, raw := range allowed {
		trimmed := util.TrimSpace(raw)
		if trimmed == "" {
			continue
		}
		if !strings.Contains(trimmed, "/") {
			if strings.Contains(trimmed, ":") {
				trimmed += "/128"
			} else {
				trimmed += "/32"
			}
		}
		_, cidr, err := net.ParseCIDR(trimmed)
		if err != nil {
			return nil, fmt.Errorf("invalid CIDR %s: %w", trimmed, err)
		}
		nets = append(nets, cidr)
	}
	return nets, nil
}

// AdminIPAllowMiddleware 는 동작을 수행한다.
func AdminIPAllowMiddleware(allowed []*net.IPNet, logger *zap.Logger) gin.HandlerFunc {
	if len(allowed) == 0 {
		return func(c *gin.Context) {
			c.Next()
		}
	}
	return func(c *gin.Context) {
		clientIP := net.ParseIP(c.ClientIP())
		if clientIP == nil {
			logger.Warn("Invalid client IP")
			c.JSON(403, gin.H{"error": "forbidden"})
			c.Abort()
			return
		}
		for _, cidr := range allowed {
			if cidr.Contains(clientIP) {
				c.Next()
				return
			}
		}
		logger.Warn("Admin IP blocked", zap.String("ip", clientIP.String()))
		c.JSON(403, gin.H{"error": "forbidden"})
		c.Abort()
	}
}
