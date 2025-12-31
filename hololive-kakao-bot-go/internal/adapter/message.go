package adapter

import (
	"strconv"
	"strings"

	"github.com/kapu/hololive-kakao-bot-go/internal/domain"
	"github.com/kapu/hololive-kakao-bot-go/internal/iris"
	"github.com/kapu/hololive-kakao-bot-go/internal/util"
)

// MessageAdapter 는 타입이다.
type MessageAdapter struct {
	prefix string
}

// NewMessageAdapter 는 동작을 수행한다.
func NewMessageAdapter(prefix string) *MessageAdapter {
	return &MessageAdapter{prefix: prefix}
}

// ParsedCommand 는 타입이다.
type ParsedCommand struct {
	Type       domain.CommandType
	Params     map[string]any
	RawMessage string
}

// ParseMessage 는 동작을 수행한다.
func (ma *MessageAdapter) ParseMessage(message *iris.Message) *ParsedCommand {
	if message == nil || message.Msg == "" {
		return ma.createUnknownCommand("")
	}

	text := util.TrimSpace(message.Msg)
	if !strings.HasPrefix(text, ma.prefix) {
		return ma.createUnknownCommand(text)
	}

	commandText := util.TrimSpace(text[len(ma.prefix):])
	parts := strings.Fields(commandText)
	if len(parts) == 0 {
		return ma.createUnknownCommand(text)
	}

	command := util.Normalize(parts[0])
	args := parts[1:]

	if normalizedCmd, normalizedArgs, ok := normalizeCompactAlarmTokens(command, args); ok {
		command = normalizedCmd
		args = normalizedArgs
	}

	if parsed, ok := ma.tryLiveCommand(command, args, text); ok {
		return parsed
	}
	if parsed, ok := ma.tryUpcomingCommand(command, args, text); ok {
		return parsed
	}
	if parsed, ok := ma.tryScheduleCommand(command, args, text); ok {
		return parsed
	}
	if parsed, ok := ma.tryAlarmCommand(command, args, text); ok {
		return parsed
	}
	if parsed, ok := ma.tryHelpCommand(command, text); ok {
		return parsed
	}
	if parsed, ok := ma.trySubscriberCommand(command, args, text); ok {
		return parsed
	}
	if parsed, ok := ma.tryStatsCommand(command, args, text); ok {
		return parsed
	}
	if parsed, ok := ma.tryMemberInfoCommand(command, args, text); ok {
		return parsed
	}

	return ma.createUnknownCommand(text)
}

func (ma *MessageAdapter) tryLiveCommand(command string, args []string, raw string) (*ParsedCommand, bool) {
	if !ma.isLiveCommand(command) {
		return nil, false
	}

	params := make(map[string]any)
	if len(args) > 0 {
		member := util.TrimSpace(strings.Join(args, " "))
		if member != "" {
			params["member"] = member
		}
	}

	return &ParsedCommand{Type: domain.CommandLive, Params: params, RawMessage: raw}, true
}

func (ma *MessageAdapter) tryUpcomingCommand(command string, args []string, raw string) (*ParsedCommand, bool) {
	if !ma.isUpcomingCommand(command) {
		return nil, false
	}
	return &ParsedCommand{
		Type:       domain.CommandUpcoming,
		Params:     ma.parseUpcomingArgs(args),
		RawMessage: raw,
	}, true
}

func (ma *MessageAdapter) tryScheduleCommand(command string, args []string, raw string) (*ParsedCommand, bool) {
	if !ma.isScheduleCommand(command) {
		return nil, false
	}
	if len(args) == 0 && util.Contains([]string{"멤버", "member"}, command) {
		return nil, false
	}
	params := ma.parseScheduleArgs(args)
	params["_raw_command"] = command
	return &ParsedCommand{
		Type:       domain.CommandSchedule,
		Params:     params,
		RawMessage: raw,
	}, true
}

func (ma *MessageAdapter) tryAlarmCommand(command string, args []string, raw string) (*ParsedCommand, bool) {
	if !ma.isAlarmCommand(command, args) {
		return nil, false
	}
	return ma.parseAlarmCommand(command, args, raw), true
}

