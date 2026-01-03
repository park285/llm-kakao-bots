package llmrest

import (
	"context"
	"fmt"
	"net"
	"strings"
	"time"

	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"

	llmv1 "github.com/park285/llm-kakao-bots/game-bot-go/internal/common/llmrest/pb/llm/v1"
)

// Config: LLM 서버 통신 설정입니다.
// BaseURL 스킴은 grpc:// 또는 unix://이어야 합니다.
type Config struct {
	BaseURL        string
	APIKey         string
	Timeout        time.Duration
	ConnectTimeout time.Duration
	EnableOTel     bool // OpenTelemetry 계측 활성화
}

// Client: LLM 서버와 gRPC로 통신하기 위한 클라이언트입니다.
type Client struct {
	grpcConn    *grpc.ClientConn
	grpcClient  llmv1.LLMServiceClient
	grpcTimeout time.Duration
	apiKey      string
}

// New: 새로운 Client 인스턴스를 생성하고 초기화합니다.
// BaseURL 스킴은 grpc:// (TCP) 또는 unix:// (UDS)이어야 합니다.
func New(cfg Config) (*Client, error) {
	baseURL := strings.TrimSpace(cfg.BaseURL)
	if baseURL == "" {
		return nil, fmt.Errorf("base url is required")
	}

	const grpcMaxMsgSizeBytes = 16 * 1024 * 1024

	apiKey := strings.TrimSpace(cfg.APIKey)
	interceptor := func(
		ctx context.Context,
		method string,
		req, reply any,
		cc *grpc.ClientConn,
		invoker grpc.UnaryInvoker,
		opts ...grpc.CallOption,
	) error {
		if apiKey != "" {
			ctx = metadata.AppendToOutgoingContext(ctx, "x-api-key", apiKey)
		}
		// Context에서 Request ID를 추출하여 메타데이터로 전파
		if reqID := extractRequestID(ctx); reqID != "" {
			ctx = metadata.AppendToOutgoingContext(ctx, "x-request-id", reqID)
		}
		return invoker(ctx, method, req, reply, cc, opts...)
	}

	baseOpts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithUnaryInterceptor(interceptor),
		grpc.WithDefaultCallOptions(
			grpc.MaxCallRecvMsgSize(grpcMaxMsgSizeBytes),
			grpc.MaxCallSendMsgSize(grpcMaxMsgSizeBytes),
		),
	}

	// OTel StatsHandler: TraceContext 자동 전파
	if cfg.EnableOTel {
		baseOpts = append(baseOpts, grpc.WithStatsHandler(otelgrpc.NewClientHandler()))
	}

	var target string
	var dialOpts []grpc.DialOption

	lowerURL := strings.ToLower(baseURL)

	switch {
	case strings.HasPrefix(lowerURL, "unix://"):
		// UDS 모드: unix:///var/run/grpc/llm.sock
		socketPath := strings.TrimPrefix(baseURL, "unix://")
		socketPath = strings.TrimPrefix(socketPath, "UNIX://")
		if socketPath == "" {
			return nil, fmt.Errorf("invalid unix url: socket path is empty")
		}

		// passthrough 스킴 사용: gRPC가 주소를 그대로 사용함
		target = "passthrough:///" + socketPath
		dialOpts = baseOpts
		dialOpts = append(dialOpts, grpc.WithContextDialer(func(ctx context.Context, addr string) (net.Conn, error) {
			var d net.Dialer
			return d.DialContext(ctx, "unix", socketPath)
		}))

	case strings.HasPrefix(lowerURL, "grpc://"):
		// TCP 모드: grpc://mcp-llm-server:40528
		host := strings.TrimPrefix(baseURL, "grpc://")
		host = strings.TrimPrefix(host, "GRPC://")
		host = strings.TrimSuffix(host, "/")

		if host == "" {
			return nil, fmt.Errorf("invalid grpc url: host is empty")
		}

		// 포트가 없으면 기본 포트 추가함
		const defaultGRPCPort = "40528"
		if !strings.Contains(host, ":") {
			host = net.JoinHostPort(host, defaultGRPCPort)
		}

		target = host
		dialOpts = baseOpts

	default:
		return nil, fmt.Errorf("unsupported scheme: base url must start with grpc:// or unix://, got %q", baseURL)
	}

	conn, err := grpc.NewClient(target, dialOpts...)
	if err != nil {
		return nil, fmt.Errorf("create grpc client failed: %w", err)
	}

	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = 30 * time.Second
	}

	return &Client{
		grpcConn:    conn,
		grpcClient:  llmv1.NewLLMServiceClient(conn),
		grpcTimeout: timeout,
		apiKey:      apiKey,
	}, nil
}

func (c *Client) grpcCallContext(ctx context.Context) (context.Context, context.CancelFunc) {
	if c == nil || c.grpcTimeout <= 0 {
		return ctx, func() {}
	}
	if _, ok := ctx.Deadline(); ok {
		return ctx, func() {}
	}
	return context.WithTimeout(ctx, c.grpcTimeout)
}

// Close: gRPC 연결을 정리합니다.
func (c *Client) Close() error {
	if c == nil || c.grpcConn == nil {
		return nil
	}
	if err := c.grpcConn.Close(); err != nil {
		return fmt.Errorf("grpc conn close failed: %w", err)
	}
	return nil
}

// requestIDKey: Context에서 Request ID를 저장하는 키 타입
type requestIDKey struct{}

// extractRequestID: Context에서 Request ID를 추출합니다.
// 여러 키 형식(문자열, 별도 타입)을 순차적으로 확인합니다.
func extractRequestID(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	// 1. 별도 타입 키 확인 (권장 방식)
	if v, ok := ctx.Value(requestIDKey{}).(string); ok && v != "" {
		return v
	}
	// 2. 문자열 키 확인 (호환성)
	if v, ok := ctx.Value("request_id").(string); ok && v != "" {
		return v
	}
	return ""
}

// WithRequestID: Context에 Request ID를 추가합니다.
func WithRequestID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, requestIDKey{}, id)
}
