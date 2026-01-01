package grpcserver

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"os"
	"strings"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"github.com/park285/llm-kakao-bots/mcp-llm-server-go/internal/config"
)

const (
	defaultHost = "127.0.0.1"
	defaultPort = 40528

	requestIDKey = "request_id"

	maxRecvMsgSizeBytes = 16 * 1024 * 1024
)

type ctxKey string

// NewServer: gRPC 서버와 listener들을 생성합니다.
// TCP listener는 항상 생성되며, UDS listener는 SocketPath가 설정된 경우에만 생성됩니다.
func NewServer(cfg *config.Config, logger *slog.Logger) (*grpc.Server, net.Listener, net.Listener, error) {
	host := defaultHost
	port := defaultPort
	enabled := true
	apiKey := ""
	apiKeyRequired := false
	socketPath := ""

	if cfg != nil {
		host = strings.TrimSpace(cfg.GRPC.Host)
		port = cfg.GRPC.Port
		enabled = cfg.GRPC.Enabled
		socketPath = strings.TrimSpace(cfg.GRPC.SocketPath)

		apiKey = strings.TrimSpace(cfg.HTTPAuth.APIKey)
		apiKeyRequired = cfg.HTTPAuth.Required
	}
	if !enabled {
		return nil, nil, nil, nil
	}
	if host == "" {
		host = defaultHost
	}
	if port <= 0 {
		port = defaultPort
	}

	// TCP listener 생성
	addr := fmt.Sprintf("%s:%d", host, port)
	var lc net.ListenConfig
	listenCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	tcpLis, err := lc.Listen(listenCtx, "tcp", addr)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("tcp listen: %w", err)
	}

	// UDS listener 생성 (설정된 경우에만)
	var udsLis net.Listener
	if socketPath != "" {
		// 기존 소켓 파일이 있으면 삭제함 (비정상 종료 시 잔존 파일 처리)
		if err := os.Remove(socketPath); err != nil && !os.IsNotExist(err) {
			_ = tcpLis.Close()
			return nil, nil, nil, fmt.Errorf("remove existing socket: %w", err)
		}

		udsLis, err = lc.Listen(listenCtx, "unix", socketPath)
		if err != nil {
			_ = tcpLis.Close()
			return nil, nil, nil, fmt.Errorf("unix listen: %w", err)
		}

		if logger != nil {
			logger.Info("grpc_uds_enabled", "socket_path", socketPath)
		}
	}

	server := grpc.NewServer(
		grpc.MaxRecvMsgSize(maxRecvMsgSizeBytes),
		grpc.ChainUnaryInterceptor(
			unaryInterceptor(logger, apiKey, apiKeyRequired),
			errorMapperInterceptor(),
		),
	)
	return server, tcpLis, udsLis, nil
}

func unaryInterceptor(logger *slog.Logger, apiKey string, apiKeyRequired bool) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req any,
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (any, error) {
		start := time.Now()

		requestID := resolveRequestID(ctx)
		ctx = context.WithValue(ctx, ctxKey(requestIDKey), requestID)
		_ = grpc.SetHeader(ctx, metadata.Pairs("x-request-id", requestID))

		if err := authorize(ctx, apiKey, apiKeyRequired); err != nil {
			logGRPCRequest(logger, info, requestID, time.Since(start), err)
			return nil, err
		}

		resp, err := handler(ctx, req)
		logGRPCRequest(logger, info, requestID, time.Since(start), err)
		return resp, err
	}
}

func logGRPCRequest(logger *slog.Logger, info *grpc.UnaryServerInfo, requestID string, latency time.Duration, err error) {
	if logger == nil {
		return
	}

	method := ""
	if info != nil {
		method = info.FullMethod
	}

	fields := []any{
		"request_id", requestID,
		"method", method,
		"latency", latency,
	}
	if err != nil {
		fields = append(fields, "err", err)
		logger.Warn("grpc_request_failed", fields...)
		return
	}
	logger.Debug("grpc_request", fields...)
}

func authorize(ctx context.Context, expected string, required bool) error {
	if expected == "" {
		if required {
			return status.Error(codes.Internal, "http api key required but not configured")
		}
		return nil
	}

	provided := extractAPIKey(ctx)
	if provided == "" || subtle.ConstantTimeCompare([]byte(provided), []byte(expected)) != 1 {
		return status.Error(codes.Unauthenticated, "invalid api key")
	}
	return nil
}

func extractAPIKey(ctx context.Context) string {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return ""
	}
	if values := md.Get("x-api-key"); len(values) > 0 {
		if value := strings.TrimSpace(values[0]); value != "" {
			return value
		}
	}

	if values := md.Get("authorization"); len(values) > 0 {
		value := strings.TrimSpace(values[0])
		if strings.HasPrefix(strings.ToLower(value), "bearer ") {
			token := strings.TrimSpace(value[7:])
			return token
		}
	}
	return ""
}

func resolveRequestID(ctx context.Context) string {
	if ctx == nil {
		return ""
	}

	md, ok := metadata.FromIncomingContext(ctx)
	if ok {
		if values := md.Get("x-request-id"); len(values) > 0 {
			if value := strings.TrimSpace(values[0]); value != "" {
				return value
			}
		}
	}

	requestID, err := generateRequestID()
	if err != nil {
		return ""
	}
	return requestID
}

func generateRequestID() (string, error) {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("rand: %w", err)
	}
	return hex.EncodeToString(bytes), nil
}

// RequestIDFromContext: gRPC 컨텍스트에서 request_id를 조회합니다.
func RequestIDFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	value := ctx.Value(ctxKey(requestIDKey))
	if value == nil {
		return ""
	}
	requestID, ok := value.(string)
	if !ok {
		return ""
	}
	return requestID
}

// IsDisabledError: gRPC 미들웨어에서 반환한 비활성화 상태인지 확인합니다.
func IsDisabledError(err error) bool {
	if err == nil {
		return false
	}
	return errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded)
}
