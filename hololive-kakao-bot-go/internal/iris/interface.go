package iris

import "context"

// Client: Iris 메시지 전송 인터페이스다.
type Client interface {
	SendMessage(ctx context.Context, room, message string) error
	SendImage(ctx context.Context, room, imageBase64 string) error
	Ping(ctx context.Context) bool
	GetConfig(ctx context.Context) (*Config, error)
	Decrypt(ctx context.Context, data string) (string, error)
}
