package twentyq

import (
	"embed"
	"fmt"
	"io/fs"
	"math/rand/v2"
	"strings"
	"sync"

	"github.com/goccy/go-json"
)

//go:embed topics/*.json
var topicsFS embed.FS

// TopicEntry: 토픽 상세 정보 구조체입니다.
type TopicEntry struct {
	Name     string         `json:"name"`
	Details  map[string]any `json:"details"`
	Category string         `json:"category"`
}

// TopicLoader: 카테고리별 토픽을 로드하고 선택합니다.
type TopicLoader struct {
	mu     sync.RWMutex
	topics map[string][]TopicEntry
	rng    *rand.Rand
}

// topicItemRaw JSON 파싱용 중간 구조체.
type topicItemRaw struct {
	Name    string         `json:"name"`
	Details map[string]any `json:"details"`
}

// categoryFileMapping: 카테고리별 파일 매핑입니다.
var categoryFileMapping = map[string]string{
	"object":        "object.json",
	"food":          "food.json",
	"place":         "place.json",
	"concept":       "concept.json",
	"movie":         "movie.json",
	"organism":      "organism.json",
	"idiom_proverb": "saza.json",
}

// AllCategories: 사용 가능한 모든 카테고리 목록입니다.
var AllCategories = []string{
	"organism",
	"food",
	"object",
	"place",
	"concept",
	"movie",
	"idiom_proverb",
}

// NewTopicLoader: TopicLoader를 생성하고 토픽을 로드합니다.
func NewTopicLoader() (*TopicLoader, error) {
	loader := &TopicLoader{
		topics: make(map[string][]TopicEntry),
		rng:    rand.New(rand.NewPCG(rand.Uint64(), rand.Uint64())),
	}
	if err := loader.load(); err != nil {
		return nil, err
	}
	return loader, nil
}

func (l *TopicLoader) load() error {
	totalCount := 0
	for category, filename := range categoryFileMapping {
		entries, err := l.loadCategory(category, filename)
		if err != nil {
			return fmt.Errorf("load category %s: %w", category, err)
		}
		l.topics[category] = entries
		totalCount += len(entries)
	}
	if totalCount == 0 {
		return fmt.Errorf("no topics loaded")
	}
	return nil
}

func (l *TopicLoader) loadCategory(category string, filename string) ([]TopicEntry, error) {
	path := "topics/" + filename
	data, err := fs.ReadFile(topicsFS, path)
	if err != nil {
		return nil, fmt.Errorf("read file: %w", err)
	}

	// JSON 구조: {"category": [{name, details}, ...]}
	var root map[string][]topicItemRaw
	if err := json.Unmarshal(data, &root); err != nil {
		return nil, fmt.Errorf("unmarshal: %w", err)
	}

	items, ok := root[category]
	if !ok {
		return nil, fmt.Errorf("category key not found: %s", category)
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

// SelectTopic: 조건에 맞는 토픽을 랜덤하게 선택합니다.
func (l *TopicLoader) SelectTopic(category string, bannedTopics []string, excludedCategories []string) (TopicEntry, error) {
	l.mu.RLock()
	defer l.mu.RUnlock()

	excludedSet := makeCategorySet(excludedCategories)
	selectedCategory := strings.ToLower(strings.TrimSpace(category))

	if selectedCategory == "" {
		selectedCategory = l.selectRandomCategoryExcluding(excludedSet)
	}

	categoryTopics, ok := l.topics[selectedCategory]
	if !ok {
		// 요청한 카테고리가 없으면 랜덤 선택
		selectedCategory = l.selectRandomCategoryExcluding(excludedSet)
		categoryTopics = l.topics[selectedCategory]
	}

	available := filterTopics(categoryTopics, bannedTopics)
	if len(available) == 0 {
		// 해당 카테고리에 가용 토픽이 없으면 전체에서 선택
		if len(excludedSet) == 0 {
			return l.selectFromAllCategories(bannedTopics)
		}
		return l.selectFromAllCategoriesExcluding(bannedTopics, excludedSet)
	}

	idx := l.rng.IntN(len(available))
	return available[idx], nil
}

// Categories: 사용 가능한 모든 카테고리를 반환합니다.
func (l *TopicLoader) Categories() []string {
	l.mu.RLock()
	defer l.mu.RUnlock()
	out := make([]string, 0, len(l.topics))
	for cat := range l.topics {
		out = append(out, cat)
	}
	return out
}

func (l *TopicLoader) selectRandomCategoryExcluding(excludedSet map[string]struct{}) string {
	categories := l.Categories()
	if len(categories) == 0 {
		return ""
	}
	if len(excludedSet) == 0 {
		return categories[l.rng.IntN(len(categories))]
	}

	filtered := make([]string, 0, len(categories))
	for _, cat := range categories {
		if _, ok := excludedSet[strings.ToLower(cat)]; ok {
			continue
		}
		filtered = append(filtered, cat)
	}
	if len(filtered) == 0 {
		return categories[l.rng.IntN(len(categories))]
	}
	return filtered[l.rng.IntN(len(filtered))]
}

func (l *TopicLoader) selectFromAllCategories(bannedTopics []string) (TopicEntry, error) {
	return l.selectFromAllCategoriesExcluding(bannedTopics, nil)
}

func (l *TopicLoader) selectFromAllCategoriesExcluding(bannedTopics []string, excludedSet map[string]struct{}) (TopicEntry, error) {
	all := make([]TopicEntry, 0, 256)
	for category, items := range l.topics {
		if _, ok := excludedSet[strings.ToLower(category)]; ok {
			continue
		}
		all = append(all, items...)
	}

	available := filterTopics(all, bannedTopics)
	if len(available) == 0 {
		return TopicEntry{}, fmt.Errorf("no topics available after filtering")
	}
	return available[l.rng.IntN(len(available))], nil
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
