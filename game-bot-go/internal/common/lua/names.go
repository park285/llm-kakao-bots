package lua

// 공통 스크립트 이름 상수.
const (
	ScriptPendingEnqueue = "pending_enqueue"
	ScriptPendingDequeue = "pending_dequeue"
)

// twentyq 스크립트 이름 상수.
const (
	ScriptGuessRateLimit   = "guess_rate_limit"
	ScriptLockAcquireRead  = "lock_acquire_read"
	ScriptLockAcquireWrite = "lock_acquire_write"
	ScriptLockRelease      = "lock_release"
	ScriptLockReleaseRead  = "lock_release_read"
	ScriptLockRenewRead    = "lock_renew_read"
	ScriptLockRenewWrite   = "lock_renew_write"
)

// turtlesoup 스크립트 이름 상수.
const (
	ScriptTurtleLockRelease = "turtlesoup_lock_release"
)
