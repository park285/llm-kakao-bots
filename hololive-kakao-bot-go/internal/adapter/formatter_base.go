package adapter

import (
	"fmt"
	"strings"

	"github.com/kapu/hololive-kakao-bot-go/internal/util"
)

// ResponseFormatter: 봇의 응답 메시지를 생성하는 포맷터 (카카오톡 UI 템플릿 적용)
type ResponseFormatter struct {
	prefix string
}

func splitTemplateInstruction(rendered string) (instruction string, body string) {
	trimmed := strings.TrimLeft(rendered, "\r\n")
	if trimmed == "" {
		return "", ""
	}

	parts := strings.SplitN(trimmed, "\n", 2)
	instruction = util.TrimSpace(strings.TrimSuffix(parts[0], "\r"))
	if len(parts) < 2 {
		return instruction, ""
	}

	body = strings.TrimLeft(parts[1], "\r\n")
	return instruction, body
}

// NewResponseFormatter: 새로운 ResponseFormatter 인스턴스를 생성한다.
func NewResponseFormatter(prefix string) *ResponseFormatter {
	if util.TrimSpace(prefix) == "" {
		prefix = "!"
	}
	return &ResponseFormatter{prefix: prefix}
}

// Prefix: 현재 설정된 명령어 접두사를 반환한다.
func (f *ResponseFormatter) Prefix() string {
	if f == nil {
		return "!"
	}
	if trimmed := util.TrimSpace(f.prefix); trimmed != "" {
		return trimmed
	}
	return "!"
}

// FormatError: 에러 메시지를 사용자 친화적인 포맷으로 변환한다.
func (f *ResponseFormatter) FormatError(message string) string {
	return ErrorMessage(message)
}

// MemberNotFound: 멤버를 찾을 수 없을 때의 에러 메시지를 생성한다.
func (f *ResponseFormatter) MemberNotFound(memberName string) string {
	return f.FormatError(fmt.Sprintf("'%s' 멤버를 찾을 수 없습니다.", memberName))
}
