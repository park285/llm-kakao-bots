package service

import (
	"fmt"
	"io/fs"
	"log/slog"
	"math/rand/v2"
	"strings"

	json "github.com/goccy/go-json"

	qassets "github.com/park285/llm-kakao-bots/game-bot-go/internal/twentyq/assets"
)

// TopicEntry 는 타입이다.
type TopicEntry struct {
	Name     string
	Details  map[string]any
	Category string
}

// TopicSelector 는 타입이다.
type TopicSelector struct {
	topics map[string][]TopicEntry
	rng    *rand.Rand
	logger *slog.Logger
}

// topicItemRaw JSON 파싱용 중간 구조체.
type topicItemRaw struct {
	Name    string         `json:"name"`
	Details map[string]any `json:"details"`
}

// categoryFileMapping 카테고리별 파일 매핑.
// 주의: 새 카테고리 추가 시 qconfig.AllCategories도 함께 업데이트 필요!
var categoryFileMapping = map[string]string{
	"object":        "object.json",
	"food":          "food.json",
	"place":         "place.json",
	"concept":       "concept.json",
	"movie":         "movie.json",
	"organism":      "organism.json",
	"idiom_proverb": "saza.json",
}

// NewTopicSelector 는 동작을 수행한다.
func NewTopicSelector(logger *slog.Logger) *TopicSelector {
	// rand/v2는 자동으로 안전한 시드를 사용하므로 명시적 시드 불필요
	selector := &TopicSelector{
		topics: make(map[string][]TopicEntry),
		rng:    rand.New(rand.NewPCG(rand.Uint64(), rand.Uint64())),
		logger: logger,
	}

	selector.loadTopicsFromEmbedded()
	return selector
}

func (s *TopicSelector) loadTopicsFromEmbedded() {
	totalCount := 0

	for category, filename := range categoryFileMapping {
		entries, err := s.loadCategoryFromFS(category, filename)
		if err != nil {
			s.logger.Warn("topic_load_failed", "category", category, "file", filename, "err", err)
			continue
		}
		s.topics[category] = entries
		totalCount += len(entries)
		s.logger.Info("topic_loaded", "category", category, "count", len(entries))
	}

	s.logger.Info("topic_selector_initialized", "total", totalCount, "categories", len(s.topics))
}

func (s *TopicSelector) loadCategoryFromFS(category string, filename string) ([]TopicEntry, error) {
	path := "topics/" + filename
	data, err := fs.ReadFile(qassets.TopicsFS, path)
	if err != nil {
		return nil, fmt.Errorf("read file failed: %w", err)
	}

	// JSON 구조: {"category": [{name, details}, ...]}
	var root map[string][]topicItemRaw
	if err := json.Unmarshal(data, &root); err != nil {
		return nil, fmt.Errorf("unmarshal failed: %w", err)
	}

	items, ok := root[category]
	if !ok {
		return nil, fmt.Errorf("category key not found in JSON: %s", category)
	}

	entries := make([]TopicEntry, 0, len(items))
	for _, item := range items {
		// 이름에서 괄호 부분 제거 (예: "비빔밥 (Bibimbap)" -> "비빔밥")
		name := strings.Split(item.Name, "(")[0]
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}

		entries = append(entries, TopicEntry{
			Name:     name,
			Details:  item.Details,
			Category: category,
		})
	}

	return entries, nil
}

