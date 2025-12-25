package member

import (
	"context"

	"github.com/kapu/hololive-kakao-bot-go/internal/domain"
)

// ServiceAdapter: MemberCache를 래핑하여 domain.MemberDataProvider 인터페이스를 구현하는 어댑터
// 이를 통해 도메인 로직에서 구체적인 캐시 구현에 의존하지 않고 멤버 정보를 조회할 수 있다.
type ServiceAdapter struct {
	cache *Cache
	ctx   context.Context
}

// NewMemberServiceAdapter: 새로운 MemberServiceAdapter 인스턴스를 생성한다.
func NewMemberServiceAdapter(cache *Cache) *ServiceAdapter {
	return &ServiceAdapter{
		cache: cache,
		ctx:   context.Background(),
	}
}

// FindMemberByChannelID implements MembersData interface
func (a *ServiceAdapter) FindMemberByChannelID(channelID string) *domain.Member {
	member, err := a.cache.GetByChannelID(a.ctx, channelID)
	if err != nil {
		return nil
	}
	return member
}

// FindMemberByName implements MembersData interface
func (a *ServiceAdapter) FindMemberByName(name string) *domain.Member {
	member, err := a.cache.GetByName(a.ctx, name)
	if err != nil {
		return nil
	}
	return member
}

// FindMemberByAlias implements MembersData interface
func (a *ServiceAdapter) FindMemberByAlias(alias string) *domain.Member {
	member, err := a.cache.FindByAlias(a.ctx, alias)
	if err != nil {
		return nil
	}
	return member
}

// GetChannelIDs implements MemberDataProvider interface
func (a *ServiceAdapter) GetChannelIDs() []string {
	channelIDs, err := a.cache.GetAllChannelIDs(a.ctx)
	if err != nil {
		return []string{}
	}
	return channelIDs
}

// GetAllMembers implements MemberDataProvider interface
func (a *ServiceAdapter) GetAllMembers() []*domain.Member {
	members, err := a.cache.repo.GetAllMembers(a.ctx)
	if err != nil {
		return []*domain.Member{}
	}
	return members
}

// WithContext creates a new adapter with custom context
func (a *ServiceAdapter) WithContext(ctx context.Context) domain.MemberDataProvider {
	if ctx == nil {
		ctx = context.Background()
	}
	return &ServiceAdapter{
		cache: a.cache,
		ctx:   ctx,
	}
}
