// Package docker: Docker 컨테이너 관리 기능을 제공합니다.
package docker

import (
	"context"
	"fmt"
	"log/slog"
	"sort"
	"strings"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
)

// Container: Docker 컨테이너의 상태 및 메타데이터를 담는 구조체입니다.
type Container struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Image     string    `json:"image"`
	State     string    `json:"state"`
	Status    string    `json:"status"`
	Health    string    `json:"health"`
	Managed   bool      `json:"managed"`
	Paused    bool      `json:"paused"`
	StartedAt time.Time `json:"startedAt,omitempty"`
}

// Service: Docker 컨테이너 조작(목록 조회, 시작/중지/재시작)을 담당하는 서비스입니다.
type Service struct {
	client         *client.Client
	logger         *slog.Logger
	projectName    string
	managedFilters []string
}

// NewService: Docker 클라이언트를 초기화하고 새로운 Service 인스턴스를 생성합니다.
func NewService(logger *slog.Logger, projectName string) (*Service, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, fmt.Errorf("failed to create Docker client: %w", err)
	}

	return &Service{
		client:      cli,
		logger:      logger.With(slog.String("component", "docker")),
		projectName: projectName,
		managedFilters: []string{
			"hololive",
			"mcp-llm",
			"twentyq",
			"turtle-soup",
			"valkey",
			"postgres",
		},
	}, nil
}

// Available: Docker 데몬과의 연결 상태를 확인합니다.
func (s *Service) Available(ctx context.Context) bool {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	_, err := s.client.Ping(ctx)
	return err == nil
}

// ListContainers: 관리 대상 프로젝트에 속하는 모든 컨테이너 목록을 반환합니다.
func (s *Service) ListContainers(ctx context.Context) ([]Container, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	containers, err := s.client.ContainerList(ctx, container.ListOptions{All: true})
	if err != nil {
		return nil, fmt.Errorf("failed to list containers: %w", err)
	}

	result := make([]Container, 0, len(containers))
	for i := range containers {
		c := &containers[i]
		name := strings.TrimPrefix(c.Names[0], "/")

		// 관리 대상 컨테이너만 포함
		if !s.isManaged(name) {
			continue
		}

		health := "none"
		if c.State == "running" && strings.Contains(c.Status, "(") {
			// "(healthy)", "(unhealthy)", "(starting)" 추출
			start := strings.Index(c.Status, "(")
			end := strings.Index(c.Status, ")")
			if start != -1 && end != -1 && end > start {
				health = c.Status[start+1 : end]
			}
		}

		result = append(result, Container{
			ID:      c.ID[:12],
			Name:    name,
			Image:   c.Image,
			State:   c.State,
			Status:  c.Status,
			Health:  health,
			Managed: s.isManaged(name),
			Paused:  c.State == "paused",
		})
	}

	// 이름순 정렬
	sort.Slice(result, func(i, j int) bool {
		return result[i].Name < result[j].Name
	})

	return result, nil
}

// RestartContainer: 지정된 이름의 컨테이너를 재시작합니다.
func (s *Service) RestartContainer(ctx context.Context, name string) error {
	s.logger.Info("restarting container", slog.String("container", name))

	timeout := 30
	stopOptions := container.StopOptions{Timeout: &timeout}

	if err := s.client.ContainerRestart(ctx, name, stopOptions); err != nil {
		s.logger.Error("failed to restart container",
			slog.String("container", name),
			slog.String("error", err.Error()))
		return fmt.Errorf("failed to restart container %s: %w", name, err)
	}

	s.logger.Info("container restarted successfully", slog.String("container", name))
	return nil
}

// StopContainer: 지정된 이름의 컨테이너를 중지합니다.
func (s *Service) StopContainer(ctx context.Context, name string) error {
	s.logger.Info("stopping container", slog.String("container", name))

	timeout := 30
	stopOptions := container.StopOptions{Timeout: &timeout}

	if err := s.client.ContainerStop(ctx, name, stopOptions); err != nil {
		s.logger.Error("failed to stop container",
			slog.String("container", name),
			slog.String("error", err.Error()))
		return fmt.Errorf("failed to stop container %s: %w", name, err)
	}

	s.logger.Info("container stopped successfully", slog.String("container", name))
	return nil
}

// StartContainer: 중지된 컨테이너를 시작합니다.
func (s *Service) StartContainer(ctx context.Context, name string) error {
	s.logger.Info("starting container", slog.String("container", name))

	if err := s.client.ContainerStart(ctx, name, container.StartOptions{}); err != nil {
		s.logger.Error("failed to start container",
			slog.String("container", name),
			slog.String("error", err.Error()))
		return fmt.Errorf("failed to start container %s: %w", name, err)
	}

	s.logger.Info("container started successfully", slog.String("container", name))
	return nil
}

// isManaged: 컨테이너가 이 서비스의 관리 대상인지 확인함
func (s *Service) isManaged(name string) bool {
	for _, filter := range s.managedFilters {
		if strings.Contains(name, filter) {
			return true
		}
	}
	return false
}

// Close: Docker 클라이언트 연결을 종료합니다.
func (s *Service) Close() error {
	if err := s.client.Close(); err != nil {
		return fmt.Errorf("close docker client: %w", err)
	}
	return nil
}
