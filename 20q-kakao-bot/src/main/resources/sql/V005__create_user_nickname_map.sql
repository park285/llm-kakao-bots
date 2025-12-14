-- ===================================
-- Create user_nickname_map table (닉네임→userId 매핑)
-- ===================================
CREATE TABLE IF NOT EXISTS user_nickname_map (
    id BIGSERIAL PRIMARY KEY,
    chat_id VARCHAR(255) NOT NULL,
    user_id VARCHAR(255) NOT NULL,
    last_sender VARCHAR(255) NOT NULL,
    last_seen_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_user_nickname_unique
    ON user_nickname_map(chat_id, user_id);

CREATE INDEX IF NOT EXISTS idx_user_nickname_chat_sender
    ON user_nickname_map(chat_id, lower(last_sender));

COMMENT ON TABLE user_nickname_map IS '닉네임↔사용자 매핑 (방별 마지막 닉네임 보존)';
COMMENT ON COLUMN user_nickname_map.chat_id IS '채팅방 ID';
COMMENT ON COLUMN user_nickname_map.user_id IS '사용자 ID';
COMMENT ON COLUMN user_nickname_map.last_sender IS '가장 최근 닉네임';
COMMENT ON COLUMN user_nickname_map.last_seen_at IS '닉네임이 관측된 시각';
COMMENT ON COLUMN user_nickname_map.created_at IS '행 생성 시각';
