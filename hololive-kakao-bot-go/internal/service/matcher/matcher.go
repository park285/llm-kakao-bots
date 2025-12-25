package matcher

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/kapu/hololive-kakao-bot-go/internal/domain"
	"github.com/kapu/hololive-kakao-bot-go/internal/service/cache"
	"github.com/kapu/hololive-kakao-bot-go/internal/service/holodex"
	"github.com/kapu/hololive-kakao-bot-go/internal/util"
)

// MatchCacheEntry: 멤버 매칭 결과를 캐싱하기 위한 구조체 (채널 정보 + 타임스탬프)
type MatchCacheEntry struct {
	Channel   *domain.Channel
	Timestamp time.Time
}

type matchCandidate struct {
	channelID  string
	memberName string
	source     string
}

// ChannelSelector: 모호한 검색어에 대해 모호성 해소를 돕는 채널 선택 인터페이스
type ChannelSelector interface {
	SelectBestChannel(ctx context.Context, query string, candidates []*domain.Channel) (*domain.Channel, error)
}

// MemberMatcher: 사용자 검색어(이름, 별명 등)를 기반으로 Hololive 멤버(채널)를 식별하고 매칭하는 서비스
// 다양한 매칭 전략(정확 일치, 부분 일치, 별명 검색 등)을 순차적으로 시도한다.
type MemberMatcher struct {
	membersData           domain.MemberDataProvider
	fallbackData          domain.MemberDataProvider
	cache                 *cache.Service
	holodex               *holodex.Service
	selector              ChannelSelector
	logger                *slog.Logger
	matchCache            map[string]*MatchCacheEntry
	matchCacheMu          sync.RWMutex
	matchCacheTTL         time.Duration
	matchCacheLastCleanup time.Time
}

// NewMemberMatcher: 새로운 MemberMatcher 인스턴스를 생성한다.
func NewMemberMatcher(
	ctx context.Context,
	membersData domain.MemberDataProvider,
	cache *cache.Service,
	holodex *holodex.Service,
	selector ChannelSelector,
	logger *slog.Logger,
) *MemberMatcher {
	if ctx == nil {
		ctx = context.Background()
	}

	var fallbackProvider domain.MemberDataProvider
	if fallbackData, err := domain.LoadMembersData(); err != nil {
		logger.Warn("Failed to load fallback member data", slog.Any("error", err))
	} else {
		fallbackProvider = fallbackData.WithContext(ctx)
	}

	mm := &MemberMatcher{
		membersData:           membersData,
		fallbackData:          fallbackProvider,
		cache:                 cache,
		holodex:               holodex,
		selector:              selector,
		logger:                logger,
		matchCache:            make(map[string]*MatchCacheEntry),
		matchCacheTTL:         1 * time.Minute,
		matchCacheLastCleanup: time.Now(),
	}

	provider := membersData.WithContext(ctx)

	logger.Info("MemberMatcher initialized",
		slog.Int("members", len(provider.GetAllMembers())),
	)

	return mm
}

// tryExactAliasMatch attempts exact match via database aliases (Lazy Loading from PostgreSQL)
func (mm *MemberMatcher) tryExactAliasMatch(ctx context.Context, provider, fallback domain.MemberDataProvider, queryNorm string) *matchCandidate {
	// Try provider first (PostgreSQL with Valkey cache)
	if member := provider.FindMemberByAlias(queryNorm); member != nil && member.ChannelID != "" {
		return mm.candidateFromMember(member, "alias-db")
	}

	// Try fallback
	if fallback != nil {
		if member := fallback.FindMemberByAlias(queryNorm); member != nil && member.ChannelID != "" {
			return mm.candidateFromMember(member, "alias-fallback")
		}
	}

	return nil
}

// tryExactValkeyMatch attempts exact match in dynamic Valkey data without immediate Holodex calls.
func (mm *MemberMatcher) tryExactValkeyMatch(provider domain.MemberDataProvider, query string, dynamicMembers map[string]string) *matchCandidate {
	for name, channelID := range dynamicMembers {
		if strings.EqualFold(name, query) {
			return mm.candidateFromDynamic(provider, name, channelID, "valkey-exact")
		}
	}
	return nil
}

// tryPartialStaticMatch attempts partial match in static member data.
func (mm *MemberMatcher) tryPartialStaticMatch(provider, fallback domain.MemberDataProvider, queryNorm string) *matchCandidate {
	if provider != nil {
		for _, member := range provider.GetAllMembers() {
			nameNorm := util.Normalize(member.Name)
			if strings.Contains(nameNorm, queryNorm) || strings.Contains(queryNorm, nameNorm) {
				return mm.candidateFromMember(member, "static-partial")
			}
		}
	}

	if fallback != nil {
		for _, member := range fallback.GetAllMembers() {
			nameNorm := util.Normalize(member.Name)
			if strings.Contains(nameNorm, queryNorm) || strings.Contains(queryNorm, nameNorm) {
				return mm.candidateFromMember(member, "static-partial-fallback")
			}
		}
	}

	return nil
}

