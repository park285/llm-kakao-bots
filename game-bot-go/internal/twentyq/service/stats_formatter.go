package service

import (
	"cmp"
	"fmt"
	"slices"
	"strings"

	"github.com/park285/llm-kakao-bots/game-bot-go/internal/common/messageprovider"
	qconfig "github.com/park285/llm-kakao-bots/game-bot-go/internal/twentyq/config"
	qmessages "github.com/park285/llm-kakao-bots/game-bot-go/internal/twentyq/messages"
	qmodel "github.com/park285/llm-kakao-bots/game-bot-go/internal/twentyq/model"
	"github.com/park285/llm-kakao-bots/game-bot-go/internal/twentyq/repository"
)

// formatStats 사용자 통계를 포맷팅하여 반환.
func (s *StatsService) formatStats(stats *repository.UserStats, senderName string) string {
	var parts []string

	// 카테고리별 통계 파싱
	totalGames := stats.TotalGamesCompleted
	categoryStats := make(map[string]CategoryStat)
	if stats.CategoryStatsJSON != nil && strings.TrimSpace(*stats.CategoryStatsJSON) != "" {
		categoryStats = s.parseCategoryStats(*stats.CategoryStatsJSON)
	}
	categoryStats = normalizeCategoryStats(categoryStats)
	if len(categoryStats) > 0 {
		sumCategoryGames := 0
		for _, stat := range categoryStats {
			sumCategoryGames += stat.GamesCompleted
		}
		if sumCategoryGames > 0 {
			totalGames = sumCategoryGames
		}
	}
	categoryStats = ensureAllCategoryStats(categoryStats)

	// 헤더
	parts = append(parts, s.msgProvider.Get(
		qmessages.StatsHeader,
		messageprovider.P("nickname", senderName),
		messageprovider.P("totalGames", totalGames),
	))

	if len(categoryStats) > 0 {
		s.addCategoryStats(&parts, categoryStats)
	}

	return strings.Join(parts, "\n")
}

// addCategoryStats 카테고리별 통계를 추가.
func (s *StatsService) addCategoryStats(parts *[]string, categoryStats map[string]CategoryStat) {
	// 카테고리 정렬
	type catEntry struct {
		name string
		stat CategoryStat
	}
	order := make(map[string]int, len(qconfig.AllCategories))
	for i, name := range qconfig.AllCategories {
		order[name] = i
	}
	entries := make([]catEntry, 0, len(categoryStats))
	for name, stat := range categoryStats {
		entries = append(entries, catEntry{name: name, stat: stat})
	}
	slices.SortFunc(entries, func(a, b catEntry) int {
		// GamesCompleted 내림차순 정렬
		if c := cmp.Compare(b.stat.GamesCompleted, a.stat.GamesCompleted); c != 0 {
			return c
		}
		// order map 기준 정렬
		orderA, okA := order[a.name]
		orderB, okB := order[b.name]
		if okA && okB {
			return cmp.Compare(orderA, orderB)
		}
		if okA != okB {
			if okA {
				return -1
			}
			return 1
		}
		return cmp.Compare(a.name, b.name)
	})

	for _, entry := range entries {
		s.addSingleCategoryStats(parts, entry.name, entry.stat)
	}
}

// addSingleCategoryStats 단일 카테고리 통계를 추가.
func (s *StatsService) addSingleCategoryStats(parts *[]string, category string, stat CategoryStat) {
	displayCategory := category
	if korean := categoryToKorean(category); korean != nil {
		displayCategory = *korean
	}

	// 카테고리 헤더
	*parts = append(*parts, s.msgProvider.Get(
		qmessages.StatsCategoryHdr,
		messageprovider.P("category", displayCategory),
		messageprovider.P("games", stat.GamesCompleted),
	))

	// 완주율
	completed := stat.GamesCompleted - stat.Surrenders
	if completed < 0 {
		completed = 0
	}
	completionRate := 0
	if stat.GamesCompleted > 0 {
		completionRate = (completed * percentageMultiplier) / stat.GamesCompleted
	}
	*parts = append(*parts, s.msgProvider.Get(
		qmessages.StatsCategoryResults,
		messageprovider.P("completed", completed),
		messageprovider.P("surrender", stat.Surrenders),
		messageprovider.P("completionRate", completionRate),
	))

	// 평균 질문/힌트
	avgQuestions := 0.0
	avgHints := 0.0
	if stat.GamesCompleted > 0 {
		avgQuestions = float64(stat.QuestionsAsked) / float64(stat.GamesCompleted)
		avgHints = float64(stat.HintsUsed) / float64(stat.GamesCompleted)
	}
	*parts = append(*parts, s.msgProvider.Get(
		qmessages.StatsCategoryAverages,
		messageprovider.P("avgQuestions", fmt.Sprintf("%.1f", avgQuestions)),
		messageprovider.P("avgHints", fmt.Sprintf("%.1f", avgHints)),
	))

	// 베스트 스코어 출력 제거 (모바일 메시지 길이 축소)
}

// formatRoomStats 방 전적을 포맷팅하여 반환.
func (s *StatsService) formatRoomStats(
	period qmodel.StatsPeriod,
	totalGames int,
	totalParticipants int,
	completionRate int,
	activities []ParticipantActivity,
) string {
	var parts []string

	// 헤더
	parts = append(parts,
		s.msgProvider.Get(
			qmessages.StatsRoomHeader,
			messageprovider.P("period", s.getPeriodName(period)),
		),
		"",
		s.msgProvider.Get(
			qmessages.StatsRoomSummary,
			messageprovider.P("totalGames", totalGames),
			messageprovider.P("totalParticipants", totalParticipants),
			messageprovider.P("completionRate", completionRate),
		),
	)

	// 참여 활동
	if len(activities) > 0 {
		parts = append(parts, "", s.msgProvider.Get(qmessages.StatsRoomActivityHdr))
		for _, activity := range activities {
			parts = append(parts, s.msgProvider.Get(
				qmessages.StatsRoomActivityItem,
				messageprovider.P("sender", activity.Sender),
				messageprovider.P("games", activity.GamesPlayed),
			))
		}
	}

	return strings.Join(parts, "\n")
}

// getPeriodName 기간 이름을 반환.
func (s *StatsService) getPeriodName(period qmodel.StatsPeriod) string {
	switch period {
	case qmodel.StatsPeriodDaily:
		return s.msgProvider.Get(qmessages.StatsPeriodDaily)
	case qmodel.StatsPeriodWeekly:
		return s.msgProvider.Get(qmessages.StatsPeriodWeekly)
	case qmodel.StatsPeriodMonthly:
		return s.msgProvider.Get(qmessages.StatsPeriodMonthly)
	case qmodel.StatsPeriodAll:
		return s.msgProvider.Get(qmessages.StatsPeriodAll)
	default:
		return s.msgProvider.Get(qmessages.StatsPeriodAll)
	}
}
