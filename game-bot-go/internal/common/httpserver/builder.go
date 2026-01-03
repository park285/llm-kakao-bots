package httpserver

import (
	"net/http"
	"time"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

// ServerOptions: HTTP 서버 설정 옵션
type ServerOptions struct {
	UseH2C            bool
	ReadHeaderTimeout time.Duration
	IdleTimeout       time.Duration
	MaxHeaderBytes    int
	// OpenTelemetry 설정
	EnableOTel      bool   // OTel 미들웨어 활성화
	OTelServiceName string // 서비스 이름 (Jaeger에 표시됨)
}

// NewServer: 옵션에 따라 구성된 새로운 HTTP 서버 인스턴스를 생성합니다.
func NewServer(addr string, handler http.Handler, opts ServerOptions) *http.Server {
	if handler == nil {
		handler = http.NewServeMux()
	}

	finalHandler := handler

	// OTel HTTP 미들웨어: 활성화된 경우 모든 HTTP 요청을 추적함
	if opts.EnableOTel && opts.OTelServiceName != "" {
		finalHandler = otelhttp.NewHandler(finalHandler, opts.OTelServiceName)
	}

	if opts.UseH2C {
		finalHandler = WrapH2C(finalHandler)
	}

	readHeaderTimeout := opts.ReadHeaderTimeout
	if readHeaderTimeout <= 0 {
		readHeaderTimeout = 5 * time.Second
	}

	server := &http.Server{
		Addr:              addr,
		Handler:           finalHandler,
		ReadHeaderTimeout: readHeaderTimeout,
	}
	if opts.IdleTimeout > 0 {
		server.IdleTimeout = opts.IdleTimeout
	}
	if opts.MaxHeaderBytes > 0 {
		server.MaxHeaderBytes = opts.MaxHeaderBytes
	}

	return server
}
