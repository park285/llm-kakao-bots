package messages

const (
	StartWaiting                = "start.waiting"
	StartReady                  = "start.ready"
	StartReadyWithCategory      = "start.ready_with_category"
	StartCategoryPrefix         = "start.category_prefix"
	StartInvalidCategoryWarning = "start.invalid_category_warning"

	// Resume 관련 키
	StartResumeHeader       = "start.resume_header"
	StartResumeCategoryLine = "start.resume_category_line"
	StartResumeQnAHeader    = "start.resume_qna_header"
	StartResumeHintHeader   = "start.resume_hint_header"
)

// HintWaiting: 힌트 생성 및 제공 관련 메시지 키
const (
	HintWaiting   = "hint.waiting"
	HintGenerated = "hint.generated"
)

// StatusHeaderWithCategory: 게임 상태(Status) 출력 시 사용되는 헤더 및 포맷 관련 메시지 키
const (
	StatusHeaderWithCategory = "status.header_with_category"
	StatusHeaderNoCategory   = "status.header_no_category"
	StatusHintLine           = "status.hint_line"
	StatusWrongGuesses       = "status.wrong_guesses"
	StatusQuestionAnswer     = "status.question_answer"
	StatusChainSuffix        = "status.chain_suffix"
)

// VoteStart: 항복 투표(Surrender Vote) 관련 메시지 키
const (
	VoteStart              = "vote.start"
	VoteInProgress         = "vote.in_progress"
	VoteNotFound           = "vote.not_found"
	VoteAlreadyVoted       = "vote.already_voted"
	VoteCannotVote         = "vote.cannot_vote"
	VoteAgreeProgress      = "vote.agree_progress"
	VoteProcessingFailed   = "vote.processing_failed"
	VoteRejectNotSupported = "vote.reject_not_supported"
)

// ProcessingWaiting: 일반적인 처리 대기 안내 메시지 키
const (
	ProcessingWaiting = "processing.waiting"
)

// LockRequestInProgress: 분산 락, 대기열(Queue) 상태 알림 관련 메시지 키
const (
	LockRequestInProgress = "lock.request_in_progress"
	LockMessageQueued     = "lock.message_queued"
	LockAlreadyQueued     = "lock.already_queued"
	LockQueueFull         = "lock.queue_full"

	QueueProcessing     = "queue.processing"
	QueueRetry          = "queue.retry"
	QueueRetryDuplicate = "queue.retry_duplicate"
	QueueRetryFailed    = "queue.retry_failed"
	QueueEmpty          = "queue.empty"
)

// AnswerSuccess: 정답 확인 및 오답/근접 정답 처리 관련 메시지 키
const (
	AnswerSuccess           = "answer.success"
	AnswerCorrectDefault    = "answer.correct_default"
	AnswerWrongGuess        = "answer.wrong_guess"
	AnswerCloseCall         = "answer.close_call"
	AnswerHintSectionUsed   = "answer.hint_section_used"
	AnswerHintSectionNone   = "answer.hint_section_none"
	AnswerHintItem          = "answer.hint_item"
	AnswerWrongGuessSection = "answer.wrong_guess_section"
)

// SurrenderResult: 항복 결과 안내 관련 메시지 키
const (
	SurrenderResult          = "surrender.result"
	SurrenderHintBlockHeader = "surrender.hint_block_header"
	SurrenderHintItem        = "surrender.hint_item"
	SurrenderCategoryLine    = "surrender.category_line"
)

// HelpMessage: 도움말 출력 메시지 키
const (
	HelpMessage = "help.message"
)

// ModelInfoFetchFailed: 사용 중인 AI 모델 정보 및 상태 조회 관련 메시지 키
const (
	ModelInfoFetchFailed = "model_info.fetch_failed"
	ModelInfoHeader      = "model_info.header"
	ModelInfoDefault     = "model_info.default"
	ModelInfoHints       = "model_info.hints"
	ModelInfoAnswer      = "model_info.answer"
	ModelInfoVerify      = "model_info.verify"
	ModelInfoTemperature = "model_info.temperature"
	ModelInfoMaxRetries  = "model_info.max_retries"
	ModelInfoTimeout     = "model_info.timeout"
	ModelInfoTransport   = "model_info.transport"
)

