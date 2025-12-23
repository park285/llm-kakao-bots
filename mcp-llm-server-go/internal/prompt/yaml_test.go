package prompt

import (
	"testing"
	"testing/fstest"
)

func TestLoadYAMLMapping(t *testing.T) {
	fsys := fstest.MapFS{
		"sample.yml": {Data: []byte("system: hello\nuser: hi\ncount: 3\n")},
	}

	mapping, err := LoadYAMLMapping(fsys, "sample.yml")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mapping["system"] != "hello" {
		t.Fatalf("unexpected system: %s", mapping["system"])
	}
	if mapping["count"] != "3" {
		t.Fatalf("unexpected count: %s", mapping["count"])
	}
}

func TestLoadYAMLMappingInvalidSystem(t *testing.T) {
	fsys := fstest.MapFS{
		"bad.yml": {Data: []byte("system: \"hello {name}\"\n")},
	}
	if _, err := LoadYAMLMapping(fsys, "bad.yml"); err == nil {
		t.Fatalf("expected error")
	}
}

func TestLoadYAMLDir(t *testing.T) {
	fsys := fstest.MapFS{
		"prompts/a.yml":  {Data: []byte("system: alpha\n")},
		"prompts/b.yaml": {Data: []byte("system: beta\n")},
	}

	prompts, err := LoadYAMLDir(fsys, "prompts")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(prompts) != 2 {
		t.Fatalf("expected 2 prompts, got %d", len(prompts))
	}
	if prompts["a"]["system"] != "alpha" {
		t.Fatalf("unexpected prompt value")
	}
}
