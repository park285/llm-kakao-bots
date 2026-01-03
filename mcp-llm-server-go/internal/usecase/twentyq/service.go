package twentyq

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"golang.org/x/text/unicode/norm"

	"github.com/park285/llm-kakao-bots/mcp-llm-server-go/internal/config"
	twentyqdomain "github.com/park285/llm-kakao-bots/mcp-llm-server-go/internal/domain/twentyq"
	"github.com/park285/llm-kakao-bots/mcp-llm-server-go/internal/gemini"
	"github.com/park285/llm-kakao-bots/mcp-llm-server-go/internal/guard"
	"github.com/park285/llm-kakao-bots/mcp-llm-server-go/internal/handler/shared"
	"github.com/park285/llm-kakao-bots/mcp-llm-server-go/internal/httperror"
	"github.com/park285/llm-kakao-bots/mcp-llm-server-go/internal/llm"
	"github.com/park285/llm-kakao-bots/mcp-llm-server-go/internal/prompt"
	"github.com/park285/llm-kakao-bots/mcp-llm-server-go/internal/session"
	"github.com/park285/llm-kakao-bots/mcp-llm-server-go/internal/toon"
)

// Service: TwentyQ 비즈니스 로직(HTTP/gRPC 공용) 구현체입니다.
type Service struct {
	cfg         *config.Config
	client      *gemini.Client
	guard       *guard.InjectionGuard
	store       *session.Store
	prompts     *twentyqdomain.Prompts
	topicLoader *twentyqdomain.TopicLoader
	logger      *slog.Logger
}

// New: TwentyQ Service 인스턴스를 생성합니다.
func New(
	cfg *config.Config,
	client *gemini.Client,
	injectionGuard *guard.InjectionGuard,
	store *session.Store,
	prompts *twentyqdomain.Prompts,
	topicLoader *twentyqdomain.TopicLoader,
	logger *slog.Logger,
) *Service {
	if logger == nil {
		logger = slog.Default()
	}
	return &Service{
		cfg:         cfg,
		client:      client,
		guard:       injectionGuard,
		store:       store,
		prompts:     prompts,
		topicLoader: topicLoader,
		logger:      logger,
	}
}

type AnswerRequest struct {
	SessionID *string
	ChatID    *string
	Namespace *string
	Target    string
	Category  string
	Question  string
	Details   map[string]any
}

type AnswerResult struct {
	RawText   string
	ScaleText string
}

type HintsRequest struct {
	Target   string
	Category string
	Details  map[string]any
}

type VerifyResult struct {
	Result  *string
	RawText string
}

type NormalizeResult struct {
	Original   string
	Normalized string
}

type SynonymResult struct {
	Result  *string
	RawText string
}

func (s *Service) SelectTopic(
	_ context.Context,
	requestID string,
	category string,
	bannedTopics []string,
	excludedCategories []string,
) (twentyqdomain.TopicEntry, error) {
	if s == nil || s.topicLoader == nil {
		return twentyqdomain.TopicEntry{}, httperror.NewInternalError("topic loader not configured")
	}

	topic, err := s.topicLoader.SelectTopic(category, bannedTopics, excludedCategories)
	if err != nil {
		s.logError("twentyq_select_topic_failed", err)
		return twentyqdomain.TopicEntry{}, httperror.NewInternalError("topic selection failed")
	}

	s.logInfo(
		"twentyq_topic_selected",
		"request_id", requestID,
		"category", topic.Category,
		"topic", topic.Name,
		"banned_count", len(bannedTopics),
		"excluded_categories", len(excludedCategories),
	)
	return topic, nil
}

func (s *Service) Categories() []string {
	return twentyqdomain.AllCategories
}

