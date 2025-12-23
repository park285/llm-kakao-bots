package guard

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/cloudflare/ahocorasick"
	"gopkg.in/yaml.v3"
)

type rawRulepack struct {
	Version     int       `yaml:"version"`
	Threshold   float64   `yaml:"threshold"`
	Normalizers []string  `yaml:"normalizers"`
	Rules       []rawRule `yaml:"rules"`
}

type rawRule struct {
	ID      string   `yaml:"id"`
	Type    string   `yaml:"type"`
	Pattern string   `yaml:"pattern"`
	Phrases []string `yaml:"phrases"`
	Weight  float64  `yaml:"weight"`
}

type regexRule struct {
	ID      string
	Pattern *regexp.Regexp
	Weight  float64
}

type compiledPack struct {
	Threshold     float64
	RegexRules    []regexRule
	PhraseMatcher *ahocorasick.Matcher
	Phrases       []string
	PhraseWeights map[string]float64
}

func loadRulepacks(dir string, logger *slog.Logger) []compiledPack {
	paths := findRulepackFiles(dir)
	if len(paths) == 0 {
		if logger != nil {
			logger.Warn("rulepacks_not_found", "dir", dir)
		}
		return nil
	}

	packs := make([]compiledPack, 0, len(paths))
	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			if logger != nil {
				logger.Warn("rulepack_read_failed", "path", path, "err", err)
			}
			continue
		}

		var raw rawRulepack
		err = yaml.Unmarshal(data, &raw)
		if err != nil {
			if logger != nil {
				logger.Warn("rulepack_parse_failed", "path", path, "err", err)
			}
			continue
		}

		pack, err := compileRulepack(raw, logger)
		if err != nil {
			if logger != nil {
				logger.Warn("rulepack_compile_failed", "path", path, "err", err)
			}
			continue
		}
		packs = append(packs, pack)
	}

	return packs
}

func findRulepackFiles(dir string) []string {
	var files []string
	patterns := []string{"*.yml", "*.yaml"}
	for _, pattern := range patterns {
		matches, err := filepath.Glob(filepath.Join(dir, pattern))
		if err != nil {
			continue
		}
		files = append(files, matches...)
	}
	return files
}

func compileRulepack(raw rawRulepack, logger *slog.Logger) (compiledPack, error) {
	if raw.Version == 0 {
		raw.Version = 1
	}
	if raw.Threshold == 0 {
		raw.Threshold = 0.7
	}

	var regexes []regexRule
	phrases := make([]string, 0)
	phraseWeights := make(map[string]float64)

	for _, rule := range raw.Rules {
		switch strings.ToLower(strings.TrimSpace(rule.Type)) {
		case "regex":
			if rule.ID == "" || rule.Pattern == "" {
				return compiledPack{}, fmt.Errorf("invalid regex rule")
			}
			pattern, err := regexp.Compile("(?i)" + rule.Pattern)
			if err != nil {
				if logger != nil {
					logger.Warn("rulepack_regex_invalid", "rule_id", rule.ID, "err", err)
				}
				continue
			}
			regexes = append(regexes, regexRule{
				ID:      rule.ID,
				Pattern: pattern,
				Weight:  rule.Weight,
			})
		case "phrases":
			if rule.ID == "" || len(rule.Phrases) == 0 {
				return compiledPack{}, fmt.Errorf("invalid phrases rule")
			}
			for _, phrase := range rule.Phrases {
				value := strings.ToLower(phrase)
				phrases = append(phrases, value)
				phraseWeights[value] = rule.Weight
			}
		default:
			return compiledPack{}, fmt.Errorf("unknown rule type: %s", rule.Type)
		}
	}

	var matcher *ahocorasick.Matcher
	if len(phrases) > 0 {
		patterns := make([][]byte, 0, len(phrases))
		for _, phrase := range phrases {
			patterns = append(patterns, []byte(phrase))
		}
		matcher = ahocorasick.NewMatcher(patterns)
	}

	return compiledPack{
		Threshold:     raw.Threshold,
		RegexRules:    regexes,
		PhraseMatcher: matcher,
		Phrases:       phrases,
		PhraseWeights: phraseWeights,
	}, nil
}