// tryPartialValkeyMatch attempts partial match in dynamic Valkey data.
func (mm *MemberMatcher) tryPartialValkeyMatch(provider domain.MemberDataProvider, queryNorm string, dynamicMembers map[string]string) *matchCandidate {
	for name, channelID := range dynamicMembers {
		nameNorm := util.Normalize(name)
		if strings.Contains(nameNorm, queryNorm) || strings.Contains(queryNorm, nameNorm) {
			return mm.candidateFromDynamic(provider, name, channelID, "valkey-partial")
		}
	}
	return nil
}

// tryPartialAliasMatch attempts partial match across all aliases.
func (mm *MemberMatcher) tryPartialAliasMatch(provider, fallback domain.MemberDataProvider, queryNorm string) *matchCandidate {
	if provider != nil {
		for _, member := range provider.GetAllMembers() {
			for _, alias := range member.GetAllAliases() {
				aliasNorm := util.Normalize(alias)
				if strings.Contains(aliasNorm, queryNorm) || strings.Contains(queryNorm, aliasNorm) {
					return mm.candidateFromMember(member, "alias-partial")
				}
			}
		}
	}

	if fallback != nil {
		for _, member := range fallback.GetAllMembers() {
			for _, alias := range member.GetAllAliases() {
				aliasNorm := util.Normalize(alias)
				if strings.Contains(aliasNorm, queryNorm) || strings.Contains(queryNorm, aliasNorm) {
					return mm.candidateFromMember(member, "alias-partial-fallback")
				}
			}
		}
	}

	return nil
}

func (mm *MemberMatcher) candidateFromMember(member *domain.Member, source string) *matchCandidate {
	if member == nil || member.ChannelID == "" {
		return nil
	}

	name := member.Name
	if name == "" {
		name = member.NameJa
	}
	if name == "" {
		name = member.ChannelID
	}

	return &matchCandidate{
		channelID:  member.ChannelID,
		memberName: name,
		source:     source,
	}
}

func (mm *MemberMatcher) candidateFromDynamic(provider domain.MemberDataProvider, name, channelID, source string) *matchCandidate {
	if channelID == "" {
		return nil
	}

	if provider != nil {
		if member := provider.FindMemberByChannelID(channelID); member != nil {
			if candidate := mm.candidateFromMember(member, source); candidate != nil {
				return candidate
			}
		}
	}

	if mm.fallbackData != nil {
		if member := mm.fallbackData.FindMemberByChannelID(channelID); member != nil {
			if candidate := mm.candidateFromMember(member, source); candidate != nil {
				return candidate
			}
		}
	}

	displayName := name
	if displayName == "" {
		displayName = channelID
	}

	return &matchCandidate{
		channelID:  channelID,
		memberName: displayName,
		source:     source,
	}
}

func (mm *MemberMatcher) hydrateChannel(ctx context.Context, candidate *matchCandidate) (*domain.Channel, error) {
	if candidate == nil {
		return nil, nil
	}

	fallback := &domain.Channel{
		ID:   candidate.channelID,
		Name: candidate.memberName,
	}
	if candidate.memberName != "" {
		fallback.EnglishName = toStringPtr(candidate.memberName)
	}

	if mm.holodex == nil {
		return fallback, nil
	}

	channel, err := mm.holodex.GetChannel(ctx, candidate.channelID)
	if err != nil {
		mm.logger.Warn("Failed to fetch channel from Holodex",
			slog.String("channel_id", candidate.channelID),
			slog.String("source", candidate.source),
			slog.Any("error", err),
		)
		return fallback, nil
	}

	if channel == nil {
		mm.logger.Warn("Holodex returned empty channel",
			slog.String("channel_id", candidate.channelID),
			slog.String("source", candidate.source),
		)
		return fallback, nil
	}

	if candidate.memberName != "" {
		if channel.Name == "" {
			channel.Name = candidate.memberName
		}
		if channel.EnglishName == nil {
			channel.EnglishName = toStringPtr(candidate.memberName)
		}
	}

	return channel, nil
}

func (mm *MemberMatcher) finalizeCandidate(ctx context.Context, candidate *matchCandidate) (*domain.Channel, error) {
	if candidate == nil {
		return nil, nil
	}

	if candidate.channelID == "" {
		mm.logger.Warn("Match candidate missing channel ID",
			slog.String("member", candidate.memberName),
			slog.String("source", candidate.source),
		)
		return nil, nil
	}

	channel, err := mm.hydrateChannel(ctx, candidate)
	if err != nil {
		return nil, err
	}

	if channel != nil {
		mm.logger.Debug("Match candidate resolved",
			slog.String("channel_id", candidate.channelID),
			slog.String("member", candidate.memberName),
			slog.String("source", candidate.source),
		)
	}

	return channel, nil
}

