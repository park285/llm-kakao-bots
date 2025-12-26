package adapter

import "github.com/kapu/hololive-kakao-bot-go/internal/util"

type helpTemplateData struct {
	Emoji  UIEmoji
	Prefix string
}

// FormatHelp: 도움말 메시지를 생성한다.
func (f *ResponseFormatter) FormatHelp() string {
	data := helpTemplateData{Emoji: DefaultEmoji, Prefix: f.prefix}
	rendered, err := executeFormatterTemplate("help.tmpl", data)
	if err != nil {
		return ErrorMessage(ErrDisplayHelpFailed)
	}

	instruction, body := splitTemplateInstruction(rendered)
	if instruction == "" || body == "" {
		return rendered
	}
	return util.ApplyKakaoSeeMorePadding(body, instruction)
}
