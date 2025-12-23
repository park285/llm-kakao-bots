package models

// RequiredStringFieldSchema 는 단일 문자열 필드가 필수인 스키마를 만든다.
func RequiredStringFieldSchema(field string) map[string]any {
	return requiredObjectSchema(map[string]any{
		field: map[string]any{
			"type": "string",
		},
	}, []string{field})
}

// RequiredStringArrayFieldSchema 는 문자열 배열 필드가 필수인 스키마를 만든다.
func RequiredStringArrayFieldSchema(field string) map[string]any {
	return requiredObjectSchema(map[string]any{
		field: map[string]any{
			"type": "array",
			"items": map[string]any{
				"type": "string",
			},
		},
	}, []string{field})
}

func requiredObjectSchema(properties map[string]any, required []string) map[string]any {
	return map[string]any{
		"type":       "object",
		"properties": properties,
		"required":   required,
	}
}
