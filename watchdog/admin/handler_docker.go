package admin

import (
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"llm-watchdog/watchdog"

	"github.com/gin-gonic/gin"
)

// registerDockerHandlers Docker 및 이벤트 관련 핸들러 등록
func registerDockerHandlers(api *gin.RouterGroup, w *watchdog.Watchdog, logger *slog.Logger) {
	api.GET("/docker/containers", handleListContainers(w))
	api.GET("/events", handleListEvents(w))
	api.GET("/targets/:name/logs", handleContainerLogs(w, logger))
	api.GET("/targets/:name/logs/stream", handleContainerLogsStream(w, logger))
}

// handleListContainers Docker 컨테이너 목록 조회
func handleListContainers(w *watchdog.Watchdog) gin.HandlerFunc {
	return func(c *gin.Context) {
		containers, err := w.ListDockerContainers(c.Request.Context())
		if err != nil {
			writeAPIError(c, http.StatusServiceUnavailable, "docker_error", err.Error())
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"generatedAt": timeNow(),
			"containers":  containers,
		})
	}
}

// handleListEvents 이벤트 목록 조회
func handleListEvents(w *watchdog.Watchdog) gin.HandlerFunc {
	return func(c *gin.Context) {
		limit, _ := strconv.Atoi(strings.TrimSpace(c.Query("limit")))
		events := w.SnapshotEvents(limit)
		c.JSON(http.StatusOK, gin.H{
			"events": events,
		})
	}
}

// handleContainerLogs 컨테이너 로그 조회
func handleContainerLogs(w *watchdog.Watchdog, logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		name := watchdog.CanonicalContainerName(c.Param("name"))
		_, ok := w.GetState(name)
		if !ok {
			writeAPIError(c, http.StatusNotFound, "not_found", "관리 대상 컨테이너가 아닙니다.")
			return
		}

		tail, _ := strconv.Atoi(strings.TrimSpace(c.Query("tail")))
		if tail <= 0 {
			tail = 200
		}
		if tail > 2000 {
			tail = 2000
		}

		timestamps := strings.TrimSpace(c.Query("timestamps")) != "false"

		reader, isTTY, err := w.OpenDockerLogs(c.Request.Context(), name, tail, false, timestamps)
		if err != nil {
			writeAPIError(c, http.StatusServiceUnavailable, "docker_error", err.Error())
			return
		}
		defer func() {
			if err := reader.Close(); err != nil {
				logger.Warn("docker_logs_reader_close_failed", "err", err, "container", name, "admin_email", getAdminEmail(c))
			}
		}()

		reader = watchdog.WrapDockerLogsReader(reader, isTTY)
		body, err := ioReadAllLimit(reader, 5*1024*1024)
		if err != nil {
			writeAPIError(c, http.StatusServiceUnavailable, "docker_error", err.Error())
			return
		}
		c.Data(http.StatusOK, "text/plain; charset=utf-8", body)
	}
}

// handleContainerLogsStream 컨테이너 로그 스트리밍 (SSE)
func handleContainerLogsStream(w *watchdog.Watchdog, logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		name := watchdog.CanonicalContainerName(c.Param("name"))
		adminEmail := getAdminEmail(c)
		_, ok := w.GetState(name)
		if !ok {
			writeAPIError(c, http.StatusNotFound, "not_found", "관리 대상 컨테이너가 아닙니다.")
			return
		}

		tail, _ := strconv.Atoi(strings.TrimSpace(c.Query("tail")))
		if tail <= 0 {
			tail = 200
		}
		if tail > 2000 {
			tail = 2000
		}

		reader, isTTY, err := w.OpenDockerLogs(c.Request.Context(), name, tail, true, true)
		if err != nil {
			writeAPIError(c, http.StatusServiceUnavailable, "docker_error", err.Error())
			return
		}
		defer func() {
			if err := reader.Close(); err != nil {
				logger.Warn("docker_logs_reader_close_failed", "err", err, "container", name, "admin_email", adminEmail)
			}
		}()

		reader = watchdog.WrapDockerLogsReader(reader, isTTY)

		c.Header("Content-Type", "text/event-stream")
		c.Header("Cache-Control", "no-cache")
		c.Header("Connection", "keep-alive")
		c.Status(http.StatusOK)
		c.Writer.Flush()

		logger.Info("admin_action", "action", "logs_stream", "container", name, "admin_email", adminEmail, "tail", tail)

		_ = watchdog.StreamLines(c.Request.Context(), reader, func(line string) error {
			payload, _ := jsonMarshal(gin.H{
				"container": name,
				"log":       line,
				"at":        time.Now(),
			})
			_, err := writeSSE(c.Writer, payload)
			if err != nil {
				return err
			}
			c.Writer.Flush()
			return nil
		})
	}
}
