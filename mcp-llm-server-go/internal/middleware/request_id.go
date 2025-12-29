package middleware

import (
	"crypto/rand"
	"encoding/hex"

	"github.com/gin-gonic/gin"
)

// RequestIDHeader 는 요청 ID 헤더 키다.
const RequestIDHeader = "X-Request-ID"

const requestIDKey = "request_id"

// RequestID 는 요청 ID를 부여하는 미들웨어다.
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := c.GetHeader(RequestIDHeader)
		if requestID == "" {
			requestID = generateRequestID()
		}
		c.Set(requestIDKey, requestID)

		c.Next()

		c.Header(RequestIDHeader, requestID)
	}
}

// GetRequestID: 컨텍스트의 요청 ID를 반환합니다.
func GetRequestID(c *gin.Context) string {
	if c == nil {
		return ""
	}
	value, ok := c.Get(requestIDKey)
	if !ok {
		return ""
	}
	requestID, ok := value.(string)
	if !ok {
		return ""
	}
	return requestID
}

func generateRequestID() string {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return ""
	}
	return hex.EncodeToString(bytes)
}
