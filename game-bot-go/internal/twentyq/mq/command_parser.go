package mq

import (
	"regexp"
	"strconv"
	"strings"

	qmodel "github.com/park285/llm-kakao-bots/game-bot-go/internal/twentyq/model"
)

// CommandParser 는 타입이다.
type CommandParser struct {
	prefix        string
	escapedPrefix string

	helpRe             *regexp.Regexp
	startRe            *regexp.Regexp
	hintRe             *regexp.Regexp
	surrenderRe        *regexp.Regexp
	agreeRe            *regexp.Regexp
	rejectRe           *regexp.Regexp
	statusRe           *regexp.Regexp
	modelInfoRe        *regexp.Regexp
	chainConditionalRe *regexp.Regexp
	chainRegularRe     *regexp.Regexp
	askRes             []*regexp.Regexp
	adminForceEndRe    *regexp.Regexp
	adminClearAllRe    *regexp.Regexp
	roomStatsRe        *regexp.Regexp
	userStatsRe        *regexp.Regexp
	usageRe            *regexp.Regexp
}

// NewCommandParser 는 동작을 수행한다.
func NewCommandParser(prefix string) *CommandParser {
	p := strings.TrimSpace(prefix)
	if p == "" {
		p = "/20q"
	}
	escapedPrefix := regexp.QuoteMeta(p)
	parser := &CommandParser{
		prefix:        p,
		escapedPrefix: escapedPrefix,
	}

	parser.helpRe = regexp.MustCompile("^" + escapedPrefix + `\s*$`)
	parser.startRe = regexp.MustCompile(
		"^" + escapedPrefix + `\s*(?:start|시작)(?:\s+(.+))?$`,
	)
	parser.hintRe = regexp.MustCompile(
		"^" + escapedPrefix + `\s*(?:hint|힌트|ㅎㅌ)(?:\s+(\d+))?$`,
	)
	parser.surrenderRe = regexp.MustCompile("^" + escapedPrefix + `\s*(?:surrender|하남자|포기)$`)
	parser.agreeRe = regexp.MustCompile("^" + escapedPrefix + `\s*(?:agree|동의)$`)
	parser.rejectRe = regexp.MustCompile("^" + escapedPrefix + `\s*(?:reject|거부)$`)
	parser.statusRe = regexp.MustCompile("^" + escapedPrefix + `\s*(?:status|상태|현황|상황|현재)$`)
	parser.modelInfoRe = regexp.MustCompile("^" + escapedPrefix + `\s*(?:모델|model)$`)
	parser.chainConditionalRe = regexp.MustCompile("(?i)^" + escapedPrefix + `\s+if\s+(.+,.+)$`)
	parser.chainRegularRe = regexp.MustCompile("^" + escapedPrefix + `\s+(.+,.+)$`)
	parser.askRes = []*regexp.Regexp{
		regexp.MustCompile("^" + escapedPrefix + `\s+(정답\s+.+)$`),
		regexp.MustCompile("^" + escapedPrefix + `\s*(?:ask|\?|질문)\s+(.+)$`),
		regexp.MustCompile("^" + escapedPrefix + `\s+(.+)$`),
	}
	parser.adminForceEndRe = regexp.MustCompile(
		"(?i)^" + escapedPrefix + `\s*(?:admin\s+force-end|관리자\s+강제종료)$`,
	)
	parser.adminClearAllRe = regexp.MustCompile(
		"(?i)^" + escapedPrefix + `\s*(?:admin\s+clear-all|관리자\s+전체삭제)$`,
	)
	parser.roomStatsRe = regexp.MustCompile(
		"(?i)^" + escapedPrefix + `\s*전적\s+룸(?:\s+(일간|주간|월간))?$`,
	)
	parser.userStatsRe = regexp.MustCompile("(?i)^" + escapedPrefix + `\s*전적(?:\s+(.+))?$`)

	const usagePeriodKeywords = `오늘|주간|월간|today|weekly|monthly`
	parser.usageRe = regexp.MustCompile(
		"(?i)^" + escapedPrefix + `\s*(?:사용량|usage)(?:\s+(` + usagePeriodKeywords + `))?(?:\s+(.+))?$`,
	)

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
	if cmd := p.parseAdmin(text); cmd != nil {
		return cmd
	}
	if cmd := p.parseStart(text); cmd != nil {
		return cmd
	}
	if cmd := p.parseHint(text); cmd != nil {
		return cmd
	}
	if cmd := p.parseUsage(text); cmd != nil {
		return cmd
	}
	if cmd := p.parseUserStats(text); cmd != nil {
		return cmd
	}
	if cmd := p.parseSurrender(text); cmd != nil {
		return cmd
	}
	if cmd := p.parseAgree(text); cmd != nil {
		return cmd
	}
	if cmd := p.parseReject(text); cmd != nil {
		return cmd
	}
	if cmd := p.parseStatus(text); cmd != nil {
		return cmd
	}
	if cmd := p.parseModelInfo(text); cmd != nil {
		return cmd
	}
	if cmd := p.parseChainedQuestion(text); cmd != nil {
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

	var categories []string
	if len(m) >= 2 && strings.TrimSpace(m[1]) != "" {
		parts := strings.Fields(m[1])
		for _, part := range parts {
			part = strings.TrimSpace(part)
			if part != "" {
				categories = append(categories, part)
			}
		}
	}

	return &Command{Kind: CommandStart, Categories: categories}
}

func (p *CommandParser) parseHint(text string) *Command {
	m := p.hintRe.FindStringSubmatch(text)
	if len(m) == 0 {
		return nil
	}

	count := 1
	if len(m) >= 2 {
		raw := strings.TrimSpace(m[1])
		if raw != "" {
			if v, err := strconv.Atoi(raw); err == nil {
				count = v
			}
		}
	}
	if count < 1 {
		count = 1
	}

	return &Command{Kind: CommandHints, HintCount: count}
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

func (p *CommandParser) parseReject(text string) *Command {
	if p.rejectRe.MatchString(text) {
		return &Command{Kind: CommandReject}
	}
	return nil
}

func (p *CommandParser) parseStatus(text string) *Command {
	if p.statusRe.MatchString(text) {
		return &Command{Kind: CommandStatus}
	}
	return nil
}

func (p *CommandParser) parseModelInfo(text string) *Command {
	if p.modelInfoRe.MatchString(text) {
		return &Command{Kind: CommandModelInfo}
	}
	return nil
}

// parseChainedQuestion 체인 질문 파싱.
// 쉼표가 포함된 질문을 체인 질문으로 인식.
// 예: "/스자 사람인가요, 살아있나요, 남자인가요"
// 조건부 예: "/스자 if 사람인가요, 직업인가요, 연예인인가요"
func (p *CommandParser) parseChainedQuestion(text string) *Command {
	// 조건부 체인 질문: /스자 if 질문1, 질문2, ...
	if m := p.chainConditionalRe.FindStringSubmatch(text); m != nil {
		body := strings.TrimSpace(m[1])
		questions := p.splitChainQuestions(body)
		if len(questions) >= 2 {
			return &Command{
				Kind:           CommandChainedQuestion,
				ChainQuestions: questions,
				ChainCondition: qmodel.ChainConditionIfTrue,
			}
		}
	}

	// 일반 체인 질문: /스자 질문1, 질문2, ...
	if m := p.chainRegularRe.FindStringSubmatch(text); m != nil {
		body := strings.TrimSpace(m[1])
		questions := p.splitChainQuestions(body)
		if len(questions) >= 2 {
			return &Command{
				Kind:           CommandChainedQuestion,
				ChainQuestions: questions,
				ChainCondition: qmodel.ChainConditionAlways,
			}
		}
	}

	return nil
}

// splitChainQuestions 쉼표로 구분된 질문 분리.
func (p *CommandParser) splitChainQuestions(body string) []string {
	parts := strings.Split(body, ",")
	var questions []string
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			questions = append(questions, trimmed)
		}
	}
	return questions
}

