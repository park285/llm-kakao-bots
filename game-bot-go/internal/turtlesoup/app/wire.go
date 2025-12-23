//go:build wireinject

package app

import (
	"context"
	"log/slog"

	"github.com/google/wire"

	"github.com/park285/llm-kakao-bots/game-bot-go/internal/common/bootstrap"
	tsconfig "github.com/park285/llm-kakao-bots/game-bot-go/internal/turtlesoup/config"
)

//go:generate go run github.com/google/wire/cmd/wire@v0.7.0
func Initialize(
	ctx context.Context,
	cfg *tsconfig.Config,
	logger *slog.Logger,
) (*bootstrap.ServerApp, func(), error) {
	wire.Build(
		turtleSoupProviderSet,
	)
	return nil, nil, nil
}