func (s *Service) GenerateHints(ctx context.Context, requestID string, req HintsRequest) ([]string, error) {
	if s == nil || s.client == nil || s.guard == nil || s.prompts == nil {
		return nil, httperror.NewInternalError("service not configured")
	}

	target := strings.TrimSpace(req.Target)
	if target == "" {
		return nil, httperror.NewInvalidInput("target required")
	}

	category := strings.TrimSpace(req.Category)
	if category == "" {
		return nil, httperror.NewInvalidInput("category required")
	}

	system, err := s.prompts.HintsSystem(category)
	if err != nil {
		s.logError("twentyq_hints_system_prompt_failed", err)
		return nil, httperror.NewInternalError("load hints system prompt failed")
	}

	secretToon := toon.EncodeSecret(target, category, nil)
	userContent, err := s.prompts.HintsUser(secretToon)
	if err != nil {
		s.logError("twentyq_hints_user_prompt_failed", err)
		return nil, httperror.NewInternalError("format hints user prompt failed")
	}

	detailsJSON, err := s.serializeDetails(req.Details)
	if err != nil {
		s.logError("twentyq_details_serialize_failed", err)
		return nil, httperror.NewInvalidInput("details must be a JSON object")
	}
	if safeErr := s.ensureSafeDetails(requestID, detailsJSON); safeErr != nil {
		return nil, safeErr
	}
	if detailsJSON != "" {
		userContent = userContent + "\n\n[추가 정보(JSON)]\n" + prompt.WrapXML("details_json", detailsJSON)
	}

	payload, _, err := s.client.Structured(ctx, gemini.Request{
		Prompt:       userContent,
		SystemPrompt: system,
		Task:         "hints",
	}, twentyqdomain.HintsSchema())
	if err != nil {
		return nil, fmt.Errorf("hints structured: %w", err)
	}

	hints, err := shared.ParseStringSlice(payload, "hints")
	if err != nil {
		s.logError("twentyq_hints_parse_failed", err)
		return nil, httperror.NewInternalError("invalid hints response")
	}
	return hints, nil
}

func (s *Service) AnswerQuestion(ctx context.Context, requestID string, req AnswerRequest) (AnswerResult, error) {
	if s == nil || s.guard == nil || s.client == nil || s.prompts == nil {
		return AnswerResult{}, httperror.NewInternalError("service not configured")
	}

	target := strings.TrimSpace(req.Target)
	if target == "" {
		return AnswerResult{}, httperror.NewInvalidInput("target required")
	}

	category := strings.TrimSpace(req.Category)
	if category == "" {
		return AnswerResult{}, httperror.NewInvalidInput("category required")
	}

	question := strings.TrimSpace(req.Question)
	if question == "" {
		return AnswerResult{}, httperror.NewInvalidInput("question required")
	}

	if err := s.guard.EnsureSafe(question); err != nil {
		s.logError("twentyq_question_guard_failed", err)
		return AnswerResult{}, fmt.Errorf("guard question: %w", err)
	}

	detailsJSON, err := s.serializeDetails(req.Details)
	if err != nil {
		s.logError("twentyq_details_serialize_failed", err)
		return AnswerResult{}, httperror.NewInvalidInput("details must be a JSON object")
	}
	if safeErr := s.ensureSafeDetails(requestID, detailsJSON); safeErr != nil {
		return AnswerResult{}, safeErr
	}

	sessionID, history, historyCount, err := s.resolveHistory(ctx, req, "twentyq")
	if err != nil {
		s.logError("session_create_failed", err)
		return AnswerResult{}, err
	}
	s.logAnswerRequest(sessionID, historyCount, question, requestID)

	secretToon := toon.EncodeSecret(target, category, nil)
	system, err := s.prompts.AnswerSystemWithSecret(secretToon)
	if err != nil {
		s.logError("twentyq_answer_system_prompt_failed", err)
		return AnswerResult{}, httperror.NewInternalError("load answer system prompt failed")
	}

	userContent, err := s.prompts.AnswerUser(question)
	if err != nil {
		s.logError("twentyq_answer_user_prompt_failed", err)
		return AnswerResult{}, httperror.NewInternalError("format answer user prompt failed")
	}
	if detailsJSON != "" {
		userContent = userContent + "\n\n[추가 정보(JSON)]\n" + prompt.WrapXML("details_json", detailsJSON)
	}

	rawText, scaleText, err := s.getAnswerText(ctx, system, userContent, history, requestID)
	if err != nil {
		return AnswerResult{}, err
	}
	if rawText == "" {
		return AnswerResult{RawText: "", ScaleText: ""}, nil
	}

	if scaleText != string(twentyqdomain.AnswerPolicyViolation) {
		if err := s.appendAnswerHistory(ctx, sessionID, question, scaleText); err != nil {
			s.logError("twentyq_append_history_failed", err)
			return AnswerResult{}, err
		}
	}

	return AnswerResult{RawText: rawText, ScaleText: scaleText}, nil
}

