package lua

// 공통 스크립트 이름 상수.
const (
	ScriptPendingEnqueue      = "pending_enqueue"
	ScriptPendingDequeue      = "pending_dequeue"
	ScriptPendingDequeueBatch = "pending_dequeue_batch"
)

// twentyq 스크립트 이름 상수.
const (
	ScriptGuessRateLimit = "guess_rate_limit"
	ScriptLockAcquire    = "lock_acquire"
	ScriptLockRelease    = "lock_release"
	ScriptLockRenewWrite = "lock_renew_write"
)

// turtlesoup 스크립트 이름 상수.
const (
	ScriptTurtleLockRelease = "turtlesoup_lock_release"
)
