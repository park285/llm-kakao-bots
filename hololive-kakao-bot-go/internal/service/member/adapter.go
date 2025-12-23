package member

import (
	"context"

	"github.com/kapu/hololive-kakao-bot-go/internal/domain"
)

// ServiceAdapter 는 타입이다.
type ServiceAdapter struct {
	cache *Cache
	ctx   context.Context
}

// NewMemberServiceAdapter 는 동작을 수행한다.
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
