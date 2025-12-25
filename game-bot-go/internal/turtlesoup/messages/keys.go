package messages

// 메시지 키 상수.
const (
	// StartWaiting: 게임 시작 및 시나리오 안내 관련 메시지 키
	StartWaiting           = "start.waiting"
	StartScenario          = "start.scenario"
	StartInstruction       = "start.instruction"
	StartResume            = "start.resume"
	StartResumeStatus      = "start.resume_status"
	StartInvalidDifficulty = "start.invalid_difficulty"

	// AnswerResponseSingle: 질문에 대한 답변 및 정답/오답 판정 관련 메시지 키
	AnswerResponseSingle      = "answer.response_single"
	AnswerResponseWithHistory = "answer.response_with_history"
	AnswerHistoryHeader       = "answer.history_header"
	AnswerHistoryItem         = "answer.history_item"
	AnswerCorrect             = "answer.correct"
	AnswerIncorrect           = "answer.incorrect"
	AnswerCloseCall           = "answer.close_call"

	// HintWaiting: 힌트 요청 대기 및 제공 관련 메시지 키
	HintWaiting      = "hint.waiting"
	HintGenerated    = "hint.generated"
	HintCooldown     = "hint.cooldown"
	HintLimitReached = "hint.limit_reached"
	HintEarlyStage   = "hint.early_stage"
	HintSectionUsed  = "hint.section_used"
	HintSectionNone  = "hint.section_none"
	HintItem         = "hint.hint_item"

	// ProblemDisplay: 현재 문제(시나리오) 다시 보여주기 관련 메시지 키
	ProblemDisplay = "problem.display"

	// SurrenderResult: 게임 포기(항복) 결과 안내 관련 메시지 키
	SurrenderResult          = "surrender.result"
	SurrenderHintBlockHeader = "surrender.hint_block_header"
	SurrenderHintItem        = "surrender.hint_item"

	// VoteStart: 항복 투표 진행 관련 메시지 키
	VoteStart         = "vote.start"
	VoteInProgress    = "vote.in_progress"
	VoteAlreadyActive = "vote.already_active"
	VoteNotFound      = "vote.not_found"
	VoteAlreadyVoted  = "vote.already_voted"
	VotePassed        = "vote.passed"

	// SummaryHeader: 게임 진행 요약(질문/답변 이력) 관련 메시지 키
	SummaryHeader = "summary.header"
	SummaryItem   = "summary.item"
	SummaryEmpty  = "summary.empty"

	// LockRequestInProgress: 분산 락 요청 및 대기열 상태 알림 관련 메시지 키
	LockRequestInProgress           = "lock.request_in_progress"
	LockRequestInProgressWithHolder = "lock.request_in_progress_with_holder"
	LockMessageQueued               = "lock.message_queued"
	LockAlreadyQueued               = "lock.already_queued"
	LockQueueFull                   = "lock.queue_full"

	// QueueProcessing: 대기열 처리 및 상태 관련 메시지 키
	QueueProcessing     = "queue.processing"
	QueueMessageQueued  = "queue.message_queued"
	QueueAlreadyQueued  = "queue.already_queued"
	QueueFull           = "queue.full"
	QueueEmpty          = "queue.empty"
	QueueRetry          = "queue.retry"
	QueueRetryDuplicate = "queue.retry_duplicate"
	QueueRetryFailed    = "queue.retry_failed"

	// ProcessingThinking: AI 처리 중(생각 중, 힌트 생성 중) 상태 알림 메시지 키
	ProcessingThinking       = "processing.thinking"
	ProcessingWaiting        = "processing.waiting"
	ProcessingGeneratingHint = "processing.generating_hint"
	ProcessingValidating     = "processing.validating"

	// HelpMessage: 도움말 출력 메시지 키
	HelpMessage = "help.message"

	// ErrorNoSession: 각종 에러 상황에 대한 메시지 키
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

	// ErrorAICallTimeout: AI 서비스 호출 관련 에러 메시지 키
	ErrorAICallTimeout = "error.ai_timeout"
	ErrorAIUnavailable = "error.ai_unavailable"

	// FallbackPuzzleNotFound: 백업 퍼즐 데이터 부재 시 메시지 키
	FallbackPuzzleNotFound = "fallback.puzzle_not_found"

	// UserAnonymous: 사용자 이름이 없을 때 대체할 텍스트 키
	UserAnonymous = "user.anonymous"
)
