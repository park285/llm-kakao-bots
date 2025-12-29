package twentyq

import (
	"embed"
	"fmt"
	"strings"

	"github.com/park285/llm-kakao-bots/mcp-llm-server-go/internal/prompt"
)

//go:embed prompts/*.yml
var promptsFS embed.FS

// Prompts: TwentyQ 프롬프트 모음입니다.
type Prompts struct {
	prompts map[string]map[string]string
}

// NewPrompts: TwentyQ 프롬프트를 로드합니다.
func NewPrompts() (*Prompts, error) {
	loaded, err := prompt.LoadYAMLDir(promptsFS, "prompts")
	if err != nil {
		return nil, fmt.Errorf("load twentyq prompts: %w", err)
	}
	return &Prompts{prompts: loaded}, nil
}

// HintsSystem: 힌트 시스템 프롬프트를 반환합니다.
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

// HintsUser: 힌트 유저 프롬프트를 반환합니다.
func (p *Prompts) HintsUser(secret string) (string, error) {
	data, err := p.getPrompt("hints")
	if err != nil {
		return "", err
	}
	template, err := promptField(data, "user", "hints.user")
	if err != nil {
		return "", err
	}
	formatted, err := prompt.FormatTemplate(template, map[string]string{
		"toon": prompt.WrapXML("secret", secret),
	})
	if err != nil {
		return "", fmt.Errorf("format hints.user: %w", err)
	}
	return formatted, nil
}

// AnswerSystem: 답변 시스템 프롬프트를 반환합니다.
func (p *Prompts) AnswerSystem() (string, error) {
	data, err := p.getPrompt("answer")
	if err != nil {
		return "", err
	}
	return promptField(data, "system", "answer.system")
}

// AnswerSystemWithSecret: Secret 정보를 포함한 시스템 프롬프트를 반환합니다.
// 암시적 캐싱 최적화: Secret 정보를 System Prompt에 통합하여 Static Prefix를 확보합니다.
// Toon 포맷이 이미 구조화되어 있으므로 XML 래핑 불필요
func (p *Prompts) AnswerSystemWithSecret(secret string) (string, error) {
	base, err := p.AnswerSystem()
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s\n\n[이번 게임의 정답]\n%s\n(이 내용은 사용자에게 절대 직접 노출하지 마시오.)",
		base, secret), nil // XML 래핑 제거
}

// AnswerUser: 답변 유저 프롬프트를 반환합니다.
// 암시적 캐싱 최적화: 현재 질문만 포함하여 Cache Miss 영역을 최소화합니다.
func (p *Prompts) AnswerUser(question string) (string, error) {
	data, err := p.getPrompt("answer")
	if err != nil {
		return "", err
	}
	template, err := promptField(data, "user", "answer.user")
	if err != nil {
		return "", err
	}
	result, err := prompt.FormatTemplate(template, map[string]string{
		"question": prompt.WrapXML("question", question),
	})
	if err != nil {
		return "", fmt.Errorf("format answer.user: %w", err)
	}
	return result, nil
}

// VerifySystem: 검증 시스템 프롬프트를 반환합니다.
func (p *Prompts) VerifySystem() (string, error) {
	data, err := p.getPrompt("verify-answer")
	if err != nil {
		return "", err
	}
	return promptField(data, "system", "verify-answer.system")
}

// VerifyUser: 검증 유저 프롬프트를 반환합니다.
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
		"target": prompt.WrapXML("target", target),
		"guess":  prompt.WrapXML("guess", guess),
	})
	if err != nil {
		return "", fmt.Errorf("format verify-answer.user: %w", err)
	}
	return formatted, nil
}

// NormalizeSystem: 정규화 시스템 프롬프트를 반환합니다.
func (p *Prompts) NormalizeSystem() (string, error) {
	data, err := p.getPrompt("normalize")
	if err != nil {
		return "", err
	}
	return promptField(data, "system", "normalize.system")
}

// NormalizeUser: 정규화 유저 프롬프트를 반환합니다.
func (p *Prompts) NormalizeUser(question string) (string, error) {
	data, err := p.getPrompt("normalize")
	if err != nil {
		return "", err
	}
	template, err := promptField(data, "user", "normalize.user")
	if err != nil {
		return "", err
	}
	formatted, err := prompt.FormatTemplate(template, map[string]string{
		"question": prompt.WrapXML("question", question),
	})
	if err != nil {
		return "", fmt.Errorf("format normalize.user: %w", err)
	}
	return formatted, nil
}

// SynonymSystem: 유사어 시스템 프롬프트를 반환합니다.
func (p *Prompts) SynonymSystem() (string, error) {
	data, err := p.getPrompt("synonym-check")
	if err != nil {
		return "", err
	}
	return promptField(data, "system", "synonym-check.system")
}

// SynonymUser: 유사어 유저 프롬프트를 반환합니다.
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
		"target": prompt.WrapXML("target", target),
		"guess":  prompt.WrapXML("guess", guess),
	})
	if err != nil {
		return "", fmt.Errorf("format synonym-check.user: %w", err)
	}
	return formatted, nil
}

func (p *Prompts) getPrompt(name string) (map[string]string, error) {
	if p == nil {
		return nil, fmt.Errorf("twentyq prompts not initialized")
	}
	promptMap, err := prompt.Get(p.prompts, name, "twentyq")
	if err != nil {
		return nil, fmt.Errorf("get twentyq prompt %s: %w", name, err)
	}
	return promptMap, nil
}

func promptField(data map[string]string, key string, label string) (string, error) {
	value, err := prompt.Field(data, key, label)
	if err != nil {
		return "", fmt.Errorf("get twentyq prompt field %s: %w", label, err)
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