// HealthAlive: 헬스체크 응답 메시지 키
const (
	HealthAlive = "health.alive"
)

// UserAnonymous: 사용자 이름이 없을 때 대체할 텍스트 키
const (
	UserAnonymous = "user.anonymous"
)

// ErrorNoSession: 각종 에러 상황에 대한 메시지 키
const (
	ErrorNoSession         = "error.no_session"
	ErrorNoSessionShort    = "error.no_session_short"
	ErrorUnknownCommand    = "error.unknown_command"
	ErrorGeneric           = "error.generic_error"
	ErrorInvalidQuestion   = "error.invalid_question.default"
	ErrorDuplicateQuestion = "error.invalid_question.duplicate_question"
	ErrorHintNotAvailable  = "error.hint_not_available"
	ErrorHintLimitExceeded = "error.hint_limit_exceeded"
	ErrorAITimeout         = "error.ai_timeout"
	ErrorAIUnavailable     = "error.ai_unavailable"
	ErrorAccessDenied      = "error.access_denied"
	ErrorUserBlocked       = "error.user_blocked"
	ErrorChatBlocked       = "error.chat_blocked"
	ErrorNoPermission      = "error.no_permission"
	ErrorGuessRateLimit    = "error.guess_rate_limit"
)

// StatsNotFound: 전적 조회 관련 메시지 키
const (
	StatsNotFound     = "stats.not_found"
	StatsUserNotFound = "stats.user_not_found"
	StatsHeader       = "stats.header"
	StatsSummary      = "stats.summary"
	StatsCategoryHdr  = "stats.category.header"

	StatsPeriodDaily   = "stats.period.daily"
	StatsPeriodWeekly  = "stats.period.weekly"
	StatsPeriodMonthly = "stats.period.monthly"
	StatsPeriodAll     = "stats.period.all"

	StatsRoomNoGames      = "stats.room.no_games"
	StatsRoomHeader       = "stats.room.header"
	StatsRoomSummary      = "stats.room.summary"
	StatsRoomActivityHdr  = "stats.room.activity_header"
	StatsRoomActivityItem = "stats.room.activity_item"
	StatsNoStats          = "stats.no_stats"
	StatsCategoryResults  = "stats.category.results"
	StatsCategoryAverages = "stats.category.averages"
	StatsCategoryBest     = "stats.category.best"
	StatsCategoryNoBest   = "stats.category.no_best"
)

// AdminForceEndPrefix: 관리자 전용 명령어 관련 메시지 키
const (
	AdminForceEndPrefix  = "admin.force_end_prefix"
	AdminClearAllSuccess = "admin.clear_all_success"
)

// UsageFetchFailed: 토큰 사용량 조회 및 표시 관련 메시지 키
const (
	UsageFetchFailed        = "usage.fetch_failed"
	UsageFetchFailedWeekly  = "usage.fetch_failed_weekly"
	UsageFetchFailedMonthly = "usage.fetch_failed_monthly"
	UsageHeaderToday        = "usage.header_today"
	UsageHeaderWeekly       = "usage.header_weekly"
	UsageHeaderMonthly      = "usage.header_monthly"
	UsageLabelDate          = "usage.label_date"
	UsageLabelInputOutput   = "usage.label_input_output"
	UsageLabelReasoning     = "usage.label_reasoning"
	UsageLabelTotal         = "usage.label_total"
	UsageLabelReqCount      = "usage.label_request_count"
	UsageLabelSum           = "usage.label_sum"
	UsageLabelInput         = "usage.label_input"
	UsageLabelOutput        = "usage.label_output"
	UsageLabelDailySummary  = "usage.label_daily_summary"
	UsageLabelCostHeader    = "usage.label_cost_header"
	UsageLabelCostValue     = "usage.label_cost_value"
	UsageLabelExchangeRate  = "usage.label_exchange_rate"
)

// ChainQueueItem: 체인 질문 처리 관련 메시지 키
const (
	ChainQueueItem       = "chain.queue_item"
	ChainConditionNotMet = "chain.condition_not_met"
)
