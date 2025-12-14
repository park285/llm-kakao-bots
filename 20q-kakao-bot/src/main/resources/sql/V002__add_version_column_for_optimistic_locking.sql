-- ===================================
-- Optimistic Locking을 위한 version 컬럼 추가
-- Race condition 방지: 동시 업데이트 시 OptimisticLockingFailureException 발생
-- ===================================
ALTER TABLE user_stats
    ADD COLUMN IF NOT EXISTS version BIGINT NOT NULL DEFAULT 0;

COMMENT ON COLUMN user_stats.version IS 'Optimistic locking용 버전 (동시 수정 감지)';