func (p *CommandParser) parseAsk(text string) *Command {
	for _, re := range p.askRes {
		m := re.FindStringSubmatch(text)
		if len(m) < 2 {
			continue
		}
		q := strings.TrimSpace(m[1])
		if q != "" {
			return &Command{Kind: CommandAsk, Question: q}
		}
	}
	return nil
}

// parseAdmin 관리자 명령어 파싱.
func (p *CommandParser) parseAdmin(text string) *Command {
	// /스자 admin force-end | 관리자 강제종료
	if p.adminForceEndRe.MatchString(text) {
		return &Command{Kind: CommandAdminForceEnd}
	}

	// /스자 admin clear-all | 관리자 전체삭제
	if p.adminClearAllRe.MatchString(text) {
		return &Command{Kind: CommandAdminClearAll}
	}

	return nil
}

// parseUserStats 개인/방 전적 파싱.
func (p *CommandParser) parseUserStats(text string) *Command {
	// 방 전적 조회 먼저 (더 구체적인 패턴) /스자 전적 룸 [일간|주간|월간]
	if m := p.roomStatsRe.FindStringSubmatch(text); m != nil {
		period := qmodel.StatsPeriodAll
		if len(m) >= 2 && m[1] != "" {
			switch strings.TrimSpace(m[1]) {
			case "일간":
				period = qmodel.StatsPeriodDaily
			case "주간":
				period = qmodel.StatsPeriodWeekly
			case "월간":
				period = qmodel.StatsPeriodMonthly
			}
		}
		return &Command{Kind: CommandRoomStats, RoomPeriod: period}
	}

	// 개인 전적 조회 /스자 전적 [닉네임]
	if m := p.userStatsRe.FindStringSubmatch(text); m != nil {
		var targetNickname *string
		if len(m) >= 2 && strings.TrimSpace(m[1]) != "" {
			nickname := strings.TrimSpace(m[1])
			targetNickname = &nickname
		}
		return &Command{Kind: CommandUserStats, TargetNickname: targetNickname}
	}

	return nil
}

// parseUsage 사용량(토큰) 조회 파싱.
func (p *CommandParser) parseUsage(text string) *Command {
	m := p.usageRe.FindStringSubmatch(text)
	if m == nil {
		return nil
	}

	period := qmodel.UsagePeriodToday
	if len(m) >= 2 && m[1] != "" {
		switch strings.ToLower(strings.TrimSpace(m[1])) {
		case "오늘", "today":
			period = qmodel.UsagePeriodToday
		case "주간", "weekly":
			period = qmodel.UsagePeriodWeekly
		case "월간", "monthly":
			period = qmodel.UsagePeriodMonthly
		}
	}

	var modelOverride *string
	if len(m) >= 3 && m[2] != "" {
		modelRaw := strings.ToLower(strings.TrimSpace(m[2]))
		modelCompact := strings.ReplaceAll(modelRaw, " ", "")
		switch modelCompact {
		case "2.5flash", "flash-25":
			v := "flash-25"
			modelOverride = &v
		case "3.0flash", "flash-30":
			v := "flash-30"
			modelOverride = &v
		case "2.5pro":
			v := "pro-25"
			modelOverride = &v
		case "3.0pro", "pro":
			v := "pro-30"
			modelOverride = &v
		}
	}

	return &Command{
		Kind:          CommandAdminUsage,
		UsagePeriod:   period,
		ModelOverride: modelOverride,
	}
}
