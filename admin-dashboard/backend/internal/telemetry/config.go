// Package telemetry: OpenTelemetry 기반 분산 추적 기능을 제공합니다.
package telemetry

// Config: OpenTelemetry 설정입니다.
type Config struct {
	// Enabled: true면 트레이싱을 활성화합니다.
	Enabled bool

	// ServiceName: 서비스 식별자입니다 (예: "admin-dashboard").
	ServiceName string

	// ServiceVersion: 서비스 버전입니다 (예: "1.0.0").
	ServiceVersion string

	// Environment: 배포 환경입니다 (예: "production", "development").
	Environment string

	// OTLPEndpoint: OTLP collector/exporter 주소입니다.
	// 예: "jaeger:4317" (gRPC) 또는 "jaeger:4318" (HTTP)
	OTLPEndpoint string

	// OTLPInsecure: true면 TLS 없이 연결합니다. 내부망에서만 사용하세요.
	OTLPInsecure bool

	// SampleRate: 샘플링 비율입니다 (0.0 ~ 1.0). 1.0이면 전체 트레이싱.
	// 프로덕션에서는 0.1 ~ 0.5 권장.
	SampleRate float64
}

// DefaultConfig: 기본 설정을 반환합니다. 기본값은 비활성화 상태입니다.
func DefaultConfig() Config {
	return Config{
		Enabled:      false,
		SampleRate:   1.0,
		OTLPInsecure: true,
	}
}
