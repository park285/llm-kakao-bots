-- 사용자 스탯 테이블 (방별 독립)
-- 게임 완료 시 비동기로 수집되는 개인화 통계
CREATE TABLE IF NOT EXISTS user_stats (
    id VARCHAR(511) PRIMARY KEY,
    chat_id VARCHAR(255) NOT NULL,
    user_id VARCHAR(255) NOT NULL,
    
    -- 게임 통계
    total_games_started INT NOT NULL DEFAULT 0,
    total_games_completed INT NOT NULL DEFAULT 0,
    total_surrenders INT NOT NULL DEFAULT 0,
    total_questions_asked INT NOT NULL DEFAULT 0,
    total_hints_used INT NOT NULL DEFAULT 0,
    
    -- 최고 기록 (가장 적은 질문으로 정답)
    best_score_question_count INT,
    best_score_target VARCHAR(255),
    best_score_category VARCHAR(50),
    best_score_achieved_at TIMESTAMP WITH TIME ZONE,
    
    -- 카테고리별 통계 (JSON)
    category_stats_json TEXT,
    
    -- 메타데이터
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    
    -- Optimistic Locking
    version BIGINT NOT NULL DEFAULT 0
);

-- 방별 사용자 조회 최적화
CREATE INDEX IF NOT EXISTS idx_user_stats_chat_user
    ON user_stats(chat_id, user_id);

-- 최고 기록 기준 랭킹 조회 최적화
CREATE INDEX IF NOT EXISTS idx_user_stats_best_score 
    ON user_stats(best_score_question_count ASC) 
    WHERE best_score_question_count IS NOT NULL;

-- 닉네임 매핑 (방별 최신 닉네임 보존)
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

-- 게임 세션 로그 (판당 1건)
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
