package valkeyx

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/valkey-io/valkey-go"
)

// Config 는 Valkey 클라이언트 설정이다.
type Config struct {
	Addr         string
	Username     string
	Password     string
	DB           int
	DialTimeout  time.Duration
	ReadTimeout  time.Duration
	WriteTimeout time.Duration

	PoolSize     int
	MinIdleConns int

	// DisableCache 는 클라이언트 사이드 캐싱 비활성화 여부다.
	// 테스트(miniredis)에서는 true로 설정해야 한다.
	DisableCache bool

	// UseTLS 는 TLS 사용 여부다.
	UseTLS bool
}

// NewClient 는 Valkey 클라이언트를 생성한다.
func NewClient(cfg Config) (valkey.Client, error) {
	addr := strings.TrimSpace(cfg.Addr)
	if addr == "" {
		return nil, errors.New("valkey addr is empty")
	}

	var tlsConfig *tls.Config
	if cfg.UseTLS {
		host, _, err := net.SplitHostPort(addr)
		if err != nil {
			host = addr
		}
		tlsConfig = &tls.Config{
			MinVersion: tls.VersionTLS12,
			ServerName: host,
		}
	}

	opts := valkey.ClientOption{
		InitAddress:  []string{addr},
		Username:     cfg.Username,
		Password:     cfg.Password,
		SelectDB:     cfg.DB,
		TLSConfig:    tlsConfig,
		DisableCache: cfg.DisableCache,
	}

	// Timeout 설정
	if cfg.DialTimeout > 0 {
		opts.Dialer.Timeout = cfg.DialTimeout
	}

	client, err := valkey.NewClient(opts)
	if err != nil {
		return nil, fmt.Errorf("create valkey client failed: %w", err)
	}

	return client, nil
}

// Ping 는 Valkey 연결을 확인한다.
func Ping(ctx context.Context, client valkey.Client) error {
	if client == nil {
		return errors.New("valkey client is nil")
	}
	cmd := client.B().Ping().Build()
	if err := client.Do(ctx, cmd).Error(); err != nil {
		return fmt.Errorf("valkey ping failed: %w", err)
	}
	return nil
}

// IsNil 은 Valkey nil 오류인지 확인한다.
// 래핑된 에러도 언래핑하여 체크한다.
func IsNil(err error) bool {
	if valkey.IsValkeyNil(err) {
		return true
	}
	// fmt.Errorf("%w", err)로 래핑된 경우 언래핑하여 체크
	var unwrapped error = err
	for unwrapped != nil {
		if valkey.IsValkeyNil(unwrapped) {
			return true
		}
		unwrapped = errors.Unwrap(unwrapped)
	}
	return false
}

// Close 는 Valkey 클라이언트를 닫는다.
func Close(client valkey.Client) {
	if client != nil {
		client.Close()
	}
}
