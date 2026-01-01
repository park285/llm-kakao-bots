package mq

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/park285/llm-kakao-bots/game-bot-go/internal/common/parser"
)

// CommandParser: 사용자 입력을 파싱하여 실행 가능한 명령어로 변환하는 파서
type CommandParser struct {
	parser.BaseParser

	helpRe      *regexp.Regexp
	startRe     *regexp.Regexp
	hintRe      *regexp.Regexp
	problemRe   *regexp.Regexp
	surrenderRe *regexp.Regexp
	agreeRe     *regexp.Regexp
	summaryRe   *regexp.Regexp
	answerRe    *regexp.Regexp
	askRe       *regexp.Regexp
}

// NewCommandParser: 주어진 접두사(prefix)를 사용하는 새로운 CommandParser를 생성합니다.
func NewCommandParser(prefix string) *CommandParser {
	base := parser.NewBaseParser(prefix, "/스프")
	p := &CommandParser{BaseParser: base}

	p.helpRe = p.BuildPattern(`\s*(?:도움|help)?$`)
	p.startRe = p.BuildPattern(`\s*(?:시작|start)(?:\s+(\S+))?$`)
	p.hintRe = p.BuildPattern(`\s*(?:힌트|hint)$`)
	p.problemRe = p.BuildPattern(`\s*(?:문제|제시문|problem)$`)
	p.surrenderRe = p.BuildPattern(`\s*(?:포기|surrender)$`)
	p.agreeRe = p.BuildPattern(`\s*(?:동의|agree)$`)
	p.summaryRe = p.BuildPattern(`\s*(?:정리|summary)$`)
	p.answerRe = p.BuildPattern(`\s*(?:정답|answer)\s+(.+)$`)
	p.askRe = p.BuildPattern(`\s+(.+)$`)

	return p
}

// Parse: 입력된 메시지 문자열을 파싱하여 Command 객체로 반환합니다.
func (p *CommandParser) Parse(message string) *Command {
	text := p.TrimMessage(message)
	if text == "" {
		return nil
	}

	if cmd := p.parseHelp(text); cmd != nil {
		return cmd
	}
	if cmd := p.parseStart(text); cmd != nil {
		return cmd
	}
	if cmd := p.parseHint(text); cmd != nil {
		return cmd
	}
	if cmd := p.parseProblem(text); cmd != nil {
		return cmd
	}
	if cmd := p.parseSurrender(text); cmd != nil {
		return cmd
	}
	if cmd := p.parseAgree(text); cmd != nil {
		return cmd
	}
	if cmd := p.parseSummary(text); cmd != nil {
		return cmd
	}
	if cmd := p.parseAnswer(text); cmd != nil {
		return cmd
	}
	if cmd := p.parseAsk(text); cmd != nil {
		return cmd
	}

	return &Command{Kind: CommandUnknown}
}

func (p *CommandParser) parseHelp(text string) *Command {
	if parser.MatchSimple(p.helpRe, text) {
		return &Command{Kind: CommandHelp}
	}
	return nil
}

func (p *CommandParser) parseStart(text string) *Command {
	m := p.startRe.FindStringSubmatch(text)
	if len(m) == 0 {
		return nil
	}

	rawInput := ""
	if len(m) >= 2 {
		rawInput = strings.TrimSpace(m[1])
	}
	var difficultyPtr *int
	hasInvalidInput := false

	if rawInput != "" {
		if v, err := strconv.Atoi(rawInput); err == nil {
			difficultyPtr = &v
		} else {
			hasInvalidInput = true
		}
	}

	return &Command{Kind: CommandStart, Difficulty: difficultyPtr, HasInvalidInput: hasInvalidInput}
}

func (p *CommandParser) parseHint(text string) *Command {
	if parser.MatchSimple(p.hintRe, text) {
		return &Command{Kind: CommandHint}
	}
	return nil
}

func (p *CommandParser) parseProblem(text string) *Command {
	if parser.MatchSimple(p.problemRe, text) {
		return &Command{Kind: CommandProblem}
	}
	return nil
}

func (p *CommandParser) parseSurrender(text string) *Command {
	if parser.MatchSimple(p.surrenderRe, text) {
		return &Command{Kind: CommandSurrender}
	}
	return nil
}

func (p *CommandParser) parseAgree(text string) *Command {
	if parser.MatchSimple(p.agreeRe, text) {
		return &Command{Kind: CommandAgree}
	}
	return nil
}

func (p *CommandParser) parseSummary(text string) *Command {
	if parser.MatchSimple(p.summaryRe, text) {
		return &Command{Kind: CommandSummary}
	}
	return nil
}

func (p *CommandParser) parseAnswer(text string) *Command {
	answer := parser.ExtractFirstGroup(p.answerRe, text)
	if answer == "" {
		return nil
	}
	return &Command{Kind: CommandAnswer, Answer: answer}
}

func (p *CommandParser) parseAsk(text string) *Command {
	question := parser.ExtractFirstGroup(p.askRe, text)
	if question == "" {
		return nil
	}
	return &Command{Kind: CommandAsk, Question: question}
}
