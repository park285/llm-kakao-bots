package httpserver

import (
	"net/http"
	"time"
)

// ServerOptions: HTTP 서버 설정 옵션
type ServerOptions struct {
	UseH2C            bool
	ReadHeaderTimeout time.Duration
	IdleTimeout       time.Duration
	MaxHeaderBytes    int
}

// NewServer: 옵션에 따라 구성된 새로운 HTTP 서버 인스턴스를 생성한다.
func NewServer(addr string, handler http.Handler, opts ServerOptions) *http.Server {
	if handler == nil {
		handler = http.NewServeMux()
	}

	finalHandler := handler
	if opts.UseH2C {
		finalHandler = WrapH2C(handler)
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
