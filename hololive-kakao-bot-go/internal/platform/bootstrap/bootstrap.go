package bootstrap

import (
	"fmt"

	"go.uber.org/zap"

	"github.com/kapu/hololive-kakao-bot-go/internal/adapter"
	"github.com/kapu/hololive-kakao-bot-go/internal/config"
	"github.com/kapu/hololive-kakao-bot-go/internal/service/cache"
	"github.com/kapu/hololive-kakao-bot-go/internal/service/database"
	"github.com/kapu/hololive-kakao-bot-go/internal/util"
)

// CacheResources 는 타입이다.
type CacheResources struct {
	Service *cache.Service
	Close   func()
}

// DatabaseResources 는 타입이다.
type DatabaseResources struct {
	Service *database.PostgresService
	Close   func()
}

// NewLogger 는 동작을 수행한다.
func NewLogger(cfg *config.Config) (*zap.Logger, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config must not be nil")
	}
	logger, err := util.NewLogger(cfg.Logging.Level, cfg.Logging.File)
	if err != nil {
		return nil, fmt.Errorf("create logger: %w", err)
	}
	return logger, nil
}

// NewMessageStack 는 동작을 수행한다.
func NewMessageStack(prefix string) (*adapter.MessageAdapter, *adapter.ResponseFormatter) {
	return adapter.NewMessageAdapter(prefix), adapter.NewResponseFormatter(prefix)
}

// NewCacheResources 는 동작을 수행한다.
func NewCacheResources(cfg config.ValkeyConfig, logger *zap.Logger) (*CacheResources, error) {
	cacheSvc, err := cache.NewCacheService(cache.Config{
		Host:     cfg.Host,
		Port:     cfg.Port,
		Password: cfg.Password,
		DB:       cfg.DB,
	}, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create cache service: %w", err)
	}

	res := &CacheResources{
		Service: cacheSvc,
		Close: func() {
			_ = cacheSvc.Close()
		},
	}
	return res, nil
}

// NewDatabaseResources 는 동작을 수행한다.
func NewDatabaseResources(cfg config.PostgresConfig, logger *zap.Logger) (*DatabaseResources, error) {
	dbSvc, err := database.NewPostgresService(database.PostgresConfig{
		Host:     cfg.Host,
		Port:     cfg.Port,
		User:     cfg.User,
		Password: cfg.Password,
		Database: cfg.Database,
	}, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create postgres service: %w", err)
	}

	res := &DatabaseResources{
		Service: dbSvc,
		Close: func() {
			_ = dbSvc.Close()
		},
	}
	return res, nil
}
