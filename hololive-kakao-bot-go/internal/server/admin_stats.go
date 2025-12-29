package server

import (
	"context"
	"sync"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/kapu/hololive-kakao-bot-go/internal/constants"
	"github.com/kapu/hololive-kakao-bot-go/internal/domain"
	"github.com/kapu/hololive-kakao-bot-go/internal/service/notification"
)

// GetStats: 봇 통계를 반환합니다. (성능 최적화를 위해 병렬 조회)
func (h *AdminHandler) GetStats(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), constants.RequestTimeout.AdminRequest)
	defer cancel()

	var (
		members   []*domain.Member
		alarmKeys []*notification.AlarmEntry
		wg        sync.WaitGroup
	)

	// 병렬로 데이터 조회
	wg.Add(2)
	go func() {
		defer wg.Done()
		members, _ = h.repo.GetAllMembers(ctx)
	}()
	go func() {
		defer wg.Done()
		alarmKeys, _ = h.alarm.GetAllAlarmKeys(ctx)
	}()
	wg.Wait()

	uptime := time.Since(h.startTime).Truncate(time.Second).String()

	// ACL 서비스에서 rooms 수 조회
	var roomCount int
	if h.acl != nil {
		_, rooms := h.acl.GetACLStatus()
		roomCount = len(rooms)
	}

	c.JSON(200, gin.H{
		"status":  "ok",
		"members": len(members),
		"alarms":  len(alarmKeys),
		"rooms":   roomCount,
		"version": "v1.1.0-go",
		"uptime":  uptime,
	})
}
