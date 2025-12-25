package adapter

import (
	"testing"

	"github.com/kapu/hololive-kakao-bot-go/internal/domain"
	"github.com/kapu/hololive-kakao-bot-go/internal/iris"
)

func TestParseMessage_CompactAlarmAdd(t *testing.T) {
	adapter := NewMessageAdapter("!")
	msg := &iris.Message{Msg: "!알람설정 미즈미야"}

	result := adapter.ParseMessage(msg)
	if result == nil {
		t.Fatalf("expected parsed command, got nil")
	}
	if result.Type != domain.CommandAlarmAdd {
		t.Fatalf("expected CommandAlarmAdd, got %s", result.Type)
	}

	member, ok := result.Params["member"].(string)
	if !ok {
		t.Fatalf("expected member param to exist")
	}
	if member != "미즈미야" {
		t.Fatalf("expected member to be '미즈미야', got %s", member)
	}
}

func TestParseMessage_CompactAlarmList(t *testing.T) {
	adapter := NewMessageAdapter("!")
	msg := &iris.Message{Msg: "!알람목록"}

	result := adapter.ParseMessage(msg)
	if result == nil {
		t.Fatalf("expected parsed command, got nil")
	}
	if result.Type != domain.CommandAlarmList {
		t.Fatalf("expected CommandAlarmList, got %s", result.Type)
	}
}

func TestParseMessage_InvalidAlarmCommand(t *testing.T) {
	adapter := NewMessageAdapter("!")
	msg := &iris.Message{Msg: "!알람 설정123"}

	result := adapter.ParseMessage(msg)
	if result == nil {
		t.Fatalf("expected parsed command, got nil")
	}
	if result.Type != domain.CommandAlarmInvalid {
		t.Fatalf("expected CommandAlarmInvalid, got %s", result.Type)
	}
	action, ok := result.Params["action"].(string)
	if !ok || action != "invalid" {
		t.Fatalf("expected action invalid, got %v", result.Params["action"])
	}
}
