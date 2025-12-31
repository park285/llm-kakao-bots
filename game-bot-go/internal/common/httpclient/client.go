package httpclient

import (
	"context"
	"crypto/tls"
	"net"
	"net/http"
	"time"

	"golang.org/x/net/http2"
)

// Config: HTTP 클라이언트 생성 설정입니다.
// HTTP2Enabled=true인 경우 H2C(HTTP/2 Cleartext) 통신을 위해 http2.Transport를 사용합니다.
type Config struct {
	Timeout        time.Duration
	ConnectTimeout time.Duration
	HTTP2Enabled   bool
}

// New: 설정에 따라 *http.Client를 생성합니다.
func New(cfg Config) *http.Client {
	dialer := &net.Dialer{
		Timeout:   cfg.ConnectTimeout,
		KeepAlive: 30 * time.Second,
	}

	var transport http.RoundTripper
	if cfg.HTTP2Enabled {
		transport = &http2.Transport{
			AllowHTTP: true,
			DialTLSContext: func(ctx context.Context, network, addr string, _ *tls.Config) (net.Conn, error) {
				return dialer.DialContext(ctx, network, addr)
			},
		}
	} else {
		transport = &http.Transport{
			Proxy:                 http.ProxyFromEnvironment,
			DialContext:           dialer.DialContext,
			ForceAttemptHTTP2:     false,
			MaxIdleConns:          100,
			MaxIdleConnsPerHost:   100,
			IdleConnTimeout:       90 * time.Second,
			TLSHandshakeTimeout:   10 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
		}
	}

	return &http.Client{
		Timeout:   cfg.Timeout,
		Transport: transport,
	}
}
