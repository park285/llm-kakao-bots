package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/park285/llm-kakao-bots/mcp-llm-server-go/internal/config"
	"github.com/park285/llm-kakao-bots/mcp-llm-server-go/internal/health"
)

// ModelConfigResponse 는 모델 설정 응답이다.
type ModelConfigResponse struct {
	ModelDefault          string  `json:"model_default"`
	ModelHints            string  `json:"model_hints"`
	ModelAnswer           string  `json:"model_answer"`
	ModelVerify           string  `json:"model_verify"`
	Temperature           float64 `json:"temperature"`
	ConfiguredTemperature float64 `json:"configured_temperature"`
	TimeoutSeconds        int     `json:"timeout_seconds"`
	MaxRetries            int     `json:"max_retries"`
	HTTP2Enabled          bool    `json:"http2_enabled"`
	TransportMode         string  `json:"transport_mode"`
}

// RegisterHealthRoutes 는 상태 확인 라우트를 등록한다.
func RegisterHealthRoutes(router *gin.Engine, cfg *config.Config) {
	router.GET("/health", func(c *gin.Context) {
		payload := health.Collect(c.Request.Context(), cfg, true)
		c.JSON(http.StatusOK, payload)
	})

	router.GET("/health/ready", func(c *gin.Context) {
		payload := health.Collect(c.Request.Context(), cfg, true)
		status := http.StatusOK
		if payload.Status != "ok" {
			status = http.StatusServiceUnavailable
		}
		c.JSON(status, payload)
	})

	router.GET("/health/models", func(c *gin.Context) {
		defaultModel := cfg.Gemini.DefaultModel
		hintsModel := cfg.Gemini.HintsModel
		answerModel := cfg.Gemini.AnswerModel
		verifyModel := cfg.Gemini.VerifyModel

		if hintsModel == "" {
			hintsModel = defaultModel
		}
		if answerModel == "" {
			answerModel = defaultModel
		}
		if verifyModel == "" {
			verifyModel = defaultModel
		}

		transportMode := "h1"
		if cfg.HTTP.HTTP2Enabled {
			transportMode = "h2c"
		}

		response := ModelConfigResponse{
			ModelDefault:          defaultModel,
			ModelHints:            hintsModel,
			ModelAnswer:           answerModel,
			ModelVerify:           verifyModel,
			Temperature:           cfg.Gemini.TemperatureForModel(defaultModel),
			ConfiguredTemperature: cfg.Gemini.Temperature,
			TimeoutSeconds:        cfg.Gemini.TimeoutSeconds,
			MaxRetries:            cfg.Gemini.MaxRetries,
			HTTP2Enabled:          cfg.HTTP.HTTP2Enabled,
			TransportMode:         transportMode,
		}

		c.JSON(http.StatusOK, response)
	})
}
