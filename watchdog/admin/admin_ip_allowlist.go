package admin

import (
	"fmt"
	"net"
	"net/http"
	"net/netip"
	"strings"

	"github.com/gin-gonic/gin"
)

type ipAllowlist struct {
	exact map[netip.Addr]struct{}
	cidrs []netip.Prefix
}

func newIPAllowlist(raw []string) (*ipAllowlist, error) {
	if len(raw) == 0 {
		return nil, nil
	}

	allowlist := &ipAllowlist{
		exact: make(map[netip.Addr]struct{}),
	}

	for _, item := range raw {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		if strings.Contains(item, "/") {
			prefix, err := netip.ParsePrefix(item)
			if err != nil {
				return nil, fmt.Errorf("invalid cidr %q: %w", item, err)
			}
			allowlist.cidrs = append(allowlist.cidrs, prefix)
			continue
		}
		addr, err := netip.ParseAddr(item)
		if err != nil {
			return nil, fmt.Errorf("invalid ip %q: %w", item, err)
		}
		allowlist.exact[addr] = struct{}{}
	}

	if len(allowlist.exact) == 0 && len(allowlist.cidrs) == 0 {
		return nil, nil
	}
	return allowlist, nil
}

func (l *ipAllowlist) allows(addr netip.Addr) bool {
	if l == nil || !addr.IsValid() {
		return false
	}
	if _, ok := l.exact[addr]; ok {
		return true
	}
	for _, prefix := range l.cidrs {
		if prefix.Contains(addr) {
			return true
		}
	}
	return false
}

func (l *ipAllowlist) middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		addr, ok := clientIPFromRequest(c)
		if !ok {
			writeAPIError(c, http.StatusForbidden, "ip_unknown", "클라이언트 IP를 확인할 수 없습니다.")
			return
		}
		if !l.allows(addr) {
			writeAPIError(c, http.StatusForbidden, "ip_forbidden", "허용되지 않은 IP입니다.")
			return
		}
		c.Set("admin_email", fmt.Sprintf("ip:%s", addr.String()))
		c.Next()
	}
}

func clientIPFromRequest(c *gin.Context) (netip.Addr, bool) {
	if addr, ok := parseIPCandidate(c.GetHeader("CF-Connecting-IP")); ok {
		return addr, true
	}
	if addr, ok := parseIPCandidate(firstForwardedIP(c.GetHeader("X-Forwarded-For"))); ok {
		return addr, true
	}
	if addr, ok := parseIPCandidate(c.GetHeader("X-Real-IP")); ok {
		return addr, true
	}
	if host, _, err := net.SplitHostPort(strings.TrimSpace(c.Request.RemoteAddr)); err == nil {
		if addr, ok := parseIPCandidate(host); ok {
			return addr, true
		}
	}
	return parseIPCandidate(c.Request.RemoteAddr)
}

func firstForwardedIP(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	parts := strings.Split(raw, ",")
	return strings.TrimSpace(parts[0])
}

func parseIPCandidate(raw string) (netip.Addr, bool) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return netip.Addr{}, false
	}
	if host, _, err := net.SplitHostPort(value); err == nil {
		return parseIPCandidate(host)
	}
	value = strings.TrimPrefix(value, "[")
	value = strings.TrimSuffix(value, "]")
	if idx := strings.Index(value, "%"); idx >= 0 {
		value = value[:idx]
	}
	if addr, err := netip.ParseAddr(value); err == nil {
		return addr, true
	}
	return netip.Addr{}, false
}
