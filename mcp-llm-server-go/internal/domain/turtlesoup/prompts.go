package turtlesoup

import (
	"embed"
	"fmt"
	"strconv"

	"github.com/park285/llm-kakao-bots/mcp-llm-server-go/internal/prompt"
)

//go:embed prompts/*.yml
var promptsFS embed.FS

// Prompts 는 Turtle Soup 프롬프트 모음이다.
type Prompts struct {
	prompts map[string]map[string]string
}

// NewPrompts: Turtle Soup 프롬프트를 로드합니다.
func NewPrompts() (*Prompts, error) {
	loaded, err := prompt.LoadYAMLDir(promptsFS, "prompts")
	if err != nil {
		return nil, fmt.Errorf("load turtlesoup prompts: %w", err)
	}
	return &Prompts{prompts: loaded}, nil
}

// AnswerSystem: 정답 시스템 프롬프트를 반환합니다.
func (p *Prompts) AnswerSystem() (string, error) {
	data, err := p.getPrompt("answer")
	if err != nil {
		return "", err
	}
	return promptField(data, "system", "answer.system")
}

// AnswerSystemWithPuzzle: 퍼즐 정보를 포함한 시스템 프롬프트를 반환합니다.
// 암시적 캐싱 최적화: 퍼즐 정보를 System Prompt에 통합하여 Static Prefix를 확보합니다.
// Toon 포맷이 이미 구조화되어 있으므로 XML 래핑 불필요
func (p *Prompts) AnswerSystemWithPuzzle(puzzle string) (string, error) {
	base, err := p.AnswerSystem()
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s\n\n[이번 게임의 시나리오]\n%s\n(이 내용은 사용자에게 절대 직접 노출하지 마시오.)",
		base, puzzle), nil // XML 래핑 제거
}

// AnswerUser: 정답 유저 프롬프트를 반환합니다.
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
	formatted, err := prompt.FormatTemplate(template, map[string]string{
		"question": prompt.WrapXML("question", question),
	})
	if err != nil {
		return "", fmt.Errorf("format answer.user: %w", err)
	}
	return formatted, nil
}

// HintSystem: 힌트 시스템 프롬프트를 반환합니다.
func (p *Prompts) HintSystem() (string, error) {
	data, err := p.getPrompt("hint")
	if err != nil {
		return "", err
	}
	return promptField(data, "system", "hint.system")
}

// HintUser: 힌트 유저 프롬프트를 반환합니다.
func (p *Prompts) HintUser(puzzle string, level int) (string, error) {
	data, err := p.getPrompt("hint")
	if err != nil {
		return "", err
	}
	template, err := promptField(data, "user", "hint.user")
	if err != nil {
		return "", err
	}
	formatted, err := prompt.FormatTemplate(template, map[string]string{
		"puzzle": prompt.WrapXML("puzzle", puzzle),
		"level":  strconv.Itoa(level),
	})
	if err != nil {
		return "", fmt.Errorf("format hint.user: %w", err)
	}
	return formatted, nil
}

// ValidateSystem: 검증 시스템 프롬프트를 반환합니다.
func (p *Prompts) ValidateSystem() (string, error) {
	data, err := p.getPrompt("validate")
	if err != nil {
		return "", err
	}
	return promptField(data, "system", "validate.system")
}

// ValidateUser: 검증 유저 프롬프트를 반환합니다.
func (p *Prompts) ValidateUser(solution string, playerAnswer string) (string, error) {
	data, err := p.getPrompt("validate")
	if err != nil {
		return "", err
	}
	template, err := promptField(data, "user", "validate.user")
	if err != nil {
		return "", err
	}
	formatted, err := prompt.FormatTemplate(template, map[string]string{
		"solution":      prompt.WrapXML("solution", solution),
		"player_answer": prompt.WrapXML("player_answer", playerAnswer),
	})
	if err != nil {
		return "", fmt.Errorf("format validate.user: %w", err)
	}
	return formatted, nil
}

// RevealSystem: 해설 시스템 프롬프트를 반환합니다.
func (p *Prompts) RevealSystem() (string, error) {
	data, err := p.getPrompt("reveal")
	if err != nil {
		return "", err
	}
	return promptField(data, "system", "reveal.system")
}

// RevealUser: 해설 유저 프롬프트를 반환합니다.
func (p *Prompts) RevealUser(puzzle string) (string, error) {
	data, err := p.getPrompt("reveal")
	if err != nil {
		return "", err
	}
	template, err := promptField(data, "user", "reveal.user")
	if err != nil {
		return "", err
	}
	formatted, err := prompt.FormatTemplate(template, map[string]string{
		"puzzle": prompt.WrapXML("puzzle", puzzle),
	})
	if err != nil {
		return "", fmt.Errorf("format reveal.user: %w", err)
	}
	return formatted, nil
}

// GenerateSystem: 퍼즐 생성 시스템 프롬프트를 반환합니다.
func (p *Prompts) GenerateSystem() (string, error) {
	data, err := p.getPrompt("generate")
	if err != nil {
		return "", err
	}
	return promptField(data, "system", "generate.system")
}

// GenerateUser: 퍼즐 생성 유저 프롬프트를 반환합니다.
func (p *Prompts) GenerateUser(category string, difficulty int, theme string, examples string) (string, error) {
	data, err := p.getPrompt("generate")
	if err != nil {
		return "", err
	}
	template, err := promptField(data, "user", "generate.user")
	if err != nil {
		return "", err
	}
	formatted, err := prompt.FormatTemplate(template, map[string]string{
		"category":   prompt.WrapXML("category", category),
		"difficulty": strconv.Itoa(difficulty),
		"theme":      prompt.WrapXML("theme", theme),
		"examples":   examples,
	})
	if err != nil {
		return "", fmt.Errorf("format generate.user: %w", err)
	}
	return formatted, nil
}

// RewriteSystem: 리라이트 시스템 프롬프트를 반환합니다.
func (p *Prompts) RewriteSystem() (string, error) {
	data, err := p.getPrompt("rewrite")
	if err != nil {
		return "", err
	}
	return promptField(data, "system", "rewrite.system")
}

// RewriteUser: 리라이트 유저 프롬프트를 반환합니다.
func (p *Prompts) RewriteUser(title string, scenario string, solution string, difficulty int) (string, error) {
	data, err := p.getPrompt("rewrite")
	if err != nil {
		return "", err
	}
	template, err := promptField(data, "user", "rewrite.user")
	if err != nil {
		return "", err
	}
	formatted, err := prompt.FormatTemplate(template, map[string]string{
		"title":      prompt.WrapXML("title", title),
		"scenario":   prompt.WrapXML("scenario", scenario),
		"solution":   prompt.WrapXML("solution", solution),
		"difficulty": strconv.Itoa(difficulty),
	})
	if err != nil {
		return "", fmt.Errorf("format rewrite.user: %w", err)
	}
	return formatted, nil
}

func (p *Prompts) getPrompt(name string) (map[string]string, error) {
	if p == nil {
		return nil, fmt.Errorf("turtlesoup prompts not initialized")
	}
	promptMap, err := prompt.Get(p.prompts, name, "turtlesoup")
	if err != nil {
		return nil, fmt.Errorf("get turtlesoup prompt %s: %w", name, err)
	}
	return promptMap, nil
}

func promptField(data map[string]string, key string, label string) (string, error) {
	value, err := prompt.Field(data, key, label)
	if err != nil {
		return "", fmt.Errorf("get turtlesoup prompt field %s: %w", label, err)
	}
	return value, nil
}