func (ma *MessageAdapter) tryHelpCommand(command string, raw string) (*ParsedCommand, bool) {
	if !ma.isHelpCommand(command) {
		return nil, false
	}
	return &ParsedCommand{Type: domain.CommandHelp, Params: make(map[string]any), RawMessage: raw}, true
}

func (ma *MessageAdapter) trySubscriberCommand(command string, args []string, raw string) (*ParsedCommand, bool) {
	if !ma.isSubscriberCommand(command) {
		return nil, false
	}
	// 멤버 이름이 없으면 에러 처리를 위해 빈 member로 전달
	member := util.TrimSpace(strings.Join(args, " "))
	return &ParsedCommand{
		Type:       domain.CommandSubscriber,
		Params:     map[string]any{"member": member},
		RawMessage: raw,
	}, true
}

func (ma *MessageAdapter) tryStatsCommand(command string, args []string, raw string) (*ParsedCommand, bool) {
	if !ma.isStatsCommand(command) {
		return nil, false
	}
	return &ParsedCommand{
		Type:       domain.CommandStats,
		Params:     ma.parseStatsArgs(args),
		RawMessage: raw,
	}, true
}

func (ma *MessageAdapter) tryMemberInfoCommand(command string, args []string, raw string) (*ParsedCommand, bool) {
	if !ma.isMemberInfoCommand(command) {
		return nil, false
	}

	query := util.TrimSpace(strings.Join(args, " "))
	params := make(map[string]any)
	if query != "" {
		params["query"] = query
	}

	return &ParsedCommand{Type: domain.CommandMemberInfo, Params: params, RawMessage: raw}, true
}

func (ma *MessageAdapter) isLiveCommand(cmd string) bool {
	return util.Contains([]string{"라이브", "live", "방송중", "생방송"}, cmd)
}

func (ma *MessageAdapter) isUpcomingCommand(cmd string) bool {
	return util.Contains([]string{"예정", "upcoming"}, cmd)
}

func (ma *MessageAdapter) isScheduleCommand(cmd string) bool {
	return util.Contains([]string{"일정", "스케줄", "schedule", "멤버", "member"}, cmd)
}

func (ma *MessageAdapter) isAlarmCommand(cmd string, args []string) bool {
	if util.Contains([]string{"알람", "알림", "알림설정", "알람설정", "alarm"}, cmd) {
		return true
	}

	if len(args) > 0 {
		subCmd := util.Normalize(args[0])
		return util.Contains([]string{"추가", "set", "add", "설정", "제거", "remove", "del", "삭제", "목록", "list", "초기화", "clear"}, subCmd)
	}

	return false
}

func (ma *MessageAdapter) isHelpCommand(cmd string) bool {
	return util.Contains([]string{"도움말", "도움", "help", "명령어", "commands"}, cmd)
}

func (ma *MessageAdapter) isMemberInfoCommand(cmd string) bool {
	return util.Contains([]string{"멤버", "member", "프로필", "profile", "정보", "info"}, cmd)
}

func (ma *MessageAdapter) isSubscriberCommand(cmd string) bool {
	return util.Contains([]string{"구독자", "subscriber", "subs"}, cmd)
}

func (ma *MessageAdapter) isStatsCommand(cmd string) bool {
	return util.Contains([]string{"구독자순위", "순위", "통계", "stats", "ranking"}, cmd)
}

func (ma *MessageAdapter) parseUpcomingArgs(args []string) map[string]any {
	params := make(map[string]any)
	if len(args) > 0 {
		member := util.TrimSpace(strings.Join(args, " "))
		if member != "" {
			params["member"] = member
		}
	}
	return params
}

func (ma *MessageAdapter) parseScheduleArgs(args []string) map[string]any {
	if len(args) == 0 {
		return make(map[string]any)
	}

	member := args[0]
	days := 7

	if len(args) > 1 {
		if d, err := strconv.Atoi(args[1]); err == nil {
			days = d
			if days < 1 {
				days = 1
			}
			if days > 30 {
				days = 30
			}
		}
	}

	return map[string]any{
		"member": member,
		"days":   days,
	}
}

