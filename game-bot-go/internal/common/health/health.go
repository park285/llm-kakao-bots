// Package health: 서비스 상태 정보
package health

import (
	"runtime"
	"sync"
	"time"
)

var (
	startTime time.Time
	version   = "dev"
	initOnce  sync.Once
)

// Init: 서비스 시작 시 호출 (버전 정보 설정)
func Init(v string) {
	initOnce.Do(func() {
		startTime = time.Now()
		if v != "" {
			version = v
		}
	})
}

// Response: /health 엔드포인트 표준 응답
type Response struct {
	Status     string `json:"status"`
	Version    string `json:"version"`
	Uptime     string `json:"uptime"`
	Goroutines int    `json:"goroutines"`
}

// Get: 현재 상태 반환
func Get() Response {
	return Response{
		Status:     "ok",
		Version:    version,
		Uptime:     formatDuration(time.Since(startTime)),
		Goroutines: runtime.NumGoroutine(),
	}
}

// formatDuration: Duration을 사람이 읽기 쉬운 형식으로 변환
func formatDuration(d time.Duration) string {
	d = d.Round(time.Second)
	h := d / time.Hour
	d -= h * time.Hour
	m := d / time.Minute
	d -= m * time.Minute
	s := d / time.Second

	if h > 0 {
		return time.Duration(h*time.Hour + m*time.Minute + s*time.Second).String()
	}
	if m > 0 {
		return time.Duration(m*time.Minute + s*time.Second).String()
	}
	return time.Duration(s * time.Second).String()
}