func (mm *MemberMatcher) maybeCleanupMatchCache() {
	mm.matchCacheMu.Lock()
	defer mm.matchCacheMu.Unlock()

	if time.Since(mm.matchCacheLastCleanup) < mm.matchCacheTTL {
		return
	}

	cutoff := time.Now().Add(-mm.matchCacheTTL)
	for key, entry := range mm.matchCache {
		if entry == nil || entry.Timestamp.Before(cutoff) {
			delete(mm.matchCache, key)
		}
	}

	mm.matchCacheLastCleanup = time.Now()
}

func toStringPtr(value string) *string {
	if value == "" {
		return nil
	}
	copied := value
	return &copied
}

// loadDynamicMembers fetches member data from Valkey cache
func (mm *MemberMatcher) loadDynamicMembers(ctx context.Context) map[string]string {
	members, err := mm.cache.GetAllMembers(ctx)
	if err != nil {
		mm.logger.Warn("Failed to load dynamic members", slog.Any("error", err))
		return map[string]string{}
	}
	return members
}

// FindBestMatch: 주어진 쿼리 문자열과 가장 잘 일치하는 멤버/채널을 찾는다.
// 캐시된 결과가 있으면 반환하고, 없으면 여러 매칭 전략을 시도한다.
func (mm *MemberMatcher) FindBestMatch(ctx context.Context, query string) (*domain.Channel, error) {
	normalizedQuery := util.Normalize(query)
	cacheKey := fmt.Sprintf("match:%s", normalizedQuery)

	mm.matchCacheMu.RLock()
	cached, found := mm.matchCache[cacheKey]
	mm.matchCacheMu.RUnlock()

	if found {
		age := time.Since(cached.Timestamp)
		if age < mm.matchCacheTTL {
			return cached.Channel, nil
		}

		mm.matchCacheMu.Lock()
		delete(mm.matchCache, cacheKey)
		mm.matchCacheMu.Unlock()
	}

	channel, err := mm.findBestMatchImpl(ctx, query)

	mm.matchCacheMu.Lock()
	mm.matchCache[cacheKey] = &MatchCacheEntry{
		Channel:   channel,
		Timestamp: time.Now(),
	}
	mm.matchCacheMu.Unlock()

	mm.maybeCleanupMatchCache()

	return channel, err
}

func (mm *MemberMatcher) findBestMatchImpl(ctx context.Context, query string) (*domain.Channel, error) {
	provider := mm.membersData.WithContext(ctx)
	var fallbackProvider domain.MemberDataProvider
	if mm.fallbackData != nil {
		fallbackProvider = mm.fallbackData.WithContext(ctx)
	}
	queryNorm := util.NormalizeSuffix(query)

	// Strategy 1: Exact alias match (fastest)
	if channel, err := mm.finalizeCandidate(ctx, mm.tryExactAliasMatch(ctx, provider, fallbackProvider, queryNorm)); err != nil || channel != nil {
		return channel, err
	}

	// Load dynamic members once for strategies 2 & 4
	dynamicMembers := mm.loadDynamicMembers(ctx)

	// Strategy 2: Exact match in Valkey
	if channel, err := mm.finalizeCandidate(ctx, mm.tryExactValkeyMatch(provider, query, dynamicMembers)); err != nil || channel != nil {
		return channel, err
	}

	// Strategy 3: Partial match in static data
	if channel, err := mm.finalizeCandidate(ctx, mm.tryPartialStaticMatch(provider, fallbackProvider, queryNorm)); err != nil || channel != nil {
		return channel, err
	}

	// Strategy 4: Partial match in Valkey
	if channel, err := mm.finalizeCandidate(ctx, mm.tryPartialValkeyMatch(provider, queryNorm, dynamicMembers)); err != nil || channel != nil {
		return channel, err
	}

	// Strategy 5: Partial alias match
	if channel, err := mm.finalizeCandidate(ctx, mm.tryPartialAliasMatch(provider, fallbackProvider, queryNorm)); err != nil || channel != nil {
		return channel, err
	}

	// 내부 데이터에서 매칭 실패 - nil 반환하여 상위에서 "멤버를 찾을 수 없습니다" 오류 표시
	mm.logger.Debug("No match found in internal data", slog.String("query", query))
	return nil, nil
}

// GetAllMembers: 등록된 모든 멤버 정보를 반환한다.
func (mm *MemberMatcher) GetAllMembers() []*domain.Member {
	return mm.membersData.WithContext(context.Background()).GetAllMembers()
}

// GetMemberByChannelID: 채널 ID를 사용하여 멤버 정보를 조회한다.
func (mm *MemberMatcher) GetMemberByChannelID(ctx context.Context, channelID string) *domain.Member {
	if ctx == nil {
		ctx = context.Background()
	}
	return mm.membersData.WithContext(ctx).FindMemberByChannelID(channelID)
}
