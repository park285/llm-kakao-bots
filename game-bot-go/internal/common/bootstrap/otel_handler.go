package bootstrap

import (
	"context"
	"log/slog"

	"go.opentelemetry.io/otel/trace"
)

// OTelHandler: slog.Handler를 래핑하여 trace_id/span_id를 자동으로 로그에 추가합니다.
// OpenTelemetry 분산 추적과 로그 상관관계를 제공합니다.
type OTelHandler struct {
	inner slog.Handler
}

// NewOTelHandler: OTel 상관관계가 추가된 slog.Handler를 생성합니다.
func NewOTelHandler(inner slog.Handler) *OTelHandler {
	return &OTelHandler{inner: inner}
}

// Enabled: 로그 레벨 활성화 여부를 확인합니다.
func (h *OTelHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.inner.Enabled(ctx, level)
}

// Handle: 로그 레코드에 trace_id/span_id를 추가하고 내부 핸들러로 전달합니다.
func (h *OTelHandler) Handle(ctx context.Context, record slog.Record) error {
	// Context에서 현재 Span 추출
	span := trace.SpanFromContext(ctx)
	if span.SpanContext().IsValid() {
		spanCtx := span.SpanContext()
		// trace_id와 span_id를 로그에 추가
		record.AddAttrs(
			slog.String("trace_id", spanCtx.TraceID().String()),
			slog.String("span_id", spanCtx.SpanID().String()),
		)
	}
	// NOTE: slog.Handler 인터페이스 구현이므로 에러 래핑하지 않음 (호환성)
	//nolint:wrapcheck // slog.Handler interface implementation
	return h.inner.Handle(ctx, record)
}

// WithAttrs: 속성을 추가한 새로운 Handler를 반환합니다.
func (h *OTelHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &OTelHandler{inner: h.inner.WithAttrs(attrs)}
}

// WithGroup: 그룹을 추가한 새로운 Handler를 반환합니다.
func (h *OTelHandler) WithGroup(name string) slog.Handler {
	return &OTelHandler{inner: h.inner.WithGroup(name)}
}
