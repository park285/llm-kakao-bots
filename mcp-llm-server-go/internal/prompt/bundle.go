package prompt

import (
	"fmt"
	"io/fs"
)

// Bundle: 특정 도메인의 프롬프트 모음과 에러 메시지 라벨을 함께 관리합니다.
type Bundle struct {
	label   string
	prompts map[string]map[string]string
}

// LoadBundle: fs 내 dir 디렉터리의 YAML 프롬프트들을 로드하여 Bundle로 반환합니다.
func LoadBundle(fsys fs.FS, dir string, label string) (*Bundle, error) {
	loaded, err := LoadYAMLDir(fsys, dir)
	if err != nil {
		return nil, err
	}
	return &Bundle{label: label, prompts: loaded}, nil
}

// Prompt: 이름으로 프롬프트 맵을 조회합니다.
func (b *Bundle) Prompt(name string) (map[string]string, error) {
	if b == nil {
		return nil, fmt.Errorf("prompts not initialized")
	}
	return Get(b.prompts, name, b.label)
}

// Field: 프롬프트 맵에서 필요한 필드를 조회합니다.
func (b *Bundle) Field(data map[string]string, key string, label string) (string, error) {
	return Field(data, key, label)
}