// SelectTopic 는 동작을 수행한다.
func (s *TopicSelector) SelectTopic(category string, bannedTopics []string, excludedCategories []string) (TopicEntry, error) {
	excludedSet := makeCategorySet(excludedCategories)
	selectedCategory := strings.ToLower(strings.TrimSpace(category))
	if selectedCategory == "" {
		selectedCategory = s.selectRandomCategoryExcluding(excludedSet)
	}

	categoryTopics, ok := s.topics[selectedCategory]
	if !ok {
		s.logger.Warn("category_not_found", "requested", selectedCategory, "falling_back", true)
		selectedCategory = s.selectRandomCategoryExcluding(excludedSet)
		categoryTopics = s.topics[selectedCategory]
	}

	available := filterTopics(categoryTopics, bannedTopics)
	if len(available) == 0 {
		s.logger.Warn("no_available_topics", "category", selectedCategory, "banned_count", len(bannedTopics))
		if len(excludedSet) == 0 {
			return s.selectFromAllCategories(bannedTopics)
		}
		return s.selectFromAllCategoriesExcluding(bannedTopics, excludedSet)
	}

	idx := s.rng.IntN(len(available))
	selected := available[idx]

	s.logger.Debug("topic_selected", "category", selectedCategory, "topic", selected.Name, "available", len(available))
	return selected, nil
}

// Categories 는 동작을 수행한다.
func (s *TopicSelector) Categories() []string {
	out := make([]string, 0, len(s.topics))
	for cat := range s.topics {
		out = append(out, cat)
	}
	return out
}

func (s *TopicSelector) selectRandomCategory() string {
	return s.selectRandomCategoryExcluding(nil)
}

func (s *TopicSelector) selectRandomCategoryExcluding(excludedSet map[string]struct{}) string {
	categories := s.Categories()
	if len(categories) == 0 {
		return ""
	}
	if len(excludedSet) == 0 {
		return categories[s.rng.IntN(len(categories))]
	}

	filtered := make([]string, 0, len(categories))
	for _, cat := range categories {
		if isCategoryExcluded(cat, excludedSet) {
			continue
		}
		filtered = append(filtered, cat)
	}
	if len(filtered) == 0 {
		return ""
	}
	return filtered[s.rng.IntN(len(filtered))]
}

func (s *TopicSelector) selectFromAllCategories(bannedTopics []string) (TopicEntry, error) {
	return s.selectFromAllCategoriesExcluding(bannedTopics, nil)
}

func (s *TopicSelector) selectFromAllCategoriesExcluding(bannedTopics []string, excludedSet map[string]struct{}) (TopicEntry, error) {
	all := make([]TopicEntry, 0, 256)
	for category, items := range s.topics {
		if isCategoryExcluded(category, excludedSet) {
			continue
		}
		all = append(all, items...)
	}

	available := filterTopics(all, bannedTopics)
	if len(available) == 0 {
		return TopicEntry{}, fmt.Errorf("no topics available after filtering banned topics")
	}
	return available[s.rng.IntN(len(available))], nil
}

func makeCategorySet(categories []string) map[string]struct{} {
	if len(categories) == 0 {
		return nil
	}
	set := make(map[string]struct{}, len(categories))
	for _, cat := range categories {
		cat = strings.ToLower(strings.TrimSpace(cat))
		if cat == "" {
			continue
		}
		set[cat] = struct{}{}
	}
	if len(set) == 0 {
		return nil
	}
	return set
}

func isCategoryExcluded(category string, excludedSet map[string]struct{}) bool {
	if len(excludedSet) == 0 {
		return false
	}
	category = strings.ToLower(strings.TrimSpace(category))
	if category == "" {
		return false
	}
	_, ok := excludedSet[category]
	return ok
}

func filterTopics(topics []TopicEntry, bannedTopics []string) []TopicEntry {
	if len(bannedTopics) == 0 {
		return topics
	}

	bannedSet := make(map[string]struct{}, len(bannedTopics))
	for _, banned := range bannedTopics {
		banned = strings.ToLower(strings.TrimSpace(banned))
		if banned == "" {
			continue
		}
		bannedSet[banned] = struct{}{}
	}

	filtered := make([]TopicEntry, 0, len(topics))
	for _, topic := range topics {
		nameKey := strings.ToLower(strings.TrimSpace(topic.Name))
		if nameKey == "" {
			continue
		}
		if _, ok := bannedSet[nameKey]; ok {
			continue
		}
		filtered = append(filtered, topic)
	}
	return filtered
}
