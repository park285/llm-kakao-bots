package server

import (
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/park285/llm-kakao-bots/mcp-llm-server-go/internal/config"
)

func TestNewHTTPServer(t *testing.T) {
	router := gin.New()
	cfg := &config.Config{HTTP: config.HTTPConfig{Host: "127.0.0.1", Port: 8080, HTTP2Enabled: false}}

	server := NewHTTPServer(cfg, router)
	if server.Addr != "127.0.0.1:8080" {
		t.Fatalf("unexpected addr: %s", server.Addr)
	}
	if server.Handler != router {
		t.Fatalf("expected plain router handler")
	}

	cfg.HTTP.HTTP2Enabled = true
	server = NewHTTPServer(cfg, router)
	if server.Handler == router {
		t.Fatalf("expected wrapped handler")
	}
}
