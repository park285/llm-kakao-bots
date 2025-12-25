package shared

import (
	"fmt"

	"github.com/mitchellh/mapstructure"
)

// DecoderConfig 는 mapstructure 디코더의 기본 설정이다.
func DecoderConfig(result any) *mapstructure.DecoderConfig {
	return &mapstructure.DecoderConfig{
		Result:           result,
		TagName:          "json",
		WeaklyTypedInput: true,
		DecodeHook: mapstructure.ComposeDecodeHookFunc(
			mapstructure.StringToTimeDurationHookFunc(),
		),
	}
}

// Decode 는 map[string]any를 Go struct로 디코딩한다.
// 타입 변환 실패 시 에러를 반환하며, 런타임 패닉을 방지한다.
func Decode(input map[string]any, result any) error {
	decoder, err := mapstructure.NewDecoder(DecoderConfig(result))
	if err != nil {
		return fmt.Errorf("new decoder: %w", err)
	}
	if err := decoder.Decode(input); err != nil {
		return fmt.Errorf("decode: %w", err)
	}
	return nil
}

// DecodeStrict 는 Decode와 동일하지만 알 수 없는 필드가 있으면 에러를 반환한다.
func DecodeStrict(input map[string]any, result any) error {
	cfg := DecoderConfig(result)
	cfg.ErrorUnused = true
	decoder, err := mapstructure.NewDecoder(cfg)
	if err != nil {
		return fmt.Errorf("new decoder: %w", err)
	}
	if err := decoder.Decode(input); err != nil {
		return fmt.Errorf("decode: %w", err)
	}
	return nil
}