func (s *Service) VerifyGuess(ctx context.Context, requestID string, target string, guess string) (VerifyResult, error) {
	if s == nil || s.guard == nil || s.client == nil || s.prompts == nil {
		return VerifyResult{}, httperror.NewInternalError("service not configured")
	}

	target = strings.TrimSpace(target)
	if target == "" {
		return VerifyResult{}, httperror.NewInvalidInput("target required")
	}

	guess = strings.TrimSpace(guess)
	if guess == "" {
		return VerifyResult{}, httperror.NewInvalidInput("guess required")
	}

	// Fast-path: 정규화 후 완전 일치 검사 → LLM 호출 없이 즉시 ACCEPT
	// 비용 절감 + 결정론적 판정 (LLM 환각 위험 제거)
	if normalizeForCompare(target) == normalizeForCompare(guess) {
		resultStr := string(twentyqdomain.VerifyAccept)
		s.logInfo(
			"twentyq_verify_result",
			"request_id", requestID,
			"path", "fast", // 대시보드 필터링용: fast | llm
			"result", resultStr, // 정답 | 근접 | 오답
			"target", target,
			"guess", guess,
			"llm_calls", 0, // Fast-path는 LLM 호출 0회
			"cost_saved", true, // 비용 절감 여부
		)
		return VerifyResult{Result: &resultStr, RawText: resultStr}, nil
	}

	if err := s.guard.EnsureSafe(guess); err != nil {
		s.logError("twentyq_guess_guard_failed", err)
		return VerifyResult{}, fmt.Errorf("guard guess: %w", err)
	}

	system, userContent, err := s.buildVerifyPrompts(target, guess)
	if err != nil {
		return VerifyResult{}, err
	}

	const consensusCalls = 3
	consensus, err := s.client.StructuredWithConsensusWeighted(ctx, gemini.Request{
		Prompt:       userContent,
		SystemPrompt: system,
		Task:         "verify",
	}, twentyqdomain.VerifySchema(), "result", consensusCalls)
	if err != nil {
		return VerifyResult{}, fmt.Errorf("verify structured: %w", err)
	}

	// LLM 호출이 필요 없었을 경우 위에서 반환했으므로 여기서는 기존 흐름 유지
	s.logVerifyConsensus(requestID, consensus)

	return s.parseVerifyGuessPayload(consensus.Payload), nil
}

func (s *Service) buildVerifyPrompts(target string, guess string) (string, string, error) {
	system, err := s.prompts.VerifySystem()
	if err != nil {
		s.logError("twentyq_verify_system_prompt_failed", err)
		return "", "", httperror.NewInternalError("load verify system prompt failed")
	}
	userContent, err := s.prompts.VerifyUser(target, guess)
	if err != nil {
		s.logError("twentyq_verify_user_prompt_failed", err)
		return "", "", httperror.NewInternalError("format verify user prompt failed")
	}
	return system, userContent, nil
}

