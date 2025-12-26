package admin

import (
	"encoding/json"
	"fmt"
	"io"
	"time"
)

// timeNow 현재 시간 반환 (테스트 가능하도록 분리)
func timeNow() time.Time {
	return time.Now()
}

// jsonMarshal JSON 직렬화 헬퍼
func jsonMarshal(v any) ([]byte, error) {
	return json.Marshal(v)
}

// writeSSE SSE 형식으로 데이터 전송
func writeSSE(w io.Writer, payload []byte) (int, error) {
	return fmt.Fprintf(w, "data: %s\n\n", payload)
}
