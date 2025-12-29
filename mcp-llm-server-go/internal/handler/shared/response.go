package shared

import (
	"errors"
	"io"

	"github.com/gin-gonic/gin"

	"github.com/park285/llm-kakao-bots/mcp-llm-server-go/internal/httperror"
	"github.com/park285/llm-kakao-bots/mcp-llm-server-go/internal/middleware"
)

// WriteError: 에러 응답을 작성합니다.
func WriteError(c *gin.Context, err error) {
	if c == nil {
		return
	}
	status, payload := httperror.Response(err, middleware.GetRequestID(c))
	c.JSON(status, payload)
}

// BindJSON: 요청 본문을 JSON으로 파싱합니다.
func BindJSON(c *gin.Context, out any) bool {
	if c == nil {
		return false
	}
	if err := c.ShouldBindJSON(out); err != nil {
		WriteError(c, httperror.NewValidationError(err))
		return false
	}
	return true
}

// BindJSONAllowEmpty: 빈 본문도 허용합니다.
func BindJSONAllowEmpty(c *gin.Context, out any) bool {
	if c == nil {
		return false
	}
	if err := c.ShouldBindJSON(out); err != nil {
		if errors.Is(err, io.EOF) {
			return true
		}
		WriteError(c, httperror.NewValidationError(err))
		return false
	}
	return true
}
