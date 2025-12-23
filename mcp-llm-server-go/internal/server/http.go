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

// NewHTTPServer 는 HTTP 서버를 생성한다.
func NewHTTPServer(cfg *config.Config, router *gin.Engine) *http.Server {
	addr := fmt.Sprintf("%s:%d", cfg.HTTP.Host, cfg.HTTP.Port)
	server := &http.Server{
		Addr:              addr,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
	}

	if cfg.HTTP.HTTP2Enabled {
		server.Handler = h2c.NewHandler(router, &http2.Server{})
	}

	return server
}
