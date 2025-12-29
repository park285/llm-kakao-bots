package bootstrap

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/valkey-io/valkey-go"

	commonconfig "github.com/park285/llm-kakao-bots/game-bot-go/internal/common/config"
	"github.com/park285/llm-kakao-bots/game-bot-go/internal/common/di"
	"github.com/park285/llm-kakao-bots/game-bot-go/internal/common/valkeyx"
)

// ToValkeyDataConfig: 데이터 저장소 연결을 위한 Valkey 설정 객체를 생성합니다.
// 프로덕션 환경 효율성을 위해 클라이언트 사이드 캐싱이 활성화됩니다.
func ToValkeyDataConfig(cfg commonconfig.RedisConfig) valkeyx.Config {
	return valkeyx.Config{
		Addr:         fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
		Password:     cfg.Password,
		DB:           cfg.DB,
		DialTimeout:  cfg.DialTimeout,
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
		PoolSize:     cfg.PoolSize,
		MinIdleConns: cfg.MinIdleConns,
		DisableCache: false, // 프로덕션에서는 캐싱 활성화
	}
}

// ToValkeyMQConfig: 메시지 큐(MQ) 용도의 Valkey 설정 객체를 생성합니다.
// 큐 데이터의 실시간성을 보장하기 위해 클라이언트 캐싱을 비활성화합니다.
func ToValkeyMQConfig(cfg commonconfig.ValkeyMQConfig) valkeyx.Config {
	return valkeyx.Config{
		Addr:         fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
		Password:     cfg.Password,
		DB:           cfg.DB,
		DialTimeout:  cfg.DialTimeout,
		ReadTimeout:  cfg.Timeout,
		WriteTimeout: cfg.Timeout,
		PoolSize:     cfg.PoolSize,
		MinIdleConns: cfg.MinIdleConns,
		DisableCache: true, // MQ는 매번 상태가 변하므로 클라이언트 캐싱 비활성화
	}
}

// NewAndPingValkeyClient: Valkey 클라이언트를 생성하고 Ping 테스트를 통해 연결 연결성을 확인합니다.
// 연결 실패 시 생성된 리소스를 정리하고 에러를 반환합니다.
func NewAndPingValkeyClient(
	ctx context.Context,
	cfg valkeyx.Config,
	name string,
	closeWarnKey string,
	logger *slog.Logger,
) (valkey.Client, func(), error) {
	client, err := valkeyx.NewClient(cfg)
	if err != nil {
		return nil, nil, fmt.Errorf("create %s client failed: %w", name, err)
	}

	closeFn := func() {
		client.Close()
		logger.Debug("valkey_client_closed", "name", name)
	}

	if pingErr := valkeyx.Ping(ctx, client); pingErr != nil {
		closeFn()
		return nil, nil, fmt.Errorf("%s ping failed: %w", name, pingErr)
	}

	return client, closeFn, nil
}

// NewAndPingDataValkeyClient: 메인 데이터 저장소용 Valkey 클라이언트를 생성 및 초기화합니다.
func NewAndPingDataValkeyClient(
	ctx context.Context,
	cfg commonconfig.RedisConfig,
	logger *slog.Logger,
) (di.DataValkeyClient, func(), error) {
	client, closeFn, err := NewAndPingValkeyClient(
		ctx,
		ToValkeyDataConfig(cfg),
		"valkey",
		"valkey_close_failed",
		logger,
	)
	if err != nil {
		return di.DataValkeyClient{}, nil, err
	}
	return di.DataValkeyClient{Client: client}, closeFn, nil
}

// NewAndPingMQValkeyClient: 메시지 큐 전용 Valkey 클라이언트를 생성 및 초기화합니다.
func NewAndPingMQValkeyClient(
	ctx context.Context,
	cfg commonconfig.ValkeyMQConfig,
	logger *slog.Logger,
) (di.MQValkeyClient, func(), error) {
	client, closeFn, err := NewAndPingValkeyClient(
		ctx,
		ToValkeyMQConfig(cfg),
		"valkey mq",
		"valkey_mq_close_failed",
		logger,
	)
	if err != nil {
		return di.MQValkeyClient{}, nil, err
	}
	return di.MQValkeyClient{Client: client}, closeFn, nil
}
