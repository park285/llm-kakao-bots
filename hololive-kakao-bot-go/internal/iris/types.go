package iris

// Config 는 타입이다.
type Config struct {
	Port              int    `json:"port"`
	PollingSpeed      int    `json:"pollingSpeed"`
	MessageRate       int    `json:"messageRate"`
	WebserverEndpoint string `json:"webserverEndpoint"`
}

// DecryptRequest 는 타입이다.
type DecryptRequest struct {
	Data string `json:"data"`
}

// DecryptResponse 는 타입이다.
type DecryptResponse struct {
	Decrypted string `json:"decrypted"`
}

// ReplyRequest 는 타입이다.
type ReplyRequest struct {
	Type string `json:"type"`
	Room string `json:"room"`
	Data string `json:"data"`
}

// ImageReplyRequest 는 타입이다.
type ImageReplyRequest struct {
	Type string `json:"type"`
	Room string `json:"room"`
	Data string `json:"data"`
}

// Message 는 타입이다.
type Message struct {
	Msg    string       `json:"msg"`
	Room   string       `json:"room"`
	Sender *string      `json:"sender,omitempty"`
	JSON   *MessageJSON `json:"json,omitempty"`
}

// MessageJSON 는 타입이다.
type MessageJSON struct {
	UserID    string `json:"user_id,omitempty"`
	Message   string `json:"message,omitempty"`
	ChatID    string `json:"chat_id,omitempty"`
	Type      string `json:"type,omitempty"`
	CreatedAt string `json:"created_at,omitempty"`
}
