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

// Config: Valkey 클라이언트 연결에 필요한 설정 정보를 담고 있다.
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

	// DisableCache: 클라이언트 사이드 캐싱(Client Side Caching) 기능을 비활성화할지 여부.
	// 일반적으로 로컬 테스트 환경이나 miniredis 사용 시 true로 설정한다.
	DisableCache bool

	// UseTLS: TLS(SSL) 연결 사용 여부.
	UseTLS bool
}

// NewClient: 주어진 설정을 바탕으로 Valkey 클라이언트 인스턴스를 생성하고 초기화한다.
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

// Ping: Valkey 서버와의 연결 상태를 점검한다. (PING 명령 전송)
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

// IsNil: 발생한 에러가 Valkey nil(키가 없음) 에러인지 확인한다.
// 에러 래핑을 고려하여 언래핑 후 검사를 수행한다.
func IsNil(err error) bool {
	if valkey.IsValkeyNil(err) {
		return true
	}
	// fmt.Errorf("%w", err)로 래핑된 경우 언래핑하여 체크
	unwrapped := err
	for unwrapped != nil {
		if valkey.IsValkeyNil(unwrapped) {
			return true
		}
		unwrapped = errors.Unwrap(unwrapped)
	}
	return false
}

// Close: Valkey 클라이언트 연결을 안전하게 종료한다.
func Close(client valkey.Client) {
	if client != nil {
		client.Close()
	}
}
