package messageprovider

import (
	"testing"
)

func TestNewFromYAML(t *testing.T) {
	tests := []struct {
		name        string
		yamlContent string
		wantErr     bool
	}{
		{"valid", "key: value", false},
		{"valid nested", "section:\n  key: value", false},
		{"invalid yaml", "key: : value", true},
		{"not a map", "- list item", true},
		{"empty", "", false}, // empty yaml parses to nil, normalize returns nil, type assertion fails? Let's check logic.
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewFromYAML(tt.yamlContent)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewFromYAML() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestProvider_Get(t *testing.T) {
	yamlContent := `
simple: "hello"
nested:
  key: "nested value"
  deep:
    key: "deep value"
template: "Hello {name}, count is {count}"
numeric: 123
boolean: true
list:
  - item1
  - item2
`
	provider, err := NewFromYAML(yamlContent)
	if err != nil {
		t.Fatalf("failed to create provider: %v", err)
	}

	tests := []struct {
		name   string
		key    string
		params []Param
		want   string
	}{
		{"simple key", "simple", nil, "hello"},
		{"nested key", "nested.key", nil, "nested value"},
		{"deep nested key", "nested.deep.key", nil, "deep value"},
		{"template substitution", "template", []Param{P("name", "Alice"), P("count", 42)}, "Hello Alice, count is 42"},
		{"missing param", "template", []Param{P("name", "Bob")}, "Hello Bob, count is {count}"},
		{"numeric value", "numeric", nil, "123"},
		{"boolean value", "boolean", nil, "true"},
		{"unknown key", "unknown", nil, "unknown"},
		{"unknown nested key", "nested.unknown", nil, "nested.unknown"},
		{"key is not string", "list", nil, "[item1 item2]"}, // fmt.Sprint slice format
		{"empty key", "", nil, ""},
		{"whitespace key", "   ", nil, "   "},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := provider.Get(tt.key, tt.params...)
			if got != tt.want {
				t.Errorf("Get(%q) = %q, want %q", tt.key, got, tt.want)
			}
		})
	}
}

func TestNewFromYAMLAtPath(t *testing.T) {
	yamlContent := `
section1:
  key: "value1"
section2:
  key: "value2"
`

	t.Run("valid path", func(t *testing.T) {
		p, err := NewFromYAMLAtPath(yamlContent, "section2")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got := p.Get("key"); got != "value2" {
			t.Errorf("expected value2, got %s", got)
		}
	})

	t.Run("root path", func(t *testing.T) {
		p, err := NewFromYAMLAtPath(yamlContent, "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got := p.Get("section1.key"); got != "value1" {
			t.Errorf("expected value1, got %s", got)
		}
	})

	t.Run("invalid path", func(t *testing.T) {
		_, err := NewFromYAMLAtPath(yamlContent, "section3")
		if err == nil {
			t.Error("expected error for invalid path, got nil")
		}
	})

	t.Run("path not a map", func(t *testing.T) {
		raw := `key: "value"`
		_, err := NewFromYAMLAtPath(raw, "key")
		if err == nil {
			t.Error("expected error when path target is not a map, got nil")
		}
	})
}

func TestProvider_NilReceiver(t *testing.T) {
	var p *Provider
	if got := p.Get("key"); got != "key" {
		t.Errorf("expected 'key', got '%s'", got)
	}
}

func TestNormalizeYAMLValue(t *testing.T) {
	// map[any]any -> map[string]any 변환 테스트 (YAML 파서 특성상 필요)
	input := map[any]any{
		"key": "value",
		123:   "numeric key",
		true:  "bool key",
		"nested": map[any]any{
			"inner": "val",
		},
		"list": []any{
			map[any]any{"k": "v"},
		},
	}

	normalized := normalizeYAMLValue(input)
	m, ok := normalized.(map[string]any)
	if !ok {
		t.Fatalf("expected map[string]any, got %T", normalized)
	}

	if m["key"] != "value" {
		t.Errorf("expected value, got %v", m["key"])
	}
	if m["123"] != "numeric key" {
		t.Errorf("expected numeric key, got %v", m["123"])
	}
	if m["true"] != "bool key" {
		t.Errorf("expected bool key, got %v", m["true"])
	}

	nested, ok := m["nested"].(map[string]any)
	if !ok {
		t.Errorf("expected nested map[string]any, got %T", m["nested"])
	} else if nested["inner"] != "val" {
		t.Errorf("expected inner val, got %v", nested["inner"])
	}

	list, ok := m["list"].([]any)
	if !ok {
		t.Errorf("expected list []any, got %T", m["list"])
	} else if len(list) > 0 {
		item, ok := list[0].(map[string]any)
		if !ok {
			t.Errorf("expected list item map[string]any, got %T", list[0])
		} else if item["k"] != "v" {
			t.Errorf("expected list item v, got %v", item["k"])
		}
	}
}
