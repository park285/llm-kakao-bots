-- ===================================
-- Create game_sessions table (세션 단위 집계용)
-- ===================================
CREATE TABLE IF NOT EXISTS game_sessions (
    id BIGSERIAL PRIMARY KEY,
    session_id VARCHAR(255) NOT NULL,
    chat_id VARCHAR(255) NOT NULL,
    category VARCHAR(100) NOT NULL,
    result VARCHAR(20) NOT NULL,
    participant_count INT NOT NULL DEFAULT 0,
    question_count INT NOT NULL DEFAULT 0,
    hint_count INT NOT NULL DEFAULT 0,
    completed_at TIMESTAMP NOT NULL DEFAULT NOW(),
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_game_sessions_session_id
    ON game_sessions(session_id);

CREATE INDEX IF NOT EXISTS idx_game_sessions_chat_completed
    ON game_sessions(chat_id, completed_at);

COMMENT ON TABLE game_sessions IS '게임 세션 단위 로그 (판당 1건)';
COMMENT ON COLUMN game_sessions.session_id IS 'LLM 세션 ID(판 식별자)';
COMMENT ON COLUMN game_sessions.chat_id IS '채팅방 ID';
COMMENT ON COLUMN game_sessions.category IS '게임 카테고리';
COMMENT ON COLUMN game_sessions.result IS '게임 결과 (CORRECT, SURRENDER)';
COMMENT ON COLUMN game_sessions.participant_count IS '참여자 수';
COMMENT ON COLUMN game_sessions.question_count IS '총 질문 수';
COMMENT ON COLUMN game_sessions.hint_count IS '총 힌트 수';
COMMENT ON COLUMN game_sessions.completed_at IS '게임 완료 시각';
COMMENT ON COLUMN game_sessions.created_at IS '로그 생성 시각';
