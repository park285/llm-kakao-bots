package shared

import (
	"fmt"

	"github.com/mitchellh/mapstructure"
)

// DecoderConfig: mapstructure 디코더의 기본 설정입니다.
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

// Decode: map[string]any를 Go struct로 디코딩합니다.
// 타입 변환 실패 시 에러를 반환하며, 런타임 패닉을 방지합니다.
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

// DecodeStrict: Decode와 동일하지만 알 수 없는 필드가 있으면 에러를 반환합니다.
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
