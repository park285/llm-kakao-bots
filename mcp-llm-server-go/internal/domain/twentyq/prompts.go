package twentyq

import (
	"embed"
	"fmt"
	"strings"

	"github.com/park285/llm-kakao-bots/mcp-llm-server-go/internal/prompt"
)

//go:embed prompts/*.yml
var promptsFS embed.FS

// Prompts 는 TwentyQ 프롬프트 모음이다.
type Prompts struct {
	prompts map[string]map[string]string
}

// NewPrompts 는 TwentyQ 프롬프트를 로드한다.
func NewPrompts() (*Prompts, error) {
	loaded, err := prompt.LoadYAMLDir(promptsFS, "prompts")
	if err != nil {
		return nil, fmt.Errorf("load twentyq prompts: %w", err)
	}
	return &Prompts{prompts: loaded}, nil
}

// HintsSystem 은 힌트 시스템 프롬프트를 반환한다.
func (p *Prompts) HintsSystem(category string) (string, error) {
	data, err := p.getPrompt("hints")
	if err != nil {
		return "", err
	}
	system, err := promptField(data, "system", "hints.system")
	if err != nil {
		return "", err
	}
	if category == "" {
		return system, nil
	}

	restriction, ok := data["category_restriction"]
	if !ok || restriction == "" {
		return system, nil
	}

	forbidden := strings.Join(getForbiddenWords(category), ", ")
	formatted, err := prompt.FormatTemplate(restriction, map[string]string{
		"selectedCategory": category,
		"forbiddenWords":   forbidden,
	})
	if err != nil {
		return "", fmt.Errorf("format hints restriction: %w", err)
	}

	return system + "\n\n" + formatted, nil
}

// HintsUser 는 힌트 유저 프롬프트를 반환한다.
func (p *Prompts) HintsUser(secret string) (string, error) {
	data, err := p.getPrompt("hints")
	if err != nil {
		return "", err
	}
	template, err := promptField(data, "user", "hints.user")
	if err != nil {
		return "", err
	}
	formatted, err := prompt.FormatTemplate(template, map[string]string{"toon": secret})
	if err != nil {
		return "", fmt.Errorf("format hints.user: %w", err)
	}
	return formatted, nil
}

// AnswerSystem 은 답변 시스템 프롬프트를 반환한다.
func (p *Prompts) AnswerSystem() (string, error) {
	data, err := p.getPrompt("answer")
	if err != nil {
		return "", err
	}
	return promptField(data, "system", "answer.system")
}

// AnswerUser 는 답변 유저 프롬프트를 반환한다.
func (p *Prompts) AnswerUser(secret string, question string, history string) (string, error) {
	data, err := p.getPrompt("answer")
	if err != nil {
		return "", err
	}
	template, err := promptField(data, "user", "answer.user")
	if err != nil {
		return "", err
	}
	result, err := prompt.FormatTemplate(template, map[string]string{
		"toon":     secret,
		"question": question,
	})
	if err != nil {
		return "", fmt.Errorf("format answer.user: %w", err)
	}
	if history != "" {
		result = history + "\n\n" + result
	}
	return result, nil
}

// VerifySystem 은 검증 시스템 프롬프트를 반환한다.
func (p *Prompts) VerifySystem() (string, error) {
	data, err := p.getPrompt("verify-answer")
	if err != nil {
		return "", err
	}
	return promptField(data, "system", "verify-answer.system")
}

// VerifyUser 는 검증 유저 프롬프트를 반환한다.
func (p *Prompts) VerifyUser(target string, guess string) (string, error) {
	data, err := p.getPrompt("verify-answer")
	if err != nil {
		return "", err
	}
	template, err := promptField(data, "user", "verify-answer.user")
	if err != nil {
		return "", err
	}
	formatted, err := prompt.FormatTemplate(template, map[string]string{
		"target": target,
		"guess":  guess,
	})
	if err != nil {
		return "", fmt.Errorf("format verify-answer.user: %w", err)
	}
	return formatted, nil
}

// NormalizeSystem 은 정규화 시스템 프롬프트를 반환한다.
func (p *Prompts) NormalizeSystem() (string, error) {
	data, err := p.getPrompt("normalize")
	if err != nil {
		return "", err
	}
	return promptField(data, "system", "normalize.system")
}

// NormalizeUser 는 정규화 유저 프롬프트를 반환한다.
func (p *Prompts) NormalizeUser(question string) (string, error) {
	data, err := p.getPrompt("normalize")
	if err != nil {
		return "", err
	}
	template, err := promptField(data, "user", "normalize.user")
	if err != nil {
		return "", err
	}
	formatted, err := prompt.FormatTemplate(template, map[string]string{"question": question})
	if err != nil {
		return "", fmt.Errorf("format normalize.user: %w", err)
	}
	return formatted, nil
}

// SynonymSystem 은 유사어 시스템 프롬프트를 반환한다.
func (p *Prompts) SynonymSystem() (string, error) {
	data, err := p.getPrompt("synonym-check")
	if err != nil {
		return "", err
	}
	return promptField(data, "system", "synonym-check.system")
}

// SynonymUser 는 유사어 유저 프롬프트를 반환한다.
func (p *Prompts) SynonymUser(target string, guess string) (string, error) {
	data, err := p.getPrompt("synonym-check")
	if err != nil {
		return "", err
	}
	template, err := promptField(data, "user", "synonym-check.user")
	if err != nil {
		return "", err
	}
	formatted, err := prompt.FormatTemplate(template, map[string]string{
		"target": target,
		"guess":  guess,
	})
	if err != nil {
		return "", fmt.Errorf("format synonym-check.user: %w", err)
	}
	return formatted, nil
}

func (p *Prompts) getPrompt(name string) (map[string]string, error) {
	if p == nil || p.prompts == nil {
		return nil, fmt.Errorf("twentyq prompts not initialized")
	}
	promptMap, ok := p.prompts[name]
	if !ok {
		return nil, fmt.Errorf("prompt not found: %s", name)
	}
	return promptMap, nil
}

func promptField(data map[string]string, key string, label string) (string, error) {
	value, ok := data[key]
	if !ok {
		return "", fmt.Errorf("prompt field missing: %s", label)
	}
	return value, nil
}

func getForbiddenWords(category string) []string {
	categoryForbidden := map[string][]string{
		"음식": {"음식", "먹을 것", "식품"},
		"동물": {"동물", "생물", "생명체"},
		"사물": {"사물", "물건", "도구"},
		"장소": {"장소", "곳", "위치"},
		"인물": {"인물", "사람", "인간"},
		"개념": {"개념", "추상적", "관념"},
	}
	if forbidden, ok := categoryForbidden[category]; ok {
		return forbidden
	}
	return []string{category}
}
