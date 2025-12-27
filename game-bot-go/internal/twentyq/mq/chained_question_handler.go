package mq

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/park285/llm-kakao-bots/game-bot-go/internal/common/messageprovider"
	"github.com/park285/llm-kakao-bots/game-bot-go/internal/common/mqmsg"
	domainmodels "github.com/park285/llm-kakao-bots/game-bot-go/internal/domain/models"
	qmessages "github.com/park285/llm-kakao-bots/game-bot-go/internal/twentyq/messages"
	qmodel "github.com/park285/llm-kakao-bots/game-bot-go/internal/twentyq/model"
	qredis "github.com/park285/llm-kakao-bots/game-bot-go/internal/twentyq/redis"
	qsvc "github.com/park285/llm-kakao-bots/game-bot-go/internal/twentyq/service"
)

// ChainedQuestionRiddleService: 체인 질문 처리에 필요한 RiddleService 인터페이스
type ChainedQuestionRiddleService interface {
	AnswerWithOutcome(ctx context.Context, chatID string, userID string, sender *string, question string, isChain bool) (qsvc.AnswerOutcome, error)
	StatusSeparated(ctx context.Context, chatID string) (string, string, error)
	StatusSeparatedWithCount(ctx context.Context, chatID string) (string, string, int, error)
}

// ChainedQuestionQueueCoordinator: 체인 질문 처리에 필요한 큐 코디네이터 인터페이스
type ChainedQuestionQueueCoordinator interface {
	Enqueue(ctx context.Context, chatID string, msg qmodel.PendingMessage) (qredis.EnqueueResult, error)
	SetChainSkipFlag(ctx context.Context, chatID string, userID string) error
	CheckAndClearChainSkipFlag(ctx context.Context, chatID string, userID string) (bool, error)
}

// ChainedQuestionHandler: 사용자가 쉼표 등으로 구분하여 입력한 연속 질문(Chain Question)을 관리하는 핸들러.
// 첫 번째 질문은 즉시 실행하고, 나머지는 백그라운드 큐에 배정하여 순차적으로 처리되도록 한다.
type ChainedQuestionHandler struct {
	riddleService    ChainedQuestionRiddleService
	queueCoordinator ChainedQuestionQueueCoordinator
	msgProvider      *messageprovider.Provider
	logger           *slog.Logger
}

// NewChainedQuestionHandler 생성자.
func NewChainedQuestionHandler(
	riddleService ChainedQuestionRiddleService,
	queueCoordinator ChainedQuestionQueueCoordinator,
	msgProvider *messageprovider.Provider,
	logger *slog.Logger,
) *ChainedQuestionHandler {
	return &ChainedQuestionHandler{
		riddleService:    riddleService,
		queueCoordinator: queueCoordinator,
		msgProvider:      msgProvider,
		logger:           logger,
	}
}

// PrepareChainQueue: 첫 번째 질문을 제외한 나머지 질문들을 Redis 대기열에 '배치(Batch)' 형태로 미리 등록(Enqueue)한다.
// 또한 사용자에게 대기열에 등록되었음을 알리는 메시지를 생성하여 반환한다.
func (h *ChainedQuestionHandler) PrepareChainQueue(
	ctx context.Context,
	chatID string,
	userID string,
	sender *string,
	questions []string,
) (string, error) {
	if len(questions) <= 1 {
		return "", nil
	}

	displaySender := domainmodels.DisplayName(chatID, userID, sender, h.msgProvider.Get(qmessages.UserAnonymous))

	remainingQuestions := questions[1:]

	// 나머지 질문들을 즉시 큐에 추가 (optimistic enqueuing)
	chainMessage := qmodel.PendingMessage{
		UserID:         userID,
		Content:        "", // 체인 메시지는 content 사용 안 함
		ThreadID:       nil,
		Sender:         &displaySender,
		Timestamp:      time.Now().UnixMilli(),
		IsChainBatch:   true,
		BatchQuestions: remainingQuestions,
	}

	result, err := h.queueCoordinator.Enqueue(ctx, chatID, chainMessage)
	if err != nil {
		h.logger.Warn("chain_queue_enqueue_failed", "chatID", chatID, "err", err)
		// 큐잉 실패해도 첫 질문은 처리
	} else {
		h.logger.Info("chain_queue_enqueued", "chatID", chatID, "result", result, "questions", len(remainingQuestions))
	}

	// 큐 안내 메시지 생성
	var queueDetails strings.Builder
	for i, question := range remainingQuestions {
		item := h.msgProvider.Get(qmessages.ChainQueueItem,
			messageprovider.P("index", i+1),
			messageprovider.P("question", question))
		queueDetails.WriteString(item)
		if i < len(remainingQuestions)-1 {
			queueDetails.WriteString("\n")
		}
	}

	return h.msgProvider.Get(qmessages.LockMessageQueued,
		messageprovider.P("user", displaySender),
		messageprovider.P("queueDetails", queueDetails.String())), nil
}