func (s *Service) logVerifyConsensus(requestID string, consensus gemini.ConsensusResult) {
	payload := consensus.Payload
	if reasoning, ok := payload["reasoning"].(string); ok && reasoning != "" {
		s.logger.Debug("twentyq_verify_cot", "request_id", requestID, "reasoning", reasoning)
	}
	if len(consensus.SearchQueries) > 0 {
		s.logger.Debug("twentyq_verify_search", "request_id", requestID, "queries", consensus.SearchQueries)
	}
	if len(consensus.Votes) > 0 {
		votesByValue := make(map[string]int)
		for _, vote := range consensus.Votes {
			votesByValue[vote.Value]++
		}
		if len(votesByValue) > 1 {
			trimmedVotes := make([]gemini.ConsensusVote, 0, len(consensus.Votes))
			for _, vote := range consensus.Votes {
				trimmedVotes = append(trimmedVotes, gemini.ConsensusVote{
					Value:      vote.Value,
					Confidence: vote.Confidence,
					Reasoning:  shared.TrimRunes(vote.Reasoning, 200),
				})
			}
			s.logger.Debug(
				"twentyq_verify_consensus_votes",
				"request_id", requestID,
				"winning_value", consensus.ConsensusValue,
				"winning_count", consensus.ConsensusCount,
				"winning_weight", consensus.ConsensusWeight,
				"total_weight", consensus.TotalWeight,
				"successful_calls", consensus.SuccessfulCalls,
				"total_calls", consensus.TotalCalls,
				"votes", trimmedVotes,
			)
		}
	}

	// 만장일치 여부 판단 (합의 분석용)
	unanimous := consensus.ConsensusCount == consensus.SuccessfulCalls && consensus.SuccessfulCalls > 0

	s.logInfo(
		"twentyq_verify_result", // Fast-path와 동일한 이벤트명
		"request_id", requestID,
		"path", "llm", // 대시보드 필터링용: fast | llm
		"result", consensus.ConsensusValue,
		"llm_calls", consensus.TotalCalls, // LLM 호출 횟수
		"success", consensus.SuccessfulCalls,
		"count", consensus.ConsensusCount, // 합의 투표 수
		"weight", consensus.ConsensusWeight,
		"total_weight", consensus.TotalWeight,
		"unanimous", unanimous, // 만장일치 여부
		"cost_saved", false, // LLM 호출했으므로 비용 절감 아님
	)
}

func (s *Service) parseVerifyGuessPayload(payload map[string]any) VerifyResult {
	rawValue, parseErr := shared.ParseStringField(payload, "result")
	if parseErr != nil {
		s.logError("twentyq_verify_parse_failed", parseErr)
	}

	var result *string
	if parseErr == nil {
		resultName, ok := twentyqdomain.VerifyResultName(rawValue)
		if ok {
			result = &resultName
		}
	}
	return VerifyResult{Result: result, RawText: rawValue}
}

func (s *Service) NormalizeQuestion(ctx context.Context, requestID string, question string) (NormalizeResult, error) {
	if s == nil || s.guard == nil || s.client == nil || s.prompts == nil {
		return NormalizeResult{}, httperror.NewInternalError("service not configured")
	}

	question = strings.TrimSpace(question)
	if question == "" {
		return NormalizeResult{}, httperror.NewInvalidInput("question required")
	}

	if err := s.guard.EnsureSafe(question); err != nil {
		s.logError("twentyq_question_guard_failed", err)
		return NormalizeResult{}, fmt.Errorf("guard question: %w", err)
	}

	system, err := s.prompts.NormalizeSystem()
	if err != nil {
		s.logError("twentyq_normalize_system_prompt_failed", err)
		return NormalizeResult{}, httperror.NewInternalError("load normalize system prompt failed")
	}
	userContent, err := s.prompts.NormalizeUser(question)
	if err != nil {
		s.logError("twentyq_normalize_user_prompt_failed", err)
		return NormalizeResult{}, httperror.NewInternalError("format normalize user prompt failed")
	}

	normalized := question
	payload, _, err := s.client.Structured(ctx, gemini.Request{
		Prompt:       userContent,
		SystemPrompt: system,
	}, twentyqdomain.NormalizeSchema())
	if err == nil {
		if rawValue, parseErr := shared.ParseStringField(payload, "normalized"); parseErr == nil {
			normalized = rawValue
		} else {
			s.logError("twentyq_normalize_parse_failed", parseErr)
		}
	} else {
		s.logError("twentyq_normalize_failed", err)
	}

	return NormalizeResult{Original: question, Normalized: normalized}, nil
}

