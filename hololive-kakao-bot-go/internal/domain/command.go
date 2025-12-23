package domain

// CommandType 는 타입이다.
type CommandType string

// CommandType 상수 목록.
const (
	// CommandLive 는 상수다.
	CommandLive         CommandType = "live"
	CommandUpcoming     CommandType = "upcoming"
	CommandSchedule     CommandType = "schedule"
	CommandHelp         CommandType = "help"
	CommandAlarmAdd     CommandType = "alarm_add"
	CommandAlarmRemove  CommandType = "alarm_remove"
	CommandAlarmList    CommandType = "alarm_list"
	CommandAlarmClear   CommandType = "alarm_clear"
	CommandAlarmInvalid CommandType = "alarm_invalid"
	CommandMemberInfo   CommandType = "member_info"
	CommandStats        CommandType = "stats"
	CommandUnknown      CommandType = "unknown"
)

func (c CommandType) String() string {
	return string(c)
}

// IsValid 는 동작을 수행한다.
func (c CommandType) IsValid() bool {
	switch c {
	case CommandLive, CommandUpcoming, CommandSchedule, CommandHelp,
		CommandAlarmAdd, CommandAlarmRemove, CommandAlarmList, CommandAlarmClear, CommandAlarmInvalid,
		CommandMemberInfo, CommandStats, CommandUnknown:
		return true
	default:
		return false
	}
}

// ParseResult 는 타입이다.
type ParseResult struct {
	Command    CommandType    `json:"command"`
	Params     map[string]any `json:"params"`
	Confidence float64        `json:"confidence"`
	Reasoning  string         `json:"reasoning"`
}

// ParseResults 는 타입이다.
type ParseResults struct {
	Single   *ParseResult
	Multiple []*ParseResult
}

// ChannelSelection 는 타입이다.
type ChannelSelection struct {
	SelectedIndex int     `json:"selectedIndex"`
	Confidence    float64 `json:"confidence"`
	Reasoning     string  `json:"reasoning"`
}

// IsSingle 는 동작을 수행한다.
func (pr *ParseResults) IsSingle() bool {
	return pr.Single != nil
}

// IsMultiple 는 동작을 수행한다.
func (pr *ParseResults) IsMultiple() bool {
	return len(pr.Multiple) > 0
}

// GetCommands 는 동작을 수행한다.
func (pr *ParseResults) GetCommands() []*ParseResult {
	if pr.IsSingle() {
		return []*ParseResult{pr.Single}
	}
	return pr.Multiple
}