func (ma *MessageAdapter) parseStatsArgs(args []string) map[string]any {
	params := map[string]any{"action": "gainers"}
	for _, arg := range args {
		token := util.TrimSpace(arg)
		if token == "" {
			continue
		}

		if strings.Contains(token, "=") {
			parts := strings.SplitN(token, "=", 2)
			if len(parts) != 2 {
				continue
			}

			key := util.TrimSpace(parts[0])
			value := util.TrimSpace(parts[1])
			if key == "" || value == "" {
				continue
			}

			lowerKey := util.Normalize(key)
			if isStatsPeriodKey(lowerKey) {
				if canonical := normalizePeriodToken(value); canonical != "" {
					params["period"] = canonical
				} else {
					params["period"] = value
				}
			} else if canonical := normalizePeriodToken(value); canonical != "" {
				params["period"] = canonical
			}
			continue
		}

		if canonical := normalizePeriodToken(token); canonical != "" {
			params["period"] = canonical
		}
	}

	return params
}

func (ma *MessageAdapter) parseAlarmCommand(_ string, args []string, rawMessage string) *ParsedCommand {
	if len(args) == 0 {
		return &ParsedCommand{
			Type:       domain.CommandAlarmList,
			Params:     map[string]any{"action": "list"},
			RawMessage: rawMessage,
		}
	}

	subCmd := util.Normalize(args[0])
	restArgs := args[1:]

	if util.Contains([]string{"추가", "설정", "set", "add"}, subCmd) {
		return &ParsedCommand{
			Type: domain.CommandAlarmAdd,
			Params: map[string]any{
				"action": "add",
				"member": strings.Join(restArgs, " "),
			},
			RawMessage: rawMessage,
		}
	}

	if util.Contains([]string{"제거", "삭제", "remove", "del", "delete"}, subCmd) {
		return &ParsedCommand{
			Type: domain.CommandAlarmRemove,
			Params: map[string]any{
				"action": "remove",
				"member": strings.Join(restArgs, " "),
			},
			RawMessage: rawMessage,
		}
	}

	if util.Contains([]string{"목록", "list", "show"}, subCmd) {
		return &ParsedCommand{
			Type:       domain.CommandAlarmList,
			Params:     map[string]any{"action": "list"},
			RawMessage: rawMessage,
		}
	}

	if util.Contains([]string{"초기화", "clear", "reset"}, subCmd) {
		return &ParsedCommand{
			Type:       domain.CommandAlarmClear,
			Params:     map[string]any{"action": "clear"},
			RawMessage: rawMessage,
		}
	}

	return &ParsedCommand{
		Type: domain.CommandAlarmInvalid,
		Params: map[string]any{
			"action":      "invalid",
			"sub_command": subCmd,
			"member":      strings.Join(restArgs, " "),
		},
		RawMessage: rawMessage,
	}
}

func (ma *MessageAdapter) createUnknownCommand(text string) *ParsedCommand {
	return &ParsedCommand{
		Type:       domain.CommandUnknown,
		Params:     make(map[string]any),
		RawMessage: text,
	}
}

func isStatsPeriodKey(key string) bool {
	switch key {
	case "period", "기간", "주기", "순위", "랭킹", "구독자", "통계":
		return true
	}
	return false
}

func normalizePeriodToken(raw string) string {
	return domain.NormalizeStatsPeriodToken(raw)
}

// 알람 명령 정규화
func normalizeCompactAlarmTokens(command string, args []string) (string, []string, bool) {
	mapping := map[string]string{
		"알람설정":  "설정",
		"알림설정":  "설정",
		"알람추가":  "추가",
		"알림추가":  "추가",
		"알람목록":  "목록",
		"알림목록":  "목록",
		"알람리스트": "목록",
		"알림리스트": "목록",
		"알람제거":  "제거",
		"알림제거":  "제거",
		"알람삭제":  "삭제",
		"알림삭제":  "삭제",
		"알람초기화": "초기화",
		"알림초기화": "초기화",
		"알람리셋":  "초기화",
		"알림리셋":  "초기화",
		"알람해제":  "제거",
		"알림해제":  "제거",
	}

	subCmd, ok := mapping[command]
	if !ok {
		return command, args, false
	}

	newArgs := make([]string, 0, 1+len(args))
	newArgs = append(newArgs, subCmd)
	newArgs = append(newArgs, args...)

	return "알람", newArgs, true
}
