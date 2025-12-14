-- ===================================
-- Token Usage 테이블 생성
-- MCP LLM 서버 토큰 사용량 추적용
-- ===================================
CREATE TABLE IF NOT EXISTS token_usage (
    id BIGSERIAL PRIMARY KEY,
    usage_date DATE NOT NULL UNIQUE,
    input_tokens BIGINT NOT NULL DEFAULT 0,
    output_tokens BIGINT NOT NULL DEFAULT 0,
    reasoning_tokens BIGINT NOT NULL DEFAULT 0,
    request_count BIGINT NOT NULL DEFAULT 0,
    version BIGINT NOT NULL DEFAULT 0
);

CREATE INDEX IF NOT EXISTS idx_token_usage_date ON token_usage(usage_date);

COMMENT ON TABLE token_usage IS '일별 토큰 사용량 추적';
COMMENT ON COLUMN token_usage.usage_date IS '사용 날짜 (일별 집계)';
COMMENT ON COLUMN token_usage.input_tokens IS '입력 토큰 수';
COMMENT ON COLUMN token_usage.output_tokens IS '출력 토큰 수';
COMMENT ON COLUMN token_usage.reasoning_tokens IS '추론(thinking) 토큰 수';
COMMENT ON COLUMN token_usage.request_count IS '요청 횟수';
COMMENT ON COLUMN token_usage.version IS 'Optimistic locking용 버전';
