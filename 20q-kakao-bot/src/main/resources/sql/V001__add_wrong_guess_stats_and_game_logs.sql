-- ===================================
-- 1. Add wrong guess columns to user_stats
-- ===================================
ALTER TABLE user_stats
    ADD COLUMN IF NOT EXISTS total_wrong_guesses INT NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS best_score_wrong_guess_count INT NULL;

COMMENT ON COLUMN user_stats.total_wrong_guesses IS '총 틀린 정답 시도 횟수';
COMMENT ON COLUMN user_stats.best_score_wrong_guess_count IS '베스트 스코어 달성 시 틀린 정답 시도 횟수';

-- ===================================
-- 2. Create game_logs table for period-based statistics
-- ===================================
CREATE TABLE IF NOT EXISTS game_logs (
    id BIGSERIAL PRIMARY KEY,
    chat_id VARCHAR(255) NOT NULL,
    user_id VARCHAR(255) NOT NULL,
    sender VARCHAR(255) NOT NULL DEFAULT '',
    category VARCHAR(100) NOT NULL,
    question_count INT NOT NULL DEFAULT 0,
    hint_count INT NOT NULL DEFAULT 0,
    wrong_guess_count INT NOT NULL DEFAULT 0,
    result VARCHAR(20) NOT NULL,
    target VARCHAR(255),
    completed_at TIMESTAMP NOT NULL DEFAULT NOW(),
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE game_logs IS '게임 완료 로그 (기간별 통계용)';
COMMENT ON COLUMN game_logs.chat_id IS '채팅방 ID';
COMMENT ON COLUMN game_logs.user_id IS '사용자 ID';
COMMENT ON COLUMN game_logs.sender IS '사용자 닉네임 (표시명)';
COMMENT ON COLUMN game_logs.category IS '게임 카테고리';
COMMENT ON COLUMN game_logs.question_count IS '질문 횟수';
COMMENT ON COLUMN game_logs.hint_count IS '힌트 사용 횟수';
COMMENT ON COLUMN game_logs.wrong_guess_count IS '틀린 정답 시도 횟수';
COMMENT ON COLUMN game_logs.result IS '게임 결과 (CORRECT, SURRENDER)';
COMMENT ON COLUMN game_logs.target IS '정답 (정답 맞춤 시)';
COMMENT ON COLUMN game_logs.completed_at IS '게임 완료 시간';
COMMENT ON COLUMN game_logs.created_at IS '로그 생성 시간';

-- Create indexes for efficient queries
CREATE INDEX IF NOT EXISTS idx_game_logs_chat_completed 
    ON game_logs(chat_id, completed_at);
    
CREATE INDEX IF NOT EXISTS idx_game_logs_user_completed 
    ON game_logs(chat_id, user_id, completed_at);

COMMENT ON INDEX idx_game_logs_chat_completed IS '방별 기간별 조회용 인덱스';
COMMENT ON INDEX idx_game_logs_user_completed IS '사용자별 기간별 조회용 인덱스';
