package guard

import (
	"errors"
	"log/slog"
	"math"
	"os"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/sync/singleflight"

	"github.com/park285/llm-kakao-bots/mcp-llm-server-go/internal/cache"
	"github.com/park285/llm-kakao-bots/mcp-llm-server-go/internal/config"
)

// InjectionGuard: 입력 문자열을 검사하는 보안 가드입니다.
type InjectionGuard struct {
	cfg    *config.Config
	logger *slog.Logger
	packs  []compiledPack
	cache  *cache.TTLCache[string, Evaluation]
	group  singleflight.Group
}

// NewGuard: 입력 검증 가드를 생성합니다.
func NewGuard(cfg *config.Config, logger *slog.Logger) (*InjectionGuard, error) {
	if cfg == nil {
		return nil, errors.New("config is nil")
	}

	cacheTTL := time.Duration(cfg.Guard.CacheTTLSeconds) * time.Second
	guard := &InjectionGuard{
		cfg:    cfg,
		logger: logger,
		cache:  cache.NewTTLCache[string, Evaluation](cfg.Guard.CacheMaxSize, cacheTTL),
	}

	if cfg.Guard.Enabled {
		guard.loadRulepacks()
	}

	return guard, nil
}

// Evaluate: 입력 문자열을 평가합니다.
func (g *InjectionGuard) Evaluate(input string) Evaluation {
	if g == nil || g.cfg == nil || !g.cfg.Guard.Enabled {
		return Evaluation{Score: 0, Hits: nil, Threshold: math.Inf(1)}
	}

	if cached, ok := g.cache.Get(input); ok {
		return cached
	}

	value, _, _ := g.group.Do(input, func() (any, error) {
		result := g.evaluateInternal(input)
		g.cache.Set(input, result)
		return result, nil
	})

	if evaluation, ok := value.(Evaluation); ok {
		return evaluation
	}
	return Evaluation{Score: 0, Hits: nil, Threshold: g.threshold()}
}

// EnsureSafe: 위험 입력을 오류로 반환합니다.
func (g *InjectionGuard) EnsureSafe(input string) error {
	evaluation := g.Evaluate(input)
	if evaluation.Malicious() {
		return &BlockedError{Score: evaluation.Score, Threshold: evaluation.Threshold}
	}
	return nil
}

// IsMalicious: 입력이 위험한지 여부를 반환합니다.
func (g *InjectionGuard) IsMalicious(input string) bool {
	return g.Evaluate(input).Malicious()
}

func (g *InjectionGuard) loadRulepacks() {
	dir := g.cfg.Guard.RulepacksDir
	if dir == "" {
		dir = "rulepacks"
	}

	if !hasRulepackFiles(dir) {
		executable, err := os.Executable()
		if err == nil {
			fallback := filepath.Join(filepath.Dir(executable), "rulepacks")
			if hasRulepackFiles(fallback) {
				dir = fallback
			}
		}
	}

	g.packs = loadRulepacks(dir, g.logger)
	if g.logger != nil {
		g.logger.Info("guard_ready", "packs", len(g.packs), "threshold", g.threshold())
	}
}

func hasRulepackFiles(dir string) bool {
	entries := findRulepackFiles(dir)
	return len(entries) > 0
}

func (g *InjectionGuard) threshold() float64 {
	if g.cfg == nil {
		return 0.7
	}
	if g.cfg.Guard.Threshold > 0 {
		return g.cfg.Guard.Threshold
	}
	if len(g.packs) == 0 {
		return 0.7
	}

	maxThreshold := 0.0
	for _, pack := range g.packs {
		if pack.Threshold > maxThreshold {
			maxThreshold = pack.Threshold
		}
	}
	if maxThreshold > 0 {
		return maxThreshold
	}
	return 0.7
}

func (g *InjectionGuard) evaluateInternal(input string) Evaluation {
	threshold := g.threshold()

	if isJamoOnly(input) {
		if g.logger != nil {
			g.logger.Warn("guard_jamo_only_blocked", "input", trimForLog(input))
		}
		return Evaluation{
			Score:     threshold,
			Hits:      []Match{{ID: "jamo_only", Weight: threshold}},
			Threshold: threshold,
		}
	}

	if containsEmoji(input) {
		if g.logger != nil {
			g.logger.Warn("guard_emoji_blocked", "input", trimForLog(input))
		}
		return Evaluation{
			Score:     threshold,
			Hits:      []Match{{ID: "emoji_detected", Weight: threshold}},
			Threshold: threshold,
		}
	}

	if containsSuspiciousBase64(input) {
		if g.logger != nil {
			g.logger.Warn("guard_base64_payload_blocked", "input", trimForLog(input))
		}
		return Evaluation{
			Score:     threshold,
			Hits:      []Match{{ID: "base64_payload", Weight: threshold}},
			Threshold: threshold,
		}
	}

	// 정규화 파이프라인:
	// 1. 자모 시퀀스 조합 (ㅍㅡㄹㅗㅁㅍㅡㅌㅡ → 프롬프트)
	// 2. Homoglyph + NFKC 정규화
	composed := composeJamoSequences(input)
	normalized := normalizeText(composed)
	score, hits := g.evaluatePacks(normalized)
	return Evaluation{Score: score, Hits: hits, Threshold: threshold}
}

func (g *InjectionGuard) evaluatePacks(text string) (float64, []Match) {
	total := 0.0
	hits := make([]Match, 0)
	textLower := strings.ToLower(text)

	for _, pack := range g.packs {
		for _, rule := range pack.RegexRules {
			if rule.Pattern.MatchString(text) {
				total += rule.Weight
				hits = append(hits, Match{ID: rule.ID, Weight: rule.Weight})
			}
		}

		if pack.PhraseMatcher == nil {
			continue
		}
		matches := pack.PhraseMatcher.MatchThreadSafe([]byte(textLower))
		for _, index := range matches {
			if index < 0 || index >= len(pack.Phrases) {
				continue
			}
			phrase := pack.Phrases[index]
			weight := pack.PhraseWeights[phrase]
			if weight <= 0 {
				continue
			}
			total += weight
			hits = append(hits, Match{ID: "phrase:" + phrase, Weight: weight})
		}
	}

	return total, hits
}

func trimForLog(value string) string {
	value = strings.TrimSpace(value)
	if len(value) <= 50 {
		return value
	}
	return value[:50]
}