func (s *Service) CheckSynonym(ctx context.Context, requestID string, target string, guess string) (SynonymResult, error) {
	if s == nil || s.guard == nil || s.client == nil || s.prompts == nil {
		return SynonymResult{}, httperror.NewInternalError("service not configured")
	}

	target = strings.TrimSpace(target)
	if target == "" {
		return SynonymResult{}, httperror.NewInvalidInput("target required")
	}

	guess = strings.TrimSpace(guess)
	if guess == "" {
		return SynonymResult{}, httperror.NewInvalidInput("guess required")
	}

	if err := s.guard.EnsureSafe(guess); err != nil {
		s.logError("twentyq_guess_guard_failed", err)
		return SynonymResult{}, fmt.Errorf("guard guess: %w", err)
	}

	system, err := s.prompts.SynonymSystem()
	if err != nil {
		s.logError("twentyq_synonym_system_prompt_failed", err)
		return SynonymResult{}, httperror.NewInternalError("load synonym system prompt failed")
	}
	userContent, err := s.prompts.SynonymUser(target, guess)
	if err != nil {
		s.logError("twentyq_synonym_user_prompt_failed", err)
		return SynonymResult{}, httperror.NewInternalError("format synonym user prompt failed")
	}

	payload, _, err := s.client.Structured(ctx, gemini.Request{
		Prompt:       userContent,
		SystemPrompt: system,
		Task:         "synonym",
	}, twentyqdomain.SynonymSchema())
	if err != nil {
		return SynonymResult{}, fmt.Errorf("synonym structured: %w", err)
	}

	rawValue, parseErr := shared.ParseStringField(payload, "result")
	if parseErr != nil {
		s.logError("twentyq_synonym_parse_failed", parseErr)
	}

	var result *string
	if parseErr == nil {
		resultName, ok := twentyqdomain.SynonymResultName(rawValue)
		if ok {
			result = &resultName
		}
	}
	return SynonymResult{Result: result, RawText: rawValue}, nil
}

func (s *Service) serializeDetails(details map[string]any) (string, error) {
	if len(details) == 0 {
		return "", nil
	}
	value, err := shared.SerializeDetails(details)
	if err != nil {
		return "", fmt.Errorf("serialize details: %w", err)
	}
	return value, nil
}

func (s *Service) ensureSafeDetails(requestID string, detailsJSON string) error {
	if detailsJSON == "" {
		return nil
	}
	if s.guard == nil {
		return httperror.NewInternalError("guard not configured")
	}
	if err := s.guard.EnsureSafe(detailsJSON); err != nil {
		s.logError("twentyq_details_guard_failed", err)
		return httperror.NewInvalidInput("details blocked")
	}
	return nil
}

func (s *Service) resolveHistory(ctx context.Context, req AnswerRequest, defaultNamespace string) (string, []llm.HistoryEntry, int, error) {
	effectiveSessionID, derived := shared.ResolveSessionID(
		shared.ValueOrEmpty(req.SessionID),
		shared.ValueOrEmpty(req.ChatID),
		shared.ValueOrEmpty(req.Namespace),
		defaultNamespace,
	)

	if effectiveSessionID != "" && derived && req.SessionID == nil && s.store != nil && s.cfg != nil {
		now := time.Now()
		meta := session.Meta{
			ID:           effectiveSessionID,
			SystemPrompt: "",
			Model:        s.cfg.Gemini.DefaultModel,
			CreatedAt:    now,
			UpdatedAt:    now,
			MessageCount: 0,
		}
		if err := s.store.CreateSession(ctx, meta); err != nil {
			return "", nil, 0, fmt.Errorf("create session: %w", err)
		}
	}

	if effectiveSessionID == "" || s.store == nil {
		return "", []llm.HistoryEntry{}, 0, nil
	}

	history, err := s.store.GetHistory(ctx, effectiveSessionID)
	if err != nil {
		s.logError("session_history_failed", err)
		return effectiveSessionID, []llm.HistoryEntry{}, 0, nil
	}
	return effectiveSessionID, history, len(history), nil
}

