package domain

import "time"

// TimestampedStats 는 타입이다.
type TimestampedStats struct {
	ChannelID       string    `json:"channel_id"`
	MemberName      string    `json:"member_name"`
	SubscriberCount uint64    `json:"subscriber_count"`
	VideoCount      uint64    `json:"video_count"`
	ViewCount       uint64    `json:"view_count"`
	Timestamp       time.Time `json:"timestamp"`
}

// MilestoneType 는 타입이다.
type MilestoneType string

// MilestoneType 상수 목록.
const (
	// MilestoneSubscribers 는 상수다.
	MilestoneSubscribers MilestoneType = "subscribers"
	MilestoneVideos      MilestoneType = "videos"
	MilestoneViews       MilestoneType = "views"
)

// Milestone 는 타입이다.
type Milestone struct {
	ChannelID  string        `json:"channel_id"`
	MemberName string        `json:"member_name"`
	Type       MilestoneType `json:"type"`
	Value      uint64        `json:"value"` // e.g., 1000000 for 1M subscribers
	AchievedAt time.Time     `json:"achieved_at"`
	Notified   bool          `json:"notified"`
}

// StatsChange 는 타입이다.
type StatsChange struct {
	ChannelID        string            `json:"channel_id"`
	MemberName       string            `json:"member_name"`
	SubscriberChange int64             `json:"subscriber_change"`
	VideoChange      int64             `json:"video_change"`
	ViewChange       int64             `json:"view_change"`
	PreviousStats    *TimestampedStats `json:"previous_stats"`
	CurrentStats     *TimestampedStats `json:"current_stats"`
	DetectedAt       time.Time         `json:"detected_at"`
}

// DailySummary 는 타입이다.
type DailySummary struct {
	Date               time.Time   `json:"date"`
	TotalChanges       int         `json:"total_changes"`
	MilestonesAchieved int         `json:"milestones_achieved"`
	NewVideosDetected  int         `json:"new_videos_detected"`
	TopGainers         []RankEntry `json:"top_gainers"`
	TopUploaders       []RankEntry `json:"top_uploaders"`
}

// RankEntry 는 타입이다.
type RankEntry struct {
	ChannelID          string `json:"channel_id"`
	MemberName         string `json:"member_name"`
	Value              int64  `json:"value"`               // subscriber change or video count
	CurrentSubscribers uint64 `json:"current_subscribers"` // latest subscriber count (optional)
	Rank               int    `json:"rank"`
}

// TrendData 는 타입이다.
type TrendData struct {
	ChannelID        string    `json:"channel_id"`
	MemberName       string    `json:"member_name"`
	Period           string    `json:"period"` // "daily", "weekly", "monthly"
	SubscriberGrowth int64     `json:"subscriber_growth"`
	VideoUploadRate  float64   `json:"video_upload_rate"` // videos per day
	AvgViewsPerVideo uint64    `json:"avg_views_per_video"`
	UpdatedAt        time.Time `json:"updated_at"`
}
