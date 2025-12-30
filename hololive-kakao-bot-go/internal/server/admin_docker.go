package server

import (
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"

	"github.com/kapu/hololive-kakao-bot-go/internal/service/docker"
)

// DockerHandler: Docker 컨테이너 관리 요청을 처리하는 핸들러입니다.
type DockerHandler struct {
	docker *docker.Service
}

// NewDockerHandler: 새로운 Docker 핸들러를 생성합니다.
func NewDockerHandler(dockerSvc *docker.Service) *DockerHandler {
	return &DockerHandler{
		docker: dockerSvc,
	}
}

// GetHealth: Docker 데몬의 가용성을 확인합니다.
func (h *DockerHandler) GetHealth(c *gin.Context) {
	if h.docker == nil {
		c.JSON(http.StatusOK, gin.H{
			"status":    "unavailable",
			"available": false,
		})
		return
	}

	available := h.docker.Available(c.Request.Context())
	c.JSON(http.StatusOK, gin.H{
		"status":    statusString(available),
		"available": available,
	})
}

// GetContainers: 관리 대상 컨테이너 목록을 반환합니다.
func (h *DockerHandler) GetContainers(c *gin.Context) {
	if h.docker == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"status":  "error",
			"message": "Docker service not available",
		})
		return
	}

	containers, err := h.docker.ListContainers(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "error",
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":     "ok",
		"containers": containers,
	})
}

type containerActionRequest struct {
	Reason string `json:"reason"`
	Force  bool   `json:"force"`
}

// RestartContainer: 컨테이너를 재시작합니다.
func (h *DockerHandler) RestartContainer(c *gin.Context) {
	if h.docker == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"status":  "error",
			"message": "Docker service not available",
		})
		return
	}

	name := c.Param("name")
	if name == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": "container name is required",
		})
		return
	}
	if !h.docker.IsManaged(name) {
		c.JSON(http.StatusNotFound, gin.H{
			"status":  "error",
			"message": "container not found",
		})
		return
	}

	var req containerActionRequest
	_ = c.ShouldBindJSON(&req) // 선택적 바인딩

	if err := h.docker.RestartContainer(c.Request.Context(), name); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "error",
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "ok",
		"message": "Container restart initiated",
	})
}

// StopContainer: 컨테이너를 중지합니다.
func (h *DockerHandler) StopContainer(c *gin.Context) {
	if h.docker == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"status":  "error",
			"message": "Docker service not available",
		})
		return
	}

	name := c.Param("name")
	if name == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": "container name is required",
		})
		return
	}
	if !h.docker.IsManaged(name) {
		c.JSON(http.StatusNotFound, gin.H{
			"status":  "error",
			"message": "container not found",
		})
		return
	}

	if err := h.docker.StopContainer(c.Request.Context(), name); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "error",
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "ok",
		"message": "Container stopped",
	})
}

// StartContainer: 중지된 컨테이너를 시작합니다.
func (h *DockerHandler) StartContainer(c *gin.Context) {
	if h.docker == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"status":  "error",
			"message": "Docker service not available",
		})
		return
	}

	name := c.Param("name")
	if name == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": "container name is required",
		})
		return
	}
	if !h.docker.IsManaged(name) {
		c.JSON(http.StatusNotFound, gin.H{
			"status":  "error",
			"message": "container not found",
		})
		return
	}

	if err := h.docker.StartContainer(c.Request.Context(), name); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "error",
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "ok",
		"message": "Container started",
	})
}

func statusString(available bool) string {
	if available {
		return "ok"
	}
	return "unavailable"
}

// StreamLogs: WebSocket을 통해 컨테이너 로그를 실시간 스트리밍합니다.
// Docker 로그 스트림의 8-byte 헤더를 파싱하여 순수 로그 메시지만 전송합니다.
func (h *DockerHandler) StreamLogs(c *gin.Context) {
	if h.docker == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"status":  "error",
			"message": "Docker service not available",
		})
		return
	}

	name := c.Param("name")
	if name == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": "container name is required",
		})
		return
	}
	if !h.docker.IsManaged(name) {
		c.JSON(http.StatusNotFound, gin.H{
			"status":  "error",
			"message": "container not found",
		})
		return
	}

	// WebSocket 업그레이드
	conn, err := wsUpgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return // 업그레이드 실패 시 Upgrader가 자동으로 응답 처리
	}
	defer func() { _ = conn.Close() }()

	// Docker 로그 스트림 시작
	ctx := c.Request.Context()
	logReader, err := h.docker.GetLogStream(ctx, name)
	if err != nil {
		_ = conn.WriteJSON(gin.H{"error": err.Error()})
		return
	}
	defer func() { _ = logReader.Close() }()

	// 8-byte Docker 로그 헤더 버퍼
	header := make([]byte, 8)
	buf := make([]byte, 4096)
	const maxLogChunkSize = 1 << 20 // 1MiB

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		// Docker 로그 헤더 읽기 (8 bytes: [stream_type][0][0][0][size_be32])
		_, err := io.ReadFull(logReader, header)
		if err != nil {
			return
		}

		// 페이로드 크기 파싱 (big-endian uint32, bytes 4-7)
		size := int(header[4])<<24 | int(header[5])<<16 | int(header[6])<<8 | int(header[7])
		if size <= 0 {
			continue
		}
		if size > maxLogChunkSize {
			_, _ = io.CopyN(io.Discard, logReader, int64(size))
			continue
		}
		if size > cap(buf) {
			buf = make([]byte, size)
		}
		payload := buf[:size]

		// 실제 로그 메시지 읽기
		n, err := io.ReadFull(logReader, payload)
		if err != nil {
			return
		}

		// WebSocket으로 전송
		if err := conn.WriteMessage(websocket.TextMessage, payload[:n]); err != nil {
			return
		}
	}
}
