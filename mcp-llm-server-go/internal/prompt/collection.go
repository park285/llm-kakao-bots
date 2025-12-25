package prompt

import "fmt"

// Get 은 로드된 프롬프트 모음에서 프롬프트 맵을 가져온다.
func Get(prompts map[string]map[string]string, name string, label string) (map[string]string, error) {
	if prompts == nil {
		if label == "" {
			return nil, fmt.Errorf("prompts not initialized")
		}
		return nil, fmt.Errorf("%s prompts not initialized", label)
	}
	promptMap, ok := prompts[name]
	if !ok {
		return nil, fmt.Errorf("prompt not found: %s", name)
	}
	return promptMap, nil
}

// Field 는 프롬프트 맵에서 필요한 필드를 가져온다.
func Field(data map[string]string, key string, label string) (string, error) {
	value, ok := data[key]
	if !ok {
		return "", fmt.Errorf("prompt field missing: %s", label)
	}
	return value, nil
}
