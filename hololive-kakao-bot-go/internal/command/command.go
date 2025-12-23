package command

import (
	"context"

	"go.uber.org/zap"

	"github.com/kapu/hololive-kakao-bot-go/internal/adapter"
	"github.com/kapu/hololive-kakao-bot-go/internal/domain"
	"github.com/kapu/hololive-kakao-bot-go/internal/service/cache"
	"github.com/kapu/hololive-kakao-bot-go/internal/service/holodex"
	"github.com/kapu/hololive-kakao-bot-go/internal/service/matcher"
	"github.com/kapu/hololive-kakao-bot-go/internal/service/member"
	"github.com/kapu/hololive-kakao-bot-go/internal/service/notification"
	"github.com/kapu/hololive-kakao-bot-go/internal/service/youtube"
)

// Command 는 타입이다.
type Command interface {
	Name() string
	Description() string
	Execute(ctx context.Context, cmdCtx *domain.CommandContext, params map[string]any) error
}

// Event 는 타입이다.
type Event struct {
	Type   domain.CommandType
	Params map[string]any
}

// Dispatcher 는 타입이다.
type Dispatcher interface {
	Publish(ctx context.Context, cmdCtx *domain.CommandContext, events ...Event) (int, error)
}

// Dependencies 는 타입이다.
type Dependencies struct {
	Holodex          *holodex.Service
	Cache            *cache.Service
	Alarm            *notification.AlarmService
	Matcher          *matcher.MemberMatcher
	OfficialProfiles *member.ProfileService
	StatsRepo        *youtube.StatsRepository
	MembersData      domain.MemberDataProvider
	Formatter        *adapter.ResponseFormatter
	SendMessage      func(ctx context.Context, room, message string) error
	SendImage        func(ctx context.Context, room, imageBase64 string) error
	SendError        func(ctx context.Context, room, message string) error
	Dispatcher       Dispatcher
	Logger           *zap.Logger
}
