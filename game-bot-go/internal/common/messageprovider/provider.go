package messageprovider

import (
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

// Provider 는 타입이다.
type Provider struct {
	root map[string]any
}

// NewFromYAML 는 동작을 수행한다.
func NewFromYAML(yamlContent string) (*Provider, error) {
	var raw any
	if err := yaml.Unmarshal([]byte(yamlContent), &raw); err != nil {
		return nil, fmt.Errorf("unmarshal yaml failed: %w", err)
	}

	if raw == nil {
		return &Provider{root: make(map[string]any)}, nil
	}

	root, ok := normalizeYAMLValue(raw).(map[string]any)
	if !ok {
		return nil, fmt.Errorf("unexpected yaml root type: %T", raw)
	}

	return &Provider{root: root}, nil
}

// NewFromYAMLAtPath 는 동작을 수행한다.
func NewFromYAMLAtPath(yamlContent string, rootKey string) (*Provider, error) {
	provider, err := NewFromYAML(yamlContent)
	if err != nil {
		return nil, err
	}

	rootKey = strings.TrimSpace(rootKey)
	if rootKey == "" {
		return provider, nil
	}

	value, ok := resolveDottedKey(provider.root, rootKey)
	if !ok {
		return nil, fmt.Errorf("yaml root key not found: %q", rootKey)
	}

	sub, ok := value.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("yaml root key must be an object: %q (got %T)", rootKey, value)
	}

	return &Provider{root: sub}, nil
}

// Get 는 동작을 수행한다.
func (p *Provider) Get(key string, params ...Param) string {
	if p == nil {
		return key
	}
	if strings.TrimSpace(key) == "" {
		return key
	}

	value, ok := resolveDottedKey(p.root, key)
	if !ok {
		return key
	}

	template, ok := value.(string)
	if !ok {
		return fmt.Sprint(value)
	}

	out := template
	for _, param := range params {
		out = strings.ReplaceAll(out, "{"+param.Key+"}", fmt.Sprint(param.Value))
	}
	return out
}

// Param 는 타입이다.
type Param struct {
	Key   string
	Value any
}

// P 는 동작을 수행한다.
func P(key string, value any) Param {
	return Param{Key: key, Value: value}
}

func resolveDottedKey(root map[string]any, key string) (any, bool) {
	parts := strings.Split(key, ".")
	var current any = root

	for _, part := range parts {
		nextMap, ok := current.(map[string]any)
		if !ok {
			return nil, false
		}
		next, ok := nextMap[part]
		if !ok {
			return nil, false
		}
		current = next
	}

	return current, true
}

func normalizeYAMLValue(v any) any {
	switch typed := v.(type) {
	case map[string]any:
		out := make(map[string]any, len(typed))
		for k, vv := range typed {
			out[k] = normalizeYAMLValue(vv)
		}
		return out
	case map[any]any:
		out := make(map[string]any, len(typed))
		for k, vv := range typed {
			out[fmt.Sprint(k)] = normalizeYAMLValue(vv)
		}
		return out
	case []any:
		out := make([]any, 0, len(typed))
		for _, vv := range typed {
			out = append(out, normalizeYAMLValue(vv))
		}
		return out
	default:
		return v
	}
}
