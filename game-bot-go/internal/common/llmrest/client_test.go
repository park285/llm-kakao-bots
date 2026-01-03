package llmrest

import (
	"context"
	"fmt"
	"net"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/types/known/emptypb"

	llmv1 "github.com/park285/llm-kakao-bots/game-bot-go/internal/common/llmrest/pb/llm/v1"
	"google.golang.org/protobuf/types/known/structpb"
)

type grpcTestService struct {
	llmv1.UnimplementedLLMServiceServer
	apiKey string
	t      *testing.T
}

func (s *grpcTestService) checkAPIKey(ctx context.Context) {
	if s == nil || s.t == nil {
		return
	}

	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		s.t.Errorf("missing grpc metadata")
		return
	}

	values := md.Get("x-api-key")
	if len(values) == 0 || values[0] != s.apiKey {
		s.t.Errorf("unexpected api key: %v", values)
	}
}

func (s *grpcTestService) GetModelConfig(ctx context.Context, _ *emptypb.Empty) (*llmv1.ModelConfigResponse, error) {
	s.checkAPIKey(ctx)

	hints := "hints"
	transport := "h2c"
	return &llmv1.ModelConfigResponse{
		ModelDefault:   "gemini-3-test",
		ModelHints:     &hints,
		TimeoutSeconds: 60,
		MaxRetries:     3,
		Http2Enabled:   true,
		TransportMode:  &transport,
		Temperature:    1.0,
	}, nil
}

func (s *grpcTestService) EndSession(ctx context.Context, req *llmv1.EndSessionRequest) (*llmv1.EndSessionResponse, error) {
	s.checkAPIKey(ctx)

	if req == nil || req.SessionId == "" {
		return nil, fmt.Errorf("session_id required")
	}

	return &llmv1.EndSessionResponse{Message: "session deleted", Id: req.SessionId}, nil
}

func (s *grpcTestService) GuardIsMalicious(ctx context.Context, req *llmv1.GuardIsMaliciousRequest) (*llmv1.GuardIsMaliciousResponse, error) {
	s.checkAPIKey(ctx)

	if req == nil {
		return nil, fmt.Errorf("request required")
	}

	return &llmv1.GuardIsMaliciousResponse{Malicious: req.InputText == "malicious"}, nil
}

func (s *grpcTestService) TwentyQSelectTopic(ctx context.Context, req *llmv1.TwentyQSelectTopicRequest) (*llmv1.TwentyQSelectTopicResponse, error) {
	s.checkAPIKey(ctx)

	if req == nil {
		return nil, fmt.Errorf("request required")
	}

	details, err := structpb.NewStruct(map[string]any{"source": "grpc"})
	if err != nil {
		return nil, err
	}

	return &llmv1.TwentyQSelectTopicResponse{
		Name:     "TOPIC",
		Category: req.Category,
		Details:  details,
	}, nil
}

func (s *grpcTestService) TwentyQGetCategories(ctx context.Context, _ *emptypb.Empty) (*llmv1.TwentyQGetCategoriesResponse, error) {
	s.checkAPIKey(ctx)

	return &llmv1.TwentyQGetCategoriesResponse{Categories: []string{"ANIMALS", "FOODS"}}, nil
}

func (s *grpcTestService) TwentyQGenerateHints(ctx context.Context, req *llmv1.TwentyQGenerateHintsRequest) (*llmv1.TwentyQGenerateHintsResponse, error) {
	s.checkAPIKey(ctx)

	if req == nil {
		return nil, fmt.Errorf("request required")
	}
	if req.Target == "" || req.Category == "" {
		return nil, fmt.Errorf("target/category required")
	}

	if req.Details == nil {
		return nil, fmt.Errorf("details required")
	}
	if req.Details.AsMap()["foo"] != "bar" {
		return nil, fmt.Errorf("unexpected details: %v", req.Details.AsMap())
	}

	return &llmv1.TwentyQGenerateHintsResponse{Hints: []string{"hint-1", "hint-2"}}, nil
}

