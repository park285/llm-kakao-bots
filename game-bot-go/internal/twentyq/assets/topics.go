package assets

import "embed"

// TopicsFS 는 주제 JSON 파일 FS다.
//
//go:embed topics/*.json
var TopicsFS embed.FS