// Handle: 연속 질문 중 첫 번째 질문을 즉시 실행하여 결과를 반환한다.
// 이후 질문 진행 여부(조건)를 판단하고, 중단이 필요하면 스킵 플래그를 설정한다.
func (h *ChainedQuestionHandler) Handle(
	ctx context.Context,
	chatID string,
	userID string,
	sender *string,
	questions []string,
	condition qmodel.ChainCondition,
) (string, error) {
	if len(questions) == 0 {
		return h.msgProvider.Get(qmessages.ErrorInvalidQuestion), nil
	}

	firstQuestion := questions[0]

	// 첫 번째 질문 처리
	outcome, err := h.riddleService.AnswerWithOutcome(ctx, chatID, userID, sender, firstQuestion, false)
	if err != nil {
		return "", fmt.Errorf("chain first question failed: %w", err)
	}

	// 조건 평가 및 스킵 플래그 설정
	hasRemainingQuestions := len(questions) > 1
	shouldContinue := true
	if hasRemainingQuestions {
		shouldContinue = condition.ShouldContinue(outcome.Scale)
		if !shouldContinue {
			// 체인 스킵 플래그 설정
			if err := h.queueCoordinator.SetChainSkipFlag(ctx, chatID, userID); err != nil {
				h.logger.Warn("set_chain_skip_flag_failed", "chatID", chatID, "err", err)
			}
		}
	}

	// 스킵 알림 메시지 포함
	if !shouldContinue && hasRemainingQuestions {
		skippedQuestions := questions[1:]
		skipNotification := h.msgProvider.Get(qmessages.ChainConditionNotMet,
			messageprovider.P("questions", strings.Join(skippedQuestions, ", ")))
		return outcome.Message + "\n\n" + skipNotification, nil
	}

	return outcome.Message, nil
}

// ProcessChainBatch: 대기열에서 꺼낸 나머지 연속 질문들을 순차적으로 실행한다. (백그라운드 처리)
// 실행 전 스킵 플래그를 확인하여, 이전 질문의 결과에 따라 중단 여부를 결정한다.
func (h *ChainedQuestionHandler) ProcessChainBatch(
	ctx context.Context,
	chatID string,
	pending qmodel.PendingMessage,
	emit func(mqmsg.OutboundMessage) error,
) error {
	if !pending.IsChainBatch || len(pending.BatchQuestions) == 0 {
		return nil
	}

	// 체인 스킵 플래그 확인
	skipped, err := h.queueCoordinator.CheckAndClearChainSkipFlag(ctx, chatID, pending.UserID)
	if err != nil {
		h.logger.Warn("check_chain_skip_flag_failed", "chatID", chatID, "err", err)
	}
	if skipped {
		h.logger.Debug("chain_batch_skipped", "chatID", chatID, "userID", pending.UserID)
		skipNotification := h.msgProvider.Get(
			qmessages.ChainConditionNotMet,
			messageprovider.P("questions", strings.Join(pending.BatchQuestions, ", ")),
		)
		return emit(mqmsg.NewFinal(chatID, skipNotification, pending.ThreadID))
	}

	// 각 질문 순차 처리 (응답은 전송하지 않음)
	for i, question := range pending.BatchQuestions {
		if _, answerErr := h.riddleService.AnswerWithOutcome(ctx, chatID, pending.UserID, pending.Sender, question, true); answerErr != nil {
			h.logger.Warn("chain_question_failed", "chatID", chatID, "index", i, "err", answerErr)
		}
	}

	// 상태 메시지 조회 및 전송
	main, hint, questionCount, err := h.riddleService.StatusSeparatedWithCount(ctx, chatID)
	if err != nil {
		h.logger.Warn("chain_status_failed", "chatID", chatID, "err", err)
		main = h.msgProvider.Get(qmessages.ErrorNoSessionShort)
		hint = ""
		questionCount = 0
	}

	messages := []string{main}
	if shouldShowHint(hint, questionCount) {
		messages = append(messages, hint)
	}
	return emitChunkedResponses(chatID, pending.ThreadID, messages, emit)
}