func (s *grpcTestService) TwentyQAnswerQuestion(ctx context.Context, req *llmv1.TwentyQAnswerQuestionRequest) (*llmv1.TwentyQAnswerQuestionResponse, error) {
	s.checkAPIKey(ctx)

	if req == nil {
		return nil, fmt.Errorf("request required")
	}
	if req.ChatId == nil || *req.ChatId != "chat-1" {
		return nil, fmt.Errorf("chat_id mismatch")
	}
	if req.Namespace == nil || *req.Namespace != "twentyq" {
		return nil, fmt.Errorf("namespace mismatch")
	}
	if req.Target != "cat" || req.Category != "ANIMALS" {
		return nil, fmt.Errorf("target/category mismatch")
	}
	if req.Question != "Q?" {
		return nil, fmt.Errorf("question mismatch")
	}
	if req.Details == nil || req.Details.AsMap()["foo"] != "bar" {
		return nil, fmt.Errorf("details mismatch")
	}

	scale := "YES"
	return &llmv1.TwentyQAnswerQuestionResponse{Scale: &scale, RawText: "YES"}, nil
}

func (s *grpcTestService) TwentyQVerifyGuess(ctx context.Context, req *llmv1.TwentyQVerifyGuessRequest) (*llmv1.TwentyQVerifyGuessResponse, error) {
	s.checkAPIKey(ctx)

	if req == nil {
		return nil, fmt.Errorf("request required")
	}
	if req.Target != "cat" || req.Guess != "cat" {
		return nil, fmt.Errorf("target/guess mismatch")
	}

	result := "CORRECT"
	return &llmv1.TwentyQVerifyGuessResponse{Result: &result, RawText: "CORRECT"}, nil
}

func (s *grpcTestService) TwentyQNormalizeQuestion(ctx context.Context, req *llmv1.TwentyQNormalizeQuestionRequest) (*llmv1.TwentyQNormalizeQuestionResponse, error) {
	s.checkAPIKey(ctx)

	if req == nil {
		return nil, fmt.Errorf("request required")
	}

	return &llmv1.TwentyQNormalizeQuestionResponse{Normalized: "normalized", Original: req.Question}, nil
}

func (s *grpcTestService) TwentyQCheckSynonym(ctx context.Context, req *llmv1.TwentyQCheckSynonymRequest) (*llmv1.TwentyQCheckSynonymResponse, error) {
	s.checkAPIKey(ctx)

	if req == nil {
		return nil, fmt.Errorf("request required")
	}

	result := "SAME"
	return &llmv1.TwentyQCheckSynonymResponse{Result: &result, RawText: "SAME"}, nil
}

func (s *grpcTestService) TurtleSoupGeneratePuzzle(ctx context.Context, req *llmv1.TurtleSoupGeneratePuzzleRequest) (*llmv1.TurtleSoupGeneratePuzzleResponse, error) {
	s.checkAPIKey(ctx)

	if req == nil {
		return nil, fmt.Errorf("request required")
	}
	if req.Difficulty == nil || req.GetDifficulty() != 3 {
		return nil, fmt.Errorf("difficulty mismatch")
	}
	if req.Category == nil || req.GetCategory() != "MYSTERY" {
		return nil, fmt.Errorf("category mismatch")
	}
	if req.Theme == nil || req.GetTheme() != "space" {
		return nil, fmt.Errorf("theme mismatch")
	}

	return &llmv1.TurtleSoupGeneratePuzzleResponse{
		Title:      "title",
		Scenario:   "scenario",
		Solution:   "solution",
		Category:   req.GetCategory(),
		Difficulty: req.GetDifficulty(),
		Hints:      []string{"h1"},
	}, nil
}

func (s *grpcTestService) TurtleSoupGetRandomPuzzle(ctx context.Context, req *llmv1.TurtleSoupGetRandomPuzzleRequest) (*llmv1.TurtleSoupGetRandomPuzzleResponse, error) {
	s.checkAPIKey(ctx)

	if req == nil {
		return nil, fmt.Errorf("request required")
	}
	if req.Difficulty == nil || req.GetDifficulty() != 2 {
		return nil, fmt.Errorf("difficulty mismatch")
	}

	id := int32(10)
	title := "preset"
	question := "question"
	answer := "answer"
	difficulty := int32(2)

	return &llmv1.TurtleSoupGetRandomPuzzleResponse{
		Id:         &id,
		Title:      &title,
		Question:   &question,
		Answer:     &answer,
		Difficulty: &difficulty,
	}, nil
}

func (s *grpcTestService) TurtleSoupRewriteScenario(ctx context.Context, req *llmv1.TurtleSoupRewriteScenarioRequest) (*llmv1.TurtleSoupRewriteScenarioResponse, error) {
	s.checkAPIKey(ctx)

	if req == nil {
		return nil, fmt.Errorf("request required")
	}
	if req.Difficulty != 3 {
		return nil, fmt.Errorf("difficulty mismatch")
	}

	return &llmv1.TurtleSoupRewriteScenarioResponse{
		Scenario:         "rewritten",
		Solution:         "rewritten-solution",
		OriginalScenario: req.Scenario,
		OriginalSolution: req.Solution,
	}, nil
}

