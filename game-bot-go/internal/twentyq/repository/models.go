package repository

import "time"

// GameSession: 게임 세션 기록
// 복합 인덱스: idx_game_sessions_room_stats (chat_id, completed_at, result)
type GameSession struct {
	ID               uint64    `gorm:"column:id;primaryKey;autoIncrement"`
	SessionID        string    `gorm:"column:session_id;not null;uniqueIndex"`
	ChatID           string    `gorm:"column:chat_id;not null;index:idx_game_sessions_room_stats,priority:1"`
	Category         string    `gorm:"column:category;not null;index"`
	Target           string    `gorm:"column:target;not null;default:''"`
	Result           string    `gorm:"column:result;not null;index:idx_game_sessions_room_stats,priority:3"`
	ParticipantCount int       `gorm:"column:participant_count;not null"`
	QuestionCount    int       `gorm:"column:question_count;not null;default:0"`
	HintCount        int       `gorm:"column:hint_count;not null;default:0"`
	CompletedAt      time.Time `gorm:"column:completed_at;not null;index:idx_game_sessions_room_stats,priority:2"`
	CreatedAt        time.Time `gorm:"column:created_at;not null;autoCreateTime"`
}

func (GameSession) TableName() string { return "game_sessions" }

// GameLog: 게임 로그 (참여자별 기록)
// 복합 인덱스: idx_game_logs_activity (chat_id, completed_at, sender)
type GameLog struct {
	ID              uint64    `gorm:"column:id;primaryKey;autoIncrement"`
	ChatID          string    `gorm:"column:chat_id;not null;index:idx_game_logs_activity,priority:1"`
	UserID          string    `gorm:"column:user_id;not null;index"`
	Sender          string    `gorm:"column:sender;not null;default:'';index:idx_game_logs_activity,priority:3"`
	Category        string    `gorm:"column:category;not null;index"`
	QuestionCount   int       `gorm:"column:question_count;not null;default:0"`
	HintCount       int       `gorm:"column:hint_count;not null;default:0"`
	WrongGuessCount int       `gorm:"column:wrong_guess_count;not null;default:0"`
	Result          string    `gorm:"column:result;not null;index"`
	Target          *string   `gorm:"column:target"`
	CompletedAt     time.Time `gorm:"column:completed_at;not null;index:idx_game_logs_activity,priority:2"`
	CreatedAt       time.Time `gorm:"column:created_at;not null;autoCreateTime"`
}

func (GameLog) TableName() string { return "game_logs" }

// UserStats: 사용자 통계 집계
type UserStats struct {
	ID                   string     `gorm:"column:id;primaryKey"`
	ChatID               string     `gorm:"column:chat_id;not null;index"`
	UserID               string     `gorm:"column:user_id;not null;index"`
	TotalGamesStarted    int        `gorm:"column:total_games_started;not null;default:0"`
	TotalGamesCompleted  int        `gorm:"column:total_games_completed;not null;default:0"`
	TotalSurrenders      int        `gorm:"column:total_surrenders;not null;default:0"`
	TotalQuestionsAsked  int        `gorm:"column:total_questions_asked;not null;default:0"`
	TotalHintsUsed       int        `gorm:"column:total_hints_used;not null;default:0"`
	TotalWrongGuesses    int        `gorm:"column:total_wrong_guesses;not null;default:0"`
	BestScoreQuestionCnt *int       `gorm:"column:best_score_question_count"`
	BestScoreWrongGuess  *int       `gorm:"column:best_score_wrong_guess_count"`
	BestScoreTarget      *string    `gorm:"column:best_score_target"`
	BestScoreCategory    *string    `gorm:"column:best_score_category"`
	BestScoreAchievedAt  *time.Time `gorm:"column:best_score_achieved_at"`
	CategoryStatsJSON    *string    `gorm:"column:category_stats_json;type:jsonb"`
	CreatedAt            time.Time  `gorm:"column:created_at;not null;autoCreateTime"`
	UpdatedAt            time.Time  `gorm:"column:updated_at;not null;autoUpdateTime"`
	Version              int64      `gorm:"column:version;not null;default:0"`
}

func (UserStats) TableName() string { return "user_stats" }

// UserNicknameMap: 사용자 닉네임 매핑
type UserNicknameMap struct {
	ID         uint64    `gorm:"column:id;primaryKey;autoIncrement"`
	ChatID     string    `gorm:"column:chat_id;not null;uniqueIndex:idx_user_nickname_map_chat_user"`
	UserID     string    `gorm:"column:user_id;not null;uniqueIndex:idx_user_nickname_map_chat_user"`
	LastSender string    `gorm:"column:last_sender;not null"`
	LastSeenAt time.Time `gorm:"column:last_seen_at;not null"`
	CreatedAt  time.Time `gorm:"column:created_at;not null;autoCreateTime"`
}

func (UserNicknameMap) TableName() string { return "user_nickname_map" }

// AuditLog: 판정 리뷰 로그 (AI 오판 기록)
type AuditLog struct {
	ID            uint64    `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	SessionID     string    `gorm:"column:session_id;not null;index" json:"sessionId"`
	QuestionIndex int       `gorm:"column:question_index;not null" json:"questionIndex"`
	Verdict       string    `gorm:"column:verdict;not null" json:"verdict"`
	Reason        string    `gorm:"column:reason" json:"reason"`
	AdminUserID   string    `gorm:"column:admin_user_id;not null" json:"adminUserId"`
	CreatedAt     time.Time `gorm:"column:created_at;not null;autoCreateTime" json:"createdAt"`
}

func (AuditLog) TableName() string { return "audit_logs" }

// RefundLog: 스탯 복원 로그
type RefundLog struct {
	ID          uint64    `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	SessionID   string    `gorm:"column:session_id;not null;index" json:"sessionId"`
	UserID      string    `gorm:"column:user_id;not null;index" json:"userId"`
	AdminUserID string    `gorm:"column:admin_user_id;not null" json:"adminUserId"`
	Reason      string    `gorm:"column:reason" json:"reason"`
	CreatedAt   time.Time `gorm:"column:created_at;not null;autoCreateTime" json:"createdAt"`
}

func (RefundLog) TableName() string { return "refund_logs" }