func (s *Service) logAnswerRequest(sessionID string, historyCount int, question string, requestID string) {
	sessionLabel := sessionID
	if sessionLabel == "" {
		sessionLabel = "stateless"
	}
	questionCount := historyCount/2 + 1

	fields := []any{
		"request_id", requestID,
		"session", sessionLabel,
		"count", questionCount,
		"history_count", historyCount,
		"q_len", len(question),
	}
	s.logger.Info("twentyq_answer", fields...)
}

func (s *Service) getAnswerText(
	ctx context.Context,
	system string,
	userContent string,
	history []llm.HistoryEntry,
	requestID string,
) (string, string, error) {
	result, err := s.client.StructuredWithSearch(ctx, gemini.Request{
		Prompt:       userContent,
		SystemPrompt: system,
		History:      history,
		Task:         "answer",
	}, twentyqdomain.AnswerSchema())
	if err != nil {
		return "", "", fmt.Errorf("answer structured: %w", err)
	}

	if reasoning, ok := result.Payload["reasoning"].(string); ok && reasoning != "" {
		s.logger.Debug("twentyq_cot", "request_id", requestID, "reasoning", reasoning)
	}
	if len(result.SearchQueries) > 0 {
		s.logger.Debug("twentyq_search", "request_id", requestID, "queries", result.SearchQueries)
	}

	rawValue, ok := result.Payload["answer"].(string)
	if !ok || rawValue == "" {
		return "", "", nil
	}

	scale, ok := twentyqdomain.ParseAnswerScale(rawValue)
	scaleText := ""
	if ok {
		scaleText = string(scale)
	}
	return rawValue, scaleText, nil
}

func (s *Service) appendAnswerHistory(ctx context.Context, sessionID string, question string, scaleText string) error {
	if sessionID == "" || s.store == nil {
		return nil
	}

	historyScaleText := "UNKNOWN"
	if scaleText != "" {
		historyScaleText = scaleText
	}

	if err := s.store.AppendHistory(
		ctx,
		sessionID,
		llm.HistoryEntry{Role: "user", Content: "Q: " + question},
		llm.HistoryEntry{Role: "assistant", Content: "A: " + historyScaleText},
	); err != nil {
		return fmt.Errorf("append history: %w", err)
	}
	return nil
}

func (s *Service) logError(event string, err error) {
	if s == nil || s.logger == nil || err == nil {
		return
	}
	s.logger.Warn(event, "err", err)
}

func (s *Service) logInfo(event string, fields ...any) {
	if s == nil || s.logger == nil {
		return
	}
	s.logger.Info(event, fields...)
}

// normalizeForCompare: 문자열 비교를 위한 정규화를 수행합니다.
// 1. 유니코드 NFC 정규화 (조합형/완성형 통일)
// 2. 모든 공백 제거
// 3. 소문자 변환 (영문 대소문자 무시)
func normalizeForCompare(s string) string {
	// NFC 정규화: NFD(분해형)로 입력된 한글도 NFC(완성형)로 통일
	s = norm.NFC.String(s)
	// 모든 공백 제거 (중간 공백 포함)
	s = strings.ReplaceAll(s, " ", "")
	s = strings.ReplaceAll(s, "\t", "")
	// 소문자 변환 (영문 대소문자 무시)
	return strings.ToLower(s)
}
