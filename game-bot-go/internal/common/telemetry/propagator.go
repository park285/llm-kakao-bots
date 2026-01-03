package telemetry

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
)

// InjectContext: context에서 trace context를 추출하여 carrier에 주입합니다.
// gRPC metadata나 HTTP headers로 trace context를 전파할 때 사용합니다.
func InjectContext(ctx context.Context, carrier propagation.TextMapCarrier) {
	otel.GetTextMapPropagator().Inject(ctx, carrier)
}

// ExtractContext: carrier에서 trace context를 추출하여 새 context를 반환합니다.
// 메시지 헤더나 HTTP headers에서 부모 trace context를 복원할 때 사용합니다.
func ExtractContext(ctx context.Context, carrier propagation.TextMapCarrier) context.Context {
	return otel.GetTextMapPropagator().Extract(ctx, carrier)
}

// MapCarrier: map[string]string을 TextMapCarrier로 사용할 수 있게 해주는 어댑터입니다.
// Valkey 메시지의 Values 필드를 직접 사용할 수 있습니다.
type MapCarrier map[string]string

// Get: 주어진 키의 값을 반환합니다.
func (c MapCarrier) Get(key string) string {
	return c[key]
}

// Set: 주어진 키에 값을 설정합니다.
func (c MapCarrier) Set(key, value string) {
	c[key] = value
}

// Keys: 모든 키를 반환합니다.
func (c MapCarrier) Keys() []string {
	keys := make([]string, 0, len(c))
	for k := range c {
		keys = append(keys, k)
	}
	return keys
}