func (s *grpcTestService) TurtleSoupAnswerQuestion(ctx context.Context, req *llmv1.TurtleSoupAnswerQuestionRequest) (*llmv1.TurtleSoupAnswerQuestionResponse, error) {
	s.checkAPIKey(ctx)

	if req == nil {
		return nil, fmt.Errorf("request required")
	}
	if req.ChatId == nil || *req.ChatId != "chat-1" {
		return nil, fmt.Errorf("chat_id mismatch")
	}
	if req.Namespace == nil || *req.Namespace != "turtle-soup" {
		return nil, fmt.Errorf("namespace mismatch")
	}
	if req.Question != "Q?" {
		return nil, fmt.Errorf("question mismatch")
	}

	return &llmv1.TurtleSoupAnswerQuestionResponse{
		Answer:        "YES",
		RawText:       "RAW",
		QuestionCount: 1,
		History:       []*llmv1.TurtleSoupHistoryItem{{Question: "Q?", Answer: "YES"}},
	}, nil
}

func (s *grpcTestService) TurtleSoupValidateSolution(ctx context.Context, req *llmv1.TurtleSoupValidateSolutionRequest) (*llmv1.TurtleSoupValidateSolutionResponse, error) {
	s.checkAPIKey(ctx)

	if req == nil {
		return nil, fmt.Errorf("request required")
	}
	if req.PlayerAnswer != "answer" {
		return nil, fmt.Errorf("player_answer mismatch")
	}

	return &llmv1.TurtleSoupValidateSolutionResponse{Result: "YES", RawText: "YES"}, nil
}

func (s *grpcTestService) TurtleSoupGenerateHint(ctx context.Context, req *llmv1.TurtleSoupGenerateHintRequest) (*llmv1.TurtleSoupGenerateHintResponse, error) {
	s.checkAPIKey(ctx)

	if req == nil {
		return nil, fmt.Errorf("request required")
	}
	if req.Level != 1 {
		return nil, fmt.Errorf("level mismatch")
	}

	return &llmv1.TurtleSoupGenerateHintResponse{Hint: "HINT", Level: req.Level}, nil
}

func (s *grpcTestService) GetDailyUsage(ctx context.Context, _ *emptypb.Empty) (*llmv1.DailyUsageResponse, error) {
	s.checkAPIKey(ctx)

	return &llmv1.DailyUsageResponse{UsageDate: "2025-01-01", InputTokens: 1, OutputTokens: 2, TotalTokens: 3, ReasoningTokens: 0, RequestCount: 1, Model: "gemini-3-test"}, nil
}

func (s *grpcTestService) GetRecentUsage(ctx context.Context, req *llmv1.GetRecentUsageRequest) (*llmv1.UsageListResponse, error) {
	s.checkAPIKey(ctx)

	if req == nil {
		return nil, fmt.Errorf("request required")
	}
	if req.Days != 7 {
		return nil, fmt.Errorf("days mismatch")
	}

	usage := &llmv1.DailyUsageResponse{UsageDate: "2025-01-01", InputTokens: 1, OutputTokens: 2, TotalTokens: 3, ReasoningTokens: 0, RequestCount: 1, Model: "gemini-3-test"}
	return &llmv1.UsageListResponse{Usages: []*llmv1.DailyUsageResponse{usage}, TotalInputTokens: 1, TotalOutputTokens: 2, TotalTokens: 3, TotalRequestCount: 1, Model: "gemini-3-test"}, nil
}

func (s *grpcTestService) GetTotalUsage(ctx context.Context, req *llmv1.GetTotalUsageRequest) (*llmv1.UsageResponse, error) {
	s.checkAPIKey(ctx)

	if req == nil {
		return nil, fmt.Errorf("request required")
	}

	switch req.Days {
	case 0:
		return &llmv1.UsageResponse{InputTokens: 1, OutputTokens: 2, TotalTokens: 3, ReasoningTokens: 0, Model: "gemini-3-test"}, nil
	case 30:
		return &llmv1.UsageResponse{InputTokens: 30, OutputTokens: 60, TotalTokens: 90, ReasoningTokens: 0, Model: "gemini-3-test"}, nil
	default:
		return nil, fmt.Errorf("days mismatch: %d", req.Days)
	}
}

