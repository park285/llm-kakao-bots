package messages

// 메시지 키 상수.
const (
	// StartWaiting 는 상수다.
	StartWaiting           = "start.waiting"
	StartScenario          = "start.scenario"
	StartInstruction       = "start.instruction"
	StartResume            = "start.resume"
	StartResumeStatus      = "start.resume_status"
	StartInvalidDifficulty = "start.invalid_difficulty"

	AnswerResponseSingle      = "answer.response_single"
	AnswerResponseWithHistory = "answer.response_with_history"
	AnswerHistoryHeader       = "answer.history_header"
	AnswerHistoryItem         = "answer.history_item"
	AnswerCorrect             = "answer.correct"
	AnswerIncorrect           = "answer.incorrect"
	AnswerCloseCall           = "answer.close_call"

	HintWaiting      = "hint.waiting"
	HintGenerated    = "hint.generated"
	HintCooldown     = "hint.cooldown"
	HintLimitReached = "hint.limit_reached"
	HintEarlyStage   = "hint.early_stage"
	HintSectionUsed  = "hint.section_used"
	HintSectionNone  = "hint.section_none"
	HintItem         = "hint.hint_item"

	ProblemDisplay = "problem.display"

	SurrenderResult          = "surrender.result"
	SurrenderHintBlockHeader = "surrender.hint_block_header"
	SurrenderHintItem        = "surrender.hint_item"

	VoteStart         = "vote.start"
	VoteInProgress    = "vote.in_progress"
	VoteAlreadyActive = "vote.already_active"
	VoteNotFound      = "vote.not_found"
	VoteAlreadyVoted  = "vote.already_voted"
	VotePassed        = "vote.passed"

	SummaryHeader = "summary.header"
	SummaryItem   = "summary.item"
	SummaryEmpty  = "summary.empty"

	LockRequestInProgress           = "lock.request_in_progress"
	LockRequestInProgressWithHolder = "lock.request_in_progress_with_holder"
	LockMessageQueued               = "lock.message_queued"
	LockAlreadyQueued               = "lock.already_queued"
	LockQueueFull                   = "lock.queue_full"

	QueueProcessing     = "queue.processing"
	QueueMessageQueued  = "queue.message_queued"
	QueueAlreadyQueued  = "queue.already_queued"
	QueueFull           = "queue.full"
	QueueEmpty          = "queue.empty"
	QueueRetry          = "queue.retry"
	QueueRetryDuplicate = "queue.retry_duplicate"
	QueueRetryFailed    = "queue.retry_failed"

	ProcessingThinking       = "processing.thinking"
	ProcessingWaiting        = "processing.waiting"
	ProcessingGeneratingHint = "processing.generating_hint"
	ProcessingValidating     = "processing.validating"

	HelpMessage = "help.message"

	ErrorNoSession          = "error.no_session"
	ErrorInvalidQuestion    = "error.invalid_question"
	ErrorMaxHints           = "error.max_hints"
	ErrorInvalidAnswer      = "error.invalid_answer"
	ErrorGameAlreadyStarted = "error.game_already_started"
	ErrorGameAlreadySolved  = "error.game_already_solved"
	ErrorPuzzleGeneration   = "error.puzzle_generation"
	ErrorLockFailed         = "error.lock_failed"
	ErrorInternal           = "error.internal"
	ErrorUnknownCommand     = "error.unknown_command"
	ErrorAccessDenied       = "error.access_denied"
	ErrorUserBlocked        = "error.user_blocked"
	ErrorChatBlocked        = "error.chat_blocked"

	ErrorAICallTimeout = "error.ai_timeout"
	ErrorAIUnavailable = "error.ai_unavailable"

	FallbackPuzzleNotFound = "fallback.puzzle_not_found"

	UserAnonymous = "user.anonymous"
)
