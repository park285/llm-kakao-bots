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

// ToValkeyDataConfig 는 RedisConfig를 valkeyx.Config로 변환한다.
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

// ToValkeyMQConfig 는 ValkeyMQConfig를 valkeyx.Config로 변환한다.
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
		DisableCache: false, // 프로덕션에서는 캐싱 활성화
	}
}

// NewAndPingValkeyClient 는 Valkey 클라이언트를 생성하고 연결을 확인한다.
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

// NewAndPingDataValkeyClient 는 데이터용 Valkey 클라이언트를 생성한다.
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

// NewAndPingMQValkeyClient 는 MQ용 Valkey 클라이언트를 생성한다.
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