func TestNew(t *testing.T) {
	tests := []struct {
		name    string
		baseURL string
		wantErr bool
	}{
		{"valid grpc", "grpc://example.com:40528", false},
		{"valid grpc no port", "grpc://example.com", false},
		{"valid unix", "unix:///var/run/grpc/llm.sock", false},
		{"valid unix relative", "unix://./test.sock", false},
		{"unix empty path", "unix://", true},
		{"grpcs not allowed", "grpcs://example.com:40528", true},
		{"http not allowed", "http://example.com", true},
		{"https not allowed", "https://example.com", true},
		{"empty", "", true},
		{"no scheme", "example.com", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := New(Config{BaseURL: tt.baseURL})
			if (err != nil) != tt.wantErr {
				t.Errorf("New() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestClient_GRPC_GetModelConfig(t *testing.T) {
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen failed: %v", err)
	}
	grpcServer := grpc.NewServer()
	llmv1.RegisterLLMServiceServer(grpcServer, &grpcTestService{t: t, apiKey: "test-key"})

	go func() {
		_ = grpcServer.Serve(lis)
	}()

	t.Cleanup(func() {
		grpcServer.Stop()
		_ = lis.Close()
	})

	client, err := New(Config{BaseURL: "grpc://" + lis.Addr().String(), APIKey: "test-key", Timeout: 2 * time.Second})
	if err != nil {
		t.Fatalf("failed to create grpc client: %v", err)
	}
	t.Cleanup(func() {
		_ = client.Close()
	})

	cfg, err := client.GetModelConfig(context.Background())
	if err != nil {
		t.Fatalf("GetModelConfig failed: %v", err)
	}
	if cfg.ModelDefault != "gemini-3-test" {
		t.Fatalf("unexpected model default: %s", cfg.ModelDefault)
	}
	if cfg.ModelHints == nil || *cfg.ModelHints != "hints" {
		t.Fatalf("unexpected hints model: %v", cfg.ModelHints)
	}
	if cfg.TransportMode == nil || *cfg.TransportMode != "h2c" {
		t.Fatalf("unexpected transport mode: %v", cfg.TransportMode)
	}
}

func TestClient_GRPC_EndSession(t *testing.T) {
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen failed: %v", err)
	}
	grpcServer := grpc.NewServer()
	llmv1.RegisterLLMServiceServer(grpcServer, &grpcTestService{t: t, apiKey: "test-key"})

	go func() {
		_ = grpcServer.Serve(lis)
	}()

	t.Cleanup(func() {
		grpcServer.Stop()
		_ = lis.Close()
	})

	client, err := New(Config{BaseURL: "grpc://" + lis.Addr().String(), APIKey: "test-key", Timeout: 2 * time.Second})
	if err != nil {
		t.Fatalf("failed to create grpc client: %v", err)
	}
	t.Cleanup(func() {
		_ = client.Close()
	})

	resp, err := client.EndSession(context.Background(), "twentyq:chat-1")
	if err != nil {
		t.Fatalf("EndSession failed: %v", err)
	}
	if resp.ID != "twentyq:chat-1" {
		t.Fatalf("unexpected session id: %s", resp.ID)
	}
	if resp.Message == "" {
		t.Fatalf("expected message")
	}
}

func TestClient_GRPC_Methods(t *testing.T) {
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen failed: %v", err)
	}

	grpcServer := grpc.NewServer()
	svc := &grpcTestService{apiKey: "test-key"}
	llmv1.RegisterLLMServiceServer(grpcServer, svc)

	go func() {
		_ = grpcServer.Serve(lis)
	}()

	t.Cleanup(func() {
		grpcServer.Stop()
		_ = lis.Close()
	})

	client, err := New(Config{BaseURL: "grpc://" + lis.Addr().String(), APIKey: "test-key", Timeout: 2 * time.Second})
	if err != nil {
		t.Fatalf("failed to create grpc client: %v", err)
	}
	t.Cleanup(func() {
		_ = client.Close()
	})

	t.Run("GuardIsMalicious", func(t *testing.T) {
		svc.t = t

		malicious, err := client.GuardIsMalicious(context.Background(), "malicious")
		if err != nil {
			t.Fatalf("GuardIsMalicious failed: %v", err)
		}
		if !malicious {
			t.Fatalf("expected malicious")
		}
	})

	t.Run("TwentyQSelectTopic", func(t *testing.T) {
		svc.t = t

		resp, err := client.TwentyQSelectTopic(context.Background(), "ANIMALS", nil, nil)
		if err != nil {
			t.Fatalf("TwentyQSelectTopic failed: %v", err)
		}
		if resp.Name == "" {
			t.Fatalf("expected name")
		}
		if resp.Category != "ANIMALS" {
			t.Fatalf("unexpected category: %s", resp.Category)
		}
		if resp.Details["source"] != "grpc" {
			t.Fatalf("unexpected details: %v", resp.Details)
		}
	})

	t.Run("TwentyQGenerateHints", func(t *testing.T) {
		svc.t = t

		resp, err := client.TwentyQGenerateHints(context.Background(), "cat", "ANIMALS", map[string]any{"foo": "bar"})
		if err != nil {
			t.Fatalf("TwentyQGenerateHints failed: %v", err)
		}
		if len(resp.Hints) != 2 {
			t.Fatalf("unexpected hints: %v", resp.Hints)
		}
	})

	t.Run("TwentyQAnswerQuestion", func(t *testing.T) {
		svc.t = t

		resp, err := client.TwentyQAnswerQuestion(context.Background(), "chat-1", "twentyq", "cat", "ANIMALS", "Q?", map[string]any{"foo": "bar"})
		if err != nil {
			t.Fatalf("TwentyQAnswerQuestion failed: %v", err)
		}
		if resp.Scale == nil || *resp.Scale != "YES" {
			t.Fatalf("unexpected scale: %v", resp.Scale)
		}
		if resp.RawText != "YES" {
			t.Fatalf("unexpected raw text: %s", resp.RawText)
		}
	})

	t.Run("TwentyQVerifyGuess", func(t *testing.T) {
		svc.t = t

		resp, err := client.TwentyQVerifyGuess(context.Background(), "cat", "cat")
		if err != nil {
			t.Fatalf("TwentyQVerifyGuess failed: %v", err)
		}
		if resp.Result == nil || *resp.Result != "CORRECT" {
			t.Fatalf("unexpected result: %v", resp.Result)
		}
	})

	t.Run("TurtleSoupGeneratePuzzle", func(t *testing.T) {
		svc.t = t

		category := "MYSTERY"
		difficulty := 3
		theme := "space"

		resp, err := client.TurtleSoupGeneratePuzzle(context.Background(), TurtleSoupPuzzleGenerationRequest{Category: &category, Difficulty: &difficulty, Theme: &theme})
		if err != nil {
			t.Fatalf("TurtleSoupGeneratePuzzle failed: %v", err)
		}
		if resp.Title == "" || resp.Scenario == "" || resp.Solution == "" {
			t.Fatalf("unexpected puzzle: %+v", resp)
		}
		if resp.Difficulty != 3 {
			t.Fatalf("unexpected difficulty: %d", resp.Difficulty)
		}
	})

	t.Run("TurtleSoupGetRandomPuzzle", func(t *testing.T) {
		svc.t = t

		difficulty := 2
		resp, err := client.TurtleSoupGetRandomPuzzle(context.Background(), &difficulty)
		if err != nil {
			t.Fatalf("TurtleSoupGetRandomPuzzle failed: %v", err)
		}
		if resp.ID == nil || *resp.ID != 10 {
			t.Fatalf("unexpected id: %v", resp.ID)
		}
		if resp.Difficulty == nil || *resp.Difficulty != 2 {
			t.Fatalf("unexpected difficulty: %v", resp.Difficulty)
		}
	})

	t.Run("TurtleSoupRewriteScenario", func(t *testing.T) {
		svc.t = t

		resp, err := client.TurtleSoupRewriteScenario(context.Background(), "title", "scenario", "solution", 3)
		if err != nil {
			t.Fatalf("TurtleSoupRewriteScenario failed: %v", err)
		}
		if resp.Scenario != "rewritten" {
			t.Fatalf("unexpected scenario: %s", resp.Scenario)
		}
		if resp.OriginalScenario != "scenario" {
			t.Fatalf("unexpected original scenario: %s", resp.OriginalScenario)
		}
	})

	t.Run("TurtleSoupAnswerQuestion", func(t *testing.T) {
		svc.t = t

		resp, err := client.TurtleSoupAnswerQuestion(context.Background(), "chat-1", "turtle-soup", "scenario", "solution", "Q?")
		if err != nil {
			t.Fatalf("TurtleSoupAnswerQuestion failed: %v", err)
		}
		if resp.Answer != "YES" {
			t.Fatalf("unexpected answer: %s", resp.Answer)
		}
		if len(resp.History) != 1 {
			t.Fatalf("unexpected history: %v", resp.History)
		}
	})

	t.Run("TurtleSoupValidateSolution", func(t *testing.T) {
		svc.t = t

		resp, err := client.TurtleSoupValidateSolution(context.Background(), "chat-1", "turtle-soup", "solution", "answer")
		if err != nil {
			t.Fatalf("TurtleSoupValidateSolution failed: %v", err)
		}
		if resp.Result != "YES" {
			t.Fatalf("unexpected result: %s", resp.Result)
		}
	})

	t.Run("TurtleSoupGenerateHint", func(t *testing.T) {
		svc.t = t

		resp, err := client.TurtleSoupGenerateHint(context.Background(), "chat-1", "turtle-soup", "scenario", "solution", 1)
		if err != nil {
			t.Fatalf("TurtleSoupGenerateHint failed: %v", err)
		}
		if resp.Hint != "HINT" {
			t.Fatalf("unexpected hint: %s", resp.Hint)
		}
	})

	t.Run("GetDailyUsage", func(t *testing.T) {
		svc.t = t

		usage, err := client.GetDailyUsage(context.Background(), nil)
		if err != nil {
			t.Fatalf("GetDailyUsage failed: %v", err)
		}
		if usage.Model == nil || *usage.Model != "gemini-3-test" {
			t.Fatalf("unexpected model: %v", usage.Model)
		}
	})

	t.Run("GetRecentUsage", func(t *testing.T) {
		svc.t = t

		usage, err := client.GetRecentUsage(context.Background(), 7, nil)
		if err != nil {
			t.Fatalf("GetRecentUsage failed: %v", err)
		}
		if usage.Model == nil || *usage.Model != "gemini-3-test" {
			t.Fatalf("unexpected model: %v", usage.Model)
		}
		if len(usage.Usages) != 1 {
			t.Fatalf("unexpected usages: %v", usage.Usages)
		}
	})

	t.Run("GetTotalUsage", func(t *testing.T) {
		svc.t = t

		usage, err := client.GetTotalUsage(context.Background(), nil)
		if err != nil {
			t.Fatalf("GetTotalUsage failed: %v", err)
		}
		if usage.InputTokens != 1 {
			t.Fatalf("unexpected input tokens: %d", usage.InputTokens)
		}
	})

	t.Run("GetUsageTotalFromDB", func(t *testing.T) {
		svc.t = t

		usage, err := client.GetUsageTotalFromDB(context.Background(), 30, nil)
		if err != nil {
			t.Fatalf("GetUsageTotalFromDB failed: %v", err)
		}
		if usage.InputTokens != 30 {
			t.Fatalf("unexpected input tokens: %d", usage.InputTokens)
		}
	})
}

func TestExtractRequestID(t *testing.T) {
	tests := []struct {
		name     string
		ctx      context.Context
		expected string
	}{
		{
			name:     "nil context",
			ctx:      nil,
			expected: "",
		},
		{
			name:     "empty context",
			ctx:      context.Background(),
			expected: "",
		},
		{
			name:     "with typed key",
			ctx:      WithRequestID(context.Background(), "typed-id-123"),
			expected: "typed-id-123",
		},
		{
			name:     "with string key",
			ctx:      context.WithValue(context.Background(), "request_id", "string-id-456"),
			expected: "string-id-456",
		},
		{
			name:     "typed key takes precedence",
			ctx:      WithRequestID(context.WithValue(context.Background(), "request_id", "string-id"), "typed-id"),
			expected: "typed-id",
		},
		{
			name:     "empty string value ignored",
			ctx:      context.WithValue(context.Background(), "request_id", ""),
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractRequestID(tt.ctx)
			if got != tt.expected {
				t.Errorf("extractRequestID() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestWithRequestID(t *testing.T) {
	ctx := context.Background()
	id := "test-request-id-789"

	newCtx := WithRequestID(ctx, id)

	// Context should contain the ID
	got := extractRequestID(newCtx)
	if got != id {
		t.Errorf("WithRequestID() then extractRequestID() = %q, want %q", got, id)
	}

	// Original context should be unchanged
	if extractRequestID(ctx) != "" {
		t.Error("original context should not be modified")
	}
}
