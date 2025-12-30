package server

import (
	"net/http"

	"github.com/gorilla/websocket"
)

// wsUpgrader: WebSocket 연결 업그레이드용 설정입니다.
// 관리자 UI는 동일 도메인이므로 Origin 검증은 느슨하게 설정합니다.
var wsUpgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// 관리자 API는 IP 화이트리스트로 보호되므로 Origin 검증 완화
		return true
	},
}
