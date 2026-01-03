package telemetry

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// Provider: OpenTelemetry provider를 관리합니다.
type Provider struct {
	tracerProvider *sdktrace.TracerProvider
}

// NewProvider: TracerProvider를 초기화하고 글로벌로 설정합니다.
// cfg.Enabled가 false면 no-op Provider를 반환합니다.
func NewProvider(ctx context.Context, cfg Config) (*Provider, error) {
	if !cfg.Enabled {
		return &Provider{}, nil
	}

	// Resource: 서비스 메타데이터 정의
	// [주의] resource.Default()와 Merge하면 Schema URL 충돌이 발생할 수 있음
	// Default()는 최신 스키마(1.37)를 사용하지만 semconv는 1.26 사용
	res := resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.ServiceName(cfg.ServiceName),
		semconv.ServiceVersion(cfg.ServiceVersion),
		semconv.DeploymentEnvironment(cfg.Environment),
	)

	// OTLP Exporter 설정
	var exporterOpts []otlptracegrpc.Option
	exporterOpts = append(exporterOpts, otlptracegrpc.WithEndpoint(cfg.OTLPEndpoint))
	if cfg.OTLPInsecure {
		exporterOpts = append(exporterOpts,
			otlptracegrpc.WithDialOption(grpc.WithTransportCredentials(insecure.NewCredentials())),
		)
	}

	exporter, err := otlptracegrpc.New(ctx, exporterOpts...)
	if err != nil {
		return nil, fmt.Errorf("create exporter: %w", err)
	}

	// Sampler 설정
	// [중요] ParentBased로 감싸서 부모가 샘플링했으면 자식도 무조건 샘플링
	// 이렇게 하지 않으면 분산 추적에서 Trace가 끊길 수 있음
	var rootSampler sdktrace.Sampler
	if cfg.SampleRate >= 1.0 {
		rootSampler = sdktrace.AlwaysSample()
	} else if cfg.SampleRate <= 0 {
		rootSampler = sdktrace.NeverSample()
	} else {
		rootSampler = sdktrace.TraceIDRatioBased(cfg.SampleRate)
	}
	// ParentBased: 부모가 샘플링 결정을 했으면 그 결정을 따름
	// 부모가 없으면(Root Span) rootSampler 사용
	sampler := sdktrace.ParentBased(rootSampler)

	// TracerProvider 생성
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sampler),
	)

	// 글로벌 설정
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(
		propagation.NewCompositeTextMapPropagator(
			propagation.TraceContext{}, // W3C Trace Context
			propagation.Baggage{},      // W3C Baggage
		),
	)

	return &Provider{tracerProvider: tp}, nil
}

// Shutdown: TracerProvider를 정리합니다. 애플리케이션 종료 시 호출하세요.
// 버퍼에 남은 span들을 flush하여 데이터 유실을 방지합니다.
func (p *Provider) Shutdown(ctx context.Context) error {
	if p.tracerProvider == nil {
		return nil
	}
	return p.tracerProvider.Shutdown(ctx)
}

// IsEnabled: OpenTelemetry가 활성화되었는지 확인합니다.
func (p *Provider) IsEnabled() bool {
	return p.tracerProvider != nil
}
