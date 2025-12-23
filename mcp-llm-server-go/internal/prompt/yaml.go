package prompt

import (
	"fmt"
	"io/fs"
	"path"
	"strings"

	"gopkg.in/yaml.v3"
)

// LoadYAMLMapping 는 프롬프트 YAML 파일을 로드한다.
func LoadYAMLMapping(fsys fs.FS, filePath string) (map[string]string, error) {
	data, err := fs.ReadFile(fsys, filePath)
	if err != nil {
		return nil, fmt.Errorf("read prompt file: %w", err)
	}

	var raw map[string]any
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("parse prompt yaml: %w", err)
	}

	mapping := make(map[string]string)
	for key, value := range raw {
		if value == nil {
			mapping[key] = ""
			continue
		}
		mapping[key] = fmt.Sprint(value)
	}

	system, ok := mapping["system"]
	if ok && strings.TrimSpace(system) != "" {
		if err := ValidateSystemStatic(filePath, system); err != nil {
			return nil, err
		}
	}

	return mapping, nil
}

// LoadYAMLDir 는 디렉터리의 프롬프트 YAML 을 로드한다.
func LoadYAMLDir(fsys fs.FS, dir string) (map[string]map[string]string, error) {
	paths, err := fs.Glob(fsys, path.Join(dir, "*.yml"))
	if err != nil {
		return nil, fmt.Errorf("glob prompt dir: %w", err)
	}
	yamlPaths, err := fs.Glob(fsys, path.Join(dir, "*.yaml"))
	if err != nil {
		return nil, fmt.Errorf("glob prompt dir: %w", err)
	}
	paths = append(paths, yamlPaths...)

	prompts := make(map[string]map[string]string)
	for _, filePath := range paths {
		promptName := strings.TrimSuffix(path.Base(filePath), path.Ext(filePath))
		mapping, err := LoadYAMLMapping(fsys, filePath)
		if err != nil {
			return nil, err
		}
		prompts[promptName] = mapping
	}
	return prompts, nil
}
