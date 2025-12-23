package model

// StatsPeriod 전적 조회 기간.
type StatsPeriod string

// StatsPeriodDaily 는 전적 조회 기간 상수 목록이다.
const (
	StatsPeriodDaily   StatsPeriod = "daily"
	StatsPeriodWeekly  StatsPeriod = "weekly"
	StatsPeriodMonthly StatsPeriod = "monthly"
	StatsPeriodAll     StatsPeriod = "all"
)

// UsagePeriod 사용량(토큰) 조회 기간.
type UsagePeriod string

// UsagePeriodToday 는 사용량 조회 기간 상수 목록이다.
const (
	UsagePeriodToday   UsagePeriod = "today"
	UsagePeriodWeekly  UsagePeriod = "weekly"
	UsagePeriodMonthly UsagePeriod = "monthly"
)
