package server

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"

	"github.com/park285/llm-kakao-bots/mcp-llm-server-go/internal/config"
)

// NewHTTPServer: HTTP 서버를 생성합니다.
func NewHTTPServer(cfg *config.Config, router *gin.Engine) *http.Server {
	host := "127.0.0.1"
	port := 40527
	http2Enabled := true
	if cfg != nil {
		host = cfg.HTTP.Host
		port = cfg.HTTP.Port
		http2Enabled = cfg.HTTP.HTTP2Enabled
	}

	addr := fmt.Sprintf("%s:%d", host, port)
	readTimeout := 15 * time.Second
	writeTimeout := 60 * time.Second
	if cfg != nil && cfg.Gemini.TimeoutSeconds > 0 {
		writeTimeout = time.Duration(cfg.Gemini.TimeoutSeconds+15) * time.Second
		if writeTimeout < 60*time.Second {
			writeTimeout = 60 * time.Second
		}
	}

	server := &http.Server{
		Addr:              addr,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       readTimeout,
		WriteTimeout:      writeTimeout,
		IdleTimeout:       90 * time.Second,
		MaxHeaderBytes:    1 << 20, // 1MiB
	}

	if http2Enabled {
		server.Handler = h2c.NewHandler(router, &http2.Server{})
	}

	return server
}
