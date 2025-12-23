package usage

import "time"

// TokenUsage 는 일자별 토큰 사용량 집계를 저장하는 DB 모델이다.
type TokenUsage struct {
	ID              int64     `gorm:"column:id;primaryKey"`
	UsageDate       time.Time `gorm:"column:usage_date;type:date"`
	InputTokens     int64     `gorm:"column:input_tokens"`
	OutputTokens    int64     `gorm:"column:output_tokens"`
	ReasoningTokens int64     `gorm:"column:reasoning_tokens"`
	RequestCount    int64     `gorm:"column:request_count"`
	Version         int64     `gorm:"column:version"`
}

// TableName 은 GORM에서 사용할 테이블명을 반환한다.
func (TokenUsage) TableName() string {
	return "token_usage"
}

// DailyUsage 는 API/집계용 일자별 사용량 뷰 모델이다.
type DailyUsage struct {
	UsageDate       time.Time
	InputTokens     int64
	OutputTokens    int64
	ReasoningTokens int64
	RequestCount    int64
}

// TotalTokens 는 입력+출력 토큰 합계를 반환한다.
func (d DailyUsage) TotalTokens() int64 {
	return d.InputTokens + d.OutputTokens
}
