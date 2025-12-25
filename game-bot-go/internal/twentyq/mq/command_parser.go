package mq

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/park285/llm-kakao-bots/game-bot-go/internal/common/parser"
	qmodel "github.com/park285/llm-kakao-bots/game-bot-go/internal/twentyq/model"
)

// CommandParser: 사용자 입력 메시지에서 정규식을 이용해 게임 명령어를 추출하고 파싱하는 처리기
type CommandParser struct {
	parser.BaseParser

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

// NewCommandParser: 주어진 접두사(prefix)를 기반으로 정규식 패턴들을 초기화하여 새로운 CommandParser를 생성한다.
func NewCommandParser(prefix string) *CommandParser {
	base := parser.NewBaseParser(prefix, "/20q")
	p := &CommandParser{BaseParser: base}

	p.helpRe = p.BuildPattern(`\s*$`)
	p.startRe = p.BuildPattern(`\s*(?:start|시작)(?:\s+(.+))?$`)
	p.hintRe = p.BuildPattern(`\s*(?:hint|힌트|ㅎㅌ)(?:\s+(\d+))?$`)
	p.surrenderRe = p.BuildPattern(`\s*(?:surrender|하남자|포기)$`)
	p.agreeRe = p.BuildPattern(`\s*(?:agree|동의)$`)
	p.rejectRe = p.BuildPattern(`\s*(?:reject|거부)$`)
	p.statusRe = p.BuildPattern(`\s*(?:status|상태|현황|상황|현재)$`)
	p.modelInfoRe = p.BuildPattern(`\s*(?:모델|model)$`)
	p.chainConditionalRe = p.BuildPatternCaseInsensitive(`\s+if\s+(.+,.+)$`)
	p.chainRegularRe = p.BuildPattern(`\s+(.+,.+)$`)
	p.askRes = []*regexp.Regexp{
		p.BuildPattern(`\s+(정답\s+.+)$`),
		p.BuildPattern(`\s*(?:ask|\?|질문)\s+(.+)$`),
		p.BuildPattern(`\s+(.+)$`),
	}
	p.adminForceEndRe = p.BuildPatternCaseInsensitive(`\s*(?:admin\s+force-end|관리자\s+강제종료)$`)
	p.adminClearAllRe = p.BuildPatternCaseInsensitive(`\s*(?:admin\s+clear-all|관리자\s+전체삭제)$`)
	p.roomStatsRe = p.BuildPatternCaseInsensitive(`\s*전적\s+룸(?:\s+(일간|주간|월간))?$`)
	p.userStatsRe = p.BuildPatternCaseInsensitive(`\s*전적(?:\s+(.+))?$`)

	const usagePeriodKeywords = `오늘|주간|월간|today|weekly|monthly`
	p.usageRe = p.BuildPatternCaseInsensitive(`\s*(?:사용량|usage)(?:\s+(` + usagePeriodKeywords + `))?(?:\s+(.+))?$`)

	return p
}

// Parse: 입력된 메시지 문자열을 분석하여 해당하는 Command 객체로 반환한다.
func (p *CommandParser) Parse(message string) *Command {
	text := p.TrimMessage(message)
	if text == "" {
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

func (p *CommandParser) parseReject(text string) *Command {
	if parser.MatchSimple(p.rejectRe, text) {
		return &Command{Kind: CommandReject}
	}
	return nil
}

func (p *CommandParser) parseStatus(text string) *Command {
	if parser.MatchSimple(p.statusRe, text) {
		return &Command{Kind: CommandStatus}
	}
	return nil
}

func (p *CommandParser) parseModelInfo(text string) *Command {
	if parser.MatchSimple(p.modelInfoRe, text) {
		return &Command{Kind: CommandModelInfo}
	}
	return nil
}

// parseChainedQuestion: 쉼표(,)로 구분된 여러 질문을 포함하는 '체인 질문'을 파싱한다.
func (p *CommandParser) parseChainedQuestion(text string) *Command {
	// 조건부 체인 질문: /스자 if 질문1, 질문2, ...
	if body := parser.ExtractFirstGroup(p.chainConditionalRe, text); body != "" {
		questions := parser.SplitByComma(body)
		if len(questions) >= 2 {
			return &Command{
				Kind:           CommandChainedQuestion,
				ChainQuestions: questions,
				ChainCondition: qmodel.ChainConditionIfTrue,
			}
		}
	}

	// 일반 체인 질문: /스자 질문1, 질문2, ...
	if body := parser.ExtractFirstGroup(p.chainRegularRe, text); body != "" {
		questions := parser.SplitByComma(body)
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

func (p *CommandParser) parseAsk(text string) *Command {
	for _, re := range p.askRes {
		q := parser.ExtractFirstGroup(re, text)
		if q != "" {
			return &Command{Kind: CommandAsk, Question: q}
		}
	}
	return nil
}

// parseAdmin: 관리자 전용 명령어를 파싱한다.
func (p *CommandParser) parseAdmin(text string) *Command {
	if parser.MatchSimple(p.adminForceEndRe, text) {
		return &Command{Kind: CommandAdminForceEnd}
	}
	if parser.MatchSimple(p.adminClearAllRe, text) {
		return &Command{Kind: CommandAdminClearAll}
	}
	return nil
}

// parseUserStats: 개인 전적 또는 채팅방 전체 전적 조회 명령을 파싱한다.
func (p *CommandParser) parseUserStats(text string) *Command {
	// 방 전적 조회 먼저 (더 구체적인 패턴)
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

	// 개인 전적 조회
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

// parseUsage: 토큰 사용량 조회 명령을 파싱한다.
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
