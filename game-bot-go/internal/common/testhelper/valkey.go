package testhelper

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/valkey-io/valkey-go"
)

// TestRedisAddr: 테스트용 Redis 주소를 반환합니다.
// 환경 변수 TEST_REDIS_ADDR가 설정되어 있으면 해당 값을 사용하고,
// 그렇지 않으면 기본값 "localhost:6379"를 사용합니다.
func TestRedisAddr() string {
	if addr := os.Getenv("TEST_REDIS_ADDR"); addr != "" {
		return addr
	}
	return "localhost:6379"
}

// NewTestValkeyClient: 실제 Redis/Valkey 인스턴스에 연결하는 클라이언트를 생성합니다.
// 연결 실패 시 테스트를 스킵합니다.
func NewTestValkeyClient(t *testing.T) valkey.Client {
	t.Helper()

	addr := TestRedisAddr()
	client, err := valkey.NewClient(valkey.ClientOption{
		InitAddress:       []string{addr},
		DisableCache:      true,
		ForceSingleClient: true,
	})
	if err != nil {
		t.Skipf("skipping test: failed to create valkey client (addr=%s): %v", addr, err)
	}

	// Ping to verify connection
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := client.Do(ctx, client.B().Ping().Build()).Error(); err != nil {
		client.Close()
		t.Skipf("skipping test: redis not available (addr=%s): %v", addr, err)
	}

	return client
}

// CleanupTestKeys: 테스트용 키들을 정리합니다.
// 테스트 후 정리에 사용합니다. t가 nil이면 warning을 무시합니다.
func CleanupTestKeys(t *testing.T, client valkey.Client, prefix string) {
	if t != nil {
		t.Helper()
	}
	ctx := context.Background()

	pattern := prefix + "*"
	if t != nil {
		testPrefix := UniqueTestPrefix(t)
		if !strings.Contains(prefix, testPrefix) && strings.HasSuffix(prefix, ":") {
			pattern = prefix + "*" + testPrefix + "*"
		}
	}

	// KEYS 명령으로 패턴 매칭 후 삭제
	keys, err := client.Do(ctx, client.B().Keys().Pattern(pattern).Build()).AsStrSlice()
	if err != nil {
		if t != nil {
			t.Logf("warning: failed to get keys for cleanup: %v", err)
		}
		return
	}

	if len(keys) > 0 {
		for _, key := range keys {
			_ = client.Do(ctx, client.B().Del().Key(key).Build())
		}
	}
}

// UniqueTestPrefix: 테스트별로 고유한 키 prefix를 생성합니다.
func UniqueTestPrefix(t *testing.T) string {
	return "test:" + t.Name() + ":"
}
