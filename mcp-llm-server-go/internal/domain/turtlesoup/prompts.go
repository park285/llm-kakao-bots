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

// NewPrompts 는 Turtle Soup 프롬프트를 로드한다.
func NewPrompts() (*Prompts, error) {
	loaded, err := prompt.LoadYAMLDir(promptsFS, "prompts")
	if err != nil {
		return nil, fmt.Errorf("load turtlesoup prompts: %w", err)
	}
	return &Prompts{prompts: loaded}, nil
}

// AnswerSystem 은 정답 시스템 프롬프트를 반환한다.
func (p *Prompts) AnswerSystem() (string, error) {
	data, err := p.getPrompt("answer")
	if err != nil {
		return "", err
	}
	return promptField(data, "system", "answer.system")
}

// AnswerUser 는 정답 유저 프롬프트를 반환한다.
func (p *Prompts) AnswerUser(puzzle string, question string, history string) (string, error) {
	data, err := p.getPrompt("answer")
	if err != nil {
		return "", err
	}
	template, err := promptField(data, "user", "answer.user")
	if err != nil {
		return "", err
	}
	formatted, err := prompt.FormatTemplate(template, map[string]string{
		"puzzle":   puzzle,
		"history":  history,
		"question": question,
	})
	if err != nil {
		return "", fmt.Errorf("format answer.user: %w", err)
	}
	return formatted, nil
}

// HintSystem 은 힌트 시스템 프롬프트를 반환한다.
func (p *Prompts) HintSystem() (string, error) {
	data, err := p.getPrompt("hint")
	if err != nil {
		return "", err
	}
	return promptField(data, "system", "hint.system")
}

// HintUser 는 힌트 유저 프롬프트를 반환한다.
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
		"puzzle": puzzle,
		"level":  strconv.Itoa(level),
	})
	if err != nil {
		return "", fmt.Errorf("format hint.user: %w", err)
	}
	return formatted, nil
}

// ValidateSystem 은 검증 시스템 프롬프트를 반환한다.
func (p *Prompts) ValidateSystem() (string, error) {
	data, err := p.getPrompt("validate")
	if err != nil {
		return "", err
	}
	return promptField(data, "system", "validate.system")
}

// ValidateUser 는 검증 유저 프롬프트를 반환한다.
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
		"solution":      solution,
		"player_answer": playerAnswer,
	})
	if err != nil {
		return "", fmt.Errorf("format validate.user: %w", err)
	}
	return formatted, nil
}

// RevealSystem 은 해설 시스템 프롬프트를 반환한다.
func (p *Prompts) RevealSystem() (string, error) {
	data, err := p.getPrompt("reveal")
	if err != nil {
		return "", err
	}
	return promptField(data, "system", "reveal.system")
}

// RevealUser 는 해설 유저 프롬프트를 반환한다.
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
		"puzzle": puzzle,
	})
	if err != nil {
		return "", fmt.Errorf("format reveal.user: %w", err)
	}
	return formatted, nil
}

// GenerateSystem 은 퍼즐 생성 시스템 프롬프트를 반환한다.
func (p *Prompts) GenerateSystem() (string, error) {
	data, err := p.getPrompt("generate")
	if err != nil {
		return "", err
	}
	return promptField(data, "system", "generate.system")
}

// GenerateUser 는 퍼즐 생성 유저 프롬프트를 반환한다.
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
		"category":   category,
		"difficulty": strconv.Itoa(difficulty),
		"theme":      theme,
		"examples":   examples,
	})
	if err != nil {
		return "", fmt.Errorf("format generate.user: %w", err)
	}
	return formatted, nil
}

// RewriteSystem 은 리라이트 시스템 프롬프트를 반환한다.
func (p *Prompts) RewriteSystem() (string, error) {
	data, err := p.getPrompt("rewrite")
	if err != nil {
		return "", err
	}
	return promptField(data, "system", "rewrite.system")
}

// RewriteUser 는 리라이트 유저 프롬프트를 반환한다.
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
		"title":      title,
		"scenario":   scenario,
		"solution":   solution,
		"difficulty": strconv.Itoa(difficulty),
	})
	if err != nil {
		return "", fmt.Errorf("format rewrite.user: %w", err)
	}
	return formatted, nil
}

func (p *Prompts) getPrompt(name string) (map[string]string, error) {
	if p == nil || p.prompts == nil {
		return nil, fmt.Errorf("turtlesoup prompts not initialized")
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
