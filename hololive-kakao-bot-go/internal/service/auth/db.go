package auth

import (
	"errors"
	"strings"

	"github.com/lib/pq"
	"gorm.io/gorm"
)

func isDuplicateKeyError(err error) bool {
	if err == nil {
		return false
	}

	if errors.Is(err, gorm.ErrDuplicatedKey) {
		return true
	}

	var pqErr *pq.Error
	if errors.As(err, &pqErr) {
		// 23505: unique_violation
		return string(pqErr.Code) == "23505"
	}

	// sqlite 등 드라이버별 메시지 fallback
	msg := err.Error()
	if strings.Contains(msg, "UNIQUE constraint failed") {
		return true
	}
	if strings.Contains(msg, "duplicate key value violates unique constraint") {
		return true
	}

	return false
}
