package bot

import (
	"go.uber.org/zap"

	"github.com/kapu/hololive-kakao-bot-go/internal/adapter"
	"github.com/kapu/hololive-kakao-bot-go/internal/config"
	"github.com/kapu/hololive-kakao-bot-go/internal/domain"
	"github.com/kapu/hololive-kakao-bot-go/internal/iris"
	"github.com/kapu/hololive-kakao-bot-go/internal/service/acl"
	"github.com/kapu/hololive-kakao-bot-go/internal/service/activity"
	"github.com/kapu/hololive-kakao-bot-go/internal/service/cache"
	"github.com/kapu/hololive-kakao-bot-go/internal/service/database"
	"github.com/kapu/hololive-kakao-bot-go/internal/service/holodex"
	"github.com/kapu/hololive-kakao-bot-go/internal/service/matcher"
	"github.com/kapu/hololive-kakao-bot-go/internal/service/member"
	"github.com/kapu/hololive-kakao-bot-go/internal/service/notification"
	"github.com/kapu/hololive-kakao-bot-go/internal/service/settings"
	"github.com/kapu/hololive-kakao-bot-go/internal/service/youtube"
)

// Dependencies 는 타입이다.
type Dependencies struct {
	Config           *config.Config
	Logger           *zap.Logger
	Client           iris.Client
	MessageAdapter   *adapter.MessageAdapter
	Formatter        *adapter.ResponseFormatter
	Cache            *cache.Service
	Postgres         *database.PostgresService
	MemberRepo       *member.Repository
	MemberCache      *member.Cache
	Holodex          *holodex.Service
	Profiles         *member.ProfileService
	Alarm            *notification.AlarmService
	Matcher          *matcher.MemberMatcher
	MembersData      domain.MemberDataProvider
	Service          *youtube.Service
	Scheduler        *youtube.Scheduler
	YouTubeStatsRepo *youtube.StatsRepository
	Activity         *activity.Logger
	Settings         *settings.Service
	ACL              *acl.Service
}
