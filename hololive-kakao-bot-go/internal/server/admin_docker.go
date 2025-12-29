package server

import (
	"net/http"

	"github.com/gin-gonic/gin"

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
