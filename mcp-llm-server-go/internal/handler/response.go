package handler

import (
	"github.com/gin-gonic/gin"

	"github.com/park285/llm-kakao-bots/mcp-llm-server-go/internal/handler/shared"
)

// writeError: 에러 응답을 작성합니다 (shared.WriteError 위임).
func writeError(c *gin.Context, err error) {
	shared.WriteError(c, err)
}

// bindJSON: 요청 본문을 JSON으로 파싱합니다 (shared.BindJSON 위임).
func bindJSON(c *gin.Context, out any) bool {
	return shared.BindJSON(c, out)
}

// bindJSONAllowEmpty: 빈 본문도 허용합니다 (shared.BindJSONAllowEmpty 위임).
func bindJSONAllowEmpty(c *gin.Context, out any) bool {
	return shared.BindJSONAllowEmpty(c, out)
}
