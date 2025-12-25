package iris

// Config: Iris(메신저 연동) 서비스 설정을 담는 구조체
type Config struct {
	Port              int    `json:"port"`
	PollingSpeed      int    `json:"pollingSpeed"`
	MessageRate       int    `json:"messageRate"`
	WebserverEndpoint string `json:"webserverEndpoint"`
}

// DecryptRequest: 카카오톡 메시지 복호화 요청 구조체
type DecryptRequest struct {
	Data string `json:"data"`
}

// DecryptResponse: 카카오톡 메시지 복호화 응답 구조체
type DecryptResponse struct {
	Decrypted string `json:"decrypted"`
}

// ReplyRequest: 텍스트 답장 전송 요청 구조체
type ReplyRequest struct {
	Type string `json:"type"`
	Room string `json:"room"`
	Data string `json:"data"`
}

// ImageReplyRequest: 이미지 답장 전송 요청 구조체 (Base64 데이터 포함)
type ImageReplyRequest struct {
	Type string `json:"type"`
	Room string `json:"room"`
	Data string `json:"data"`
}

// Message: 수신된 카카오톡 메시지 구조체
type Message struct {
	Msg    string       `json:"msg"`
	Room   string       `json:"room"`
	Sender *string      `json:"sender,omitempty"`
	JSON   *MessageJSON `json:"json,omitempty"`
}

// MessageJSON: 메시지 세부 정보를 담는 JSON 구조체
type MessageJSON struct {
	UserID    string `json:"user_id,omitempty"`
	Message   string `json:"message,omitempty"`
	ChatID    string `json:"chat_id,omitempty"`
	Type      string `json:"type,omitempty"`
	CreatedAt string `json:"created_at,omitempty"`
}
