package mq

import (
	"regexp"
	"strconv"
	"strings"
)

// CommandParser 는 타입이다.
type CommandParser struct {
	prefix        string
	escapedPrefix string

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

// NewCommandParser 는 동작을 수행한다.
func NewCommandParser(prefix string) *CommandParser {
	p := strings.TrimSpace(prefix)
	if p == "" {
		p = "/스프"
	}
	escapedPrefix := regexp.QuoteMeta(p)
	parser := &CommandParser{
		prefix:        p,
		escapedPrefix: escapedPrefix,
	}

	parser.helpRe = regexp.MustCompile("^" + escapedPrefix + `\s*(?:도움|help)?$`)
	parser.startRe = regexp.MustCompile("^" + escapedPrefix + `\s*(?:시작|start)(?:\s+(\S+))?$`)
	parser.hintRe = regexp.MustCompile("^" + escapedPrefix + `\s*(?:힌트|hint)$`)
	parser.problemRe = regexp.MustCompile("^" + escapedPrefix + `\s*(?:문제|제시문|problem)$`)
	parser.surrenderRe = regexp.MustCompile("^" + escapedPrefix + `\s*(?:포기|surrender)$`)
	parser.agreeRe = regexp.MustCompile("^" + escapedPrefix + `\s*(?:동의|agree)$`)
	parser.summaryRe = regexp.MustCompile("^" + escapedPrefix + `\s*(?:정리|summary)$`)
	parser.answerRe = regexp.MustCompile("^" + escapedPrefix + `\s*(?:정답|answer)\s+(.+)$`)
	parser.askRe = regexp.MustCompile("^" + escapedPrefix + `\s+(.+)$`)

	return parser
}

// Parse 는 동작을 수행한다.
func (p *CommandParser) Parse(message string) *Command {
	text := strings.TrimSpace(message)
	if text == "" {
		return nil
	}
	if !strings.HasPrefix(text, p.prefix) {
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
	if p.helpRe.MatchString(text) {
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
	if p.hintRe.MatchString(text) {
		return &Command{Kind: CommandHint}
	}
	return nil
}

func (p *CommandParser) parseProblem(text string) *Command {
	if p.problemRe.MatchString(text) {
		return &Command{Kind: CommandProblem}
	}
	return nil
}

func (p *CommandParser) parseSurrender(text string) *Command {
	if p.surrenderRe.MatchString(text) {
		return &Command{Kind: CommandSurrender}
	}
	return nil
}

func (p *CommandParser) parseAgree(text string) *Command {
	if p.agreeRe.MatchString(text) {
		return &Command{Kind: CommandAgree}
	}
	return nil
}

func (p *CommandParser) parseSummary(text string) *Command {
	if p.summaryRe.MatchString(text) {
		return &Command{Kind: CommandSummary}
	}
	return nil
}

func (p *CommandParser) parseAnswer(text string) *Command {
	m := p.answerRe.FindStringSubmatch(text)
	if len(m) < 2 {
		return nil
	}
	answer := strings.TrimSpace(m[1])
	if answer == "" {
		return nil
	}
	return &Command{Kind: CommandAnswer, Answer: answer}
}

func (p *CommandParser) parseAsk(text string) *Command {
	m := p.askRe.FindStringSubmatch(text)
	if len(m) < 2 {
		return nil
	}
	question := strings.TrimSpace(m[1])
	if question == "" {
		return nil
	}
	return &Command{Kind: CommandAsk, Question: question}
}
