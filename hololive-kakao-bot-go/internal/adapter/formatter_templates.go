package adapter

import (
	"embed"
	"fmt"
	"strings"
	sync "sync"
	"text/template"
)

//go:embed templates/*.tmpl
var formatterTemplateFS embed.FS

var (
	formatterTemplates *template.Template
	formatterOnce      sync.Once
	errFormatter       error
)

func executeFormatterTemplate(name string, data any) (string, error) {
	formatterOnce.Do(func() {
		funcMap := template.FuncMap{
			"add": func(a, b int) int { return a + b },
			// 템플릿에서 맵을 생성할 수 있게 해주는 헬퍼 함수
			"dict": func(values ...any) (map[string]any, error) {
				if len(values)%2 != 0 {
					return nil, fmt.Errorf("dict requires even number of arguments")
				}
				dict := make(map[string]any, len(values)/2)
				for i := 0; i < len(values); i += 2 {
					key, ok := values[i].(string)
					if !ok {
						return nil, fmt.Errorf("dict keys must be strings")
					}
					dict[key] = values[i+1]
				}
				return dict, nil
			},
		}
		tmpl := template.New("formatter").Funcs(funcMap)
		var err error
		formatterTemplates, err = tmpl.ParseFS(formatterTemplateFS, "templates/*.tmpl")
		if err != nil {
			errFormatter = fmt.Errorf("failed to parse formatter templates: %w", err)
		}
	})

	if errFormatter != nil {
		return "", errFormatter
	}

	var builder strings.Builder
	if err := formatterTemplates.ExecuteTemplate(&builder, name, data); err != nil {
		return "", fmt.Errorf("failed to execute template %s: %w", name, err)
	}

	return strings.TrimRight(builder.String(), "\n"), nil
}
