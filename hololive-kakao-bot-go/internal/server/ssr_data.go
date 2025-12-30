package server

import (
	"bytes"
	"context"
	"strings"

	"github.com/goccy/go-json"

	"github.com/kapu/hololive-kakao-bot-go/internal/constants"
	"github.com/kapu/hololive-kakao-bot-go/internal/service/docker"
	"github.com/kapu/hololive-kakao-bot-go/internal/service/member"
	"github.com/kapu/hololive-kakao-bot-go/internal/service/settings"
)

// SSRData: SSR을 위해 HTML에 주입할 데이터 구조체입니다.
// React 클라이언트에서 window.__SSR_DATA__로 접근합니다.
type SSRData struct {
	// 경로별 데이터 (TanStack Query의 queryKey와 매칭)
	Members    json.RawMessage `json:"members,omitempty"`
	Settings   json.RawMessage `json:"settings,omitempty"`
	Docker     json.RawMessage `json:"docker,omitempty"`
	Containers json.RawMessage `json:"containers,omitempty"`
}

// SSRDataInjector: SSR 데이터를 HTML에 주입하는 서비스입니다.
type SSRDataInjector struct {
	memberRepo  *member.Repository
	settingsSvc *settings.Service
	dockerSvc   *docker.Service
	htmlCache   []byte // 빌드된 index.html 캐시
}

// NewSSRDataInjector: 새로운 SSR 데이터 인젝터를 생성합니다.
func NewSSRDataInjector(
	memberRepo *member.Repository,
	settingsSvc *settings.Service,
	dockerSvc *docker.Service,
) *SSRDataInjector {
	return &SSRDataInjector{
		memberRepo:  memberRepo,
		settingsSvc: settingsSvc,
		dockerSvc:   dockerSvc,
	}
}

// SetHTMLCache: 빌드된 index.html을 캐시합니다.
func (s *SSRDataInjector) SetHTMLCache(html []byte) {
	s.htmlCache = html
}

// InjectForPath: 요청 경로에 맞는 SSR 데이터를 HTML에 주입합니다.
// 인증이 필요한 경로는 인증되지 않은 요청에 대해 빈 데이터를 반환합니다.
func (s *SSRDataInjector) InjectForPath(ctx context.Context, path string, isAuthenticated bool) ([]byte, error) {
	if len(s.htmlCache) == 0 {
		return nil, nil // HTML 캐시 없음
	}

	// 인증되지 않은 사용자에게는 SSR 데이터를 주입하지 않음
	if !isAuthenticated {
		return s.htmlCache, nil
	}

	// SSR 대상 경로 확인
	ssrData, err := s.fetchDataForPath(ctx, path)
	if err != nil {
		return s.htmlCache, nil // 에러 시 원본 HTML 반환 (클라이언트에서 fetch)
	}

	if ssrData == nil {
		return s.htmlCache, nil // 프리페칭 대상 경로 아님
	}

	return s.injectData(ssrData)
}

// fetchDataForPath: 경로에 맞는 데이터를 프리페칭합니다.
func (s *SSRDataInjector) fetchDataForPath(ctx context.Context, path string) (*SSRData, error) {
	// 타임아웃 컨텍스트 설정
	timeoutCtx, cancel := context.WithTimeout(ctx, constants.RequestTimeout.AdminRequest)
	defer cancel()

	ssrData := &SSRData{}
	hasData := false

	// /dashboard/members - 멤버 목록 프리페칭
	if strings.HasPrefix(path, "/dashboard/members") {
		members, err := s.memberRepo.GetAllMembers(timeoutCtx)
		if err == nil {
			data, _ := json.Marshal(map[string]any{
				"status":  "ok",
				"members": members,
			})
			ssrData.Members = data
			hasData = true
		}
	}

	// /dashboard/settings - 설정 및 Docker 컨테이너 프리페칭
	if strings.HasPrefix(path, "/dashboard/settings") {
		// 설정 데이터
		currentSettings := s.settingsSvc.Get()
		data, _ := json.Marshal(map[string]any{
			"status": "ok",
			"settings": map[string]any{
				"alarmAdvanceMinutes": currentSettings.AlarmAdvanceMinutes,
			},
		})
		ssrData.Settings = data
		hasData = true

		// Docker 상태 및 컨테이너 목록
		if s.dockerSvc != nil {
			available := s.dockerSvc.Available(timeoutCtx)
			dockerData, _ := json.Marshal(map[string]any{
				"status":    "ok",
				"available": available,
			})
			ssrData.Docker = dockerData

			if available {
				containers, err := s.dockerSvc.ListContainers(timeoutCtx)
				if err == nil {
					containerData, _ := json.Marshal(map[string]any{
						"status":     "ok",
						"containers": containers,
					})
					ssrData.Containers = containerData
				}
			}
		}
	}

	if !hasData {
		return nil, nil
	}

	return ssrData, nil
}

// injectData: HTML에 SSR 데이터를 주입합니다.
// </head> 태그 앞에 <script> 태그로 window.__SSR_DATA__를 설정합니다.
func (s *SSRDataInjector) injectData(data *SSRData) ([]byte, error) {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return s.htmlCache, nil
	}

	// 스크립트 주입: </head> 앞에 삽입
	script := []byte(`<script>window.__SSR_DATA__=` + string(jsonData) + `;</script>`)
	injectionPoint := []byte("</head>")

	return bytes.Replace(s.htmlCache, injectionPoint, append(script, injectionPoint...), 1), nil
}
