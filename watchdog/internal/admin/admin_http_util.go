package admin

import (
	"bytes"
	"io"

	"github.com/gin-gonic/gin"
)

func writeAPIError(c *gin.Context, status int, code string, message string) {
	c.AbortWithStatusJSON(status, gin.H{
		"error": gin.H{
			"code":    code,
			"message": message,
		},
	})
}

func ioReadAllLimit(r io.Reader, limit int64) ([]byte, error) {
	limited := io.LimitReader(r, limit)
	var buf bytes.Buffer
	if _, err := buf.ReadFrom(limited); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func getAdminEmail(c *gin.Context) string {
	value, ok := c.Get("admin_email")
	if !ok {
		return ""
	}
	email, _ := value.(string)
	return email
}

func noCacheHeaders(c *gin.Context) {
	c.Header("Cache-Control", "no-store")
	c.Header("Pragma", "no-cache")
	c.Header("Expires", "0")
}
