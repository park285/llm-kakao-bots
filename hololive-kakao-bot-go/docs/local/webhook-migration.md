# WebSocket → HTTP Webhook 마이그레이션 가이드

> v1.0-websocket-based → v2.0-webhook-based

## 변경 사항 요약

### 제거된 기능
- [REMOVED] WebSocket 클라이언트 (`internal/iris/websocket.go`)
- [REMOVED] `IRIS_WS_URL` 환경 변수
- [REMOVED] 재연결 로직
- [REMOVED] WebSocket 상태 관리

### 추가된 기능
- [ADDED] HTTP Webhook 서버 (Gin 기반)
- [ADDED] `/webhook` 엔드포인트
- [ADDED] `/health` 헬스체크 엔드포인트
- [ADDED] `SERVER_PORT` 환경 변수

---

## 환경 변수 변경

### Before (WebSocket)
```bash
IRIS_BASE_URL=http://localhost:3000
IRIS_WS_URL=ws://localhost:3000/ws  # 제거됨
```

### After (HTTP Webhook)
```bash
IRIS_BASE_URL=http://localhost:3000
SERVER_PORT=30001  # 새로 추가
```

---

## Iris 서버 설정

Iris v2.0+는 내장 WebhookRouter를 통해 prefix 기반 라우팅을 지원합니다.

### 설정 파일 위치
```
/data/local/tmp/config.json
```

### 설정 예제
```json
{
  "routes": [
    {
      "prefix": "/홀로",
      "webhookUrl": "http://172.17.0.1:30001/webhook",
      "enabled": true
    }
  ],
  "dbPollingRate": 100,
  "messageSendRate": 50
}
```

### 동작 방식

1. **메시지 매칭**: `/홀로라이브 스케줄` → `/홀로` prefix 매칭
2. **Webhook 전송**: Iris가 `http://172.17.0.1:30001/webhook`로 POST 요청
3. **봇 처리**: 메시지 파싱 및 응답
4. **응답 전송**: 봇이 Iris `/reply` 엔드포인트로 HTTP POST

---

## 🚀 봇 실행

### 1. 환경 변수 설정
`.env` 파일 업데이트:
```bash
# Iris 설정
IRIS_BASE_URL=http://localhost:3000

# 봇 서버 설정
SERVER_PORT=30001

# 기타 설정 (기존과 동일)
HOLODEX_API_KEY_1=your_key
GOOGLE_API_KEY=your_key
# ...
```

### 2. 봇 실행
```bash
go run cmd/bot/main.go
```

실행 시 로그:
```
INFO: Hololive KakaoTalk Bot starting...
INFO: Starting HTTP webhook server  port=30001
INFO: Bot started (webhook mode), waiting for signals...
```

### 3. 헬스체크
```bash
curl http://localhost:30001/health
# {"status":"ok"}
```

---

## 📊 Webhook 페이로드 포맷

### Iris → 봇 (Request)
```json
{
  "room": "채팅방 이름",
  "user": "사용자 이름",
  "msg": "/홀로라이브 공지",
  "sender": "발신자 이름",
  "json": {
    "chat_id": "1234567890",
    "user_id": "9876543210"
  },
  "threadId": null
}
```

**⚠️ 중요**: `json.chat_id`를 사용해야 메시지 전송 가능 (`room` 필드 아님)

### 봇 → Iris (Response)
```json
{
  "status": "ok"
}
```

### 봇 → Iris (메시지 전송)
```http
POST http://localhost:3000/reply
Content-Type: application/json

{
  "type": "text",
  "room": "1234567890",
  "data": "안녕하세요!"
}
```

---

## 🔄 롤백 방법

문제 발생 시 WebSocket 버전으로 롤백:

```bash
# 1. 태그로 체크아웃
git checkout v1.0-websocket-based

# 2. 재빌드
CGO_ENABLED=0 go build -tags go_json -o bin/bot ./cmd/bot

# 3. 환경 변수 복구
IRIS_WS_URL=ws://localhost:3000/ws

# 4. 실행
./bin/bot
```

---

## 🧪 테스트

### 1. 로컬 테스트 (curl)
```bash
curl -X POST http://localhost:30001/webhook \
  -H "Content-Type: application/json" \
  -d '{
    "room": "테스트방",
    "user": "테스터",
    "msg": "/홀로 도움",
    "sender": "테스터",
    "json": {
      "chat_id": "1234567890",
      "user_id": "9876543210"
    }
  }'
```

### 2. Iris 통합 테스트

1. Iris `config.json` 설정 완료
2. 카카오톡에서 `/홀로 도움` 메시지 전송
3. 봇 로그 확인:
   ```
   INFO: Webhook received  chatId=1234567890 user=9876543210 msg=/홀로 도움
   INFO: Command received  type=help user=9876543210 room=1234567890
   ```

---

## 🏗️ 아키텍처 비교

### Before: WebSocket
```
┌─────────────┐         ┌──────────────┐
│   Iris      │  WS     │   봇 (Sub)   │
│  Platform   │────────▶│   WebSocket  │
│             │◀────────│   Client     │
└─────────────┘ Persist └──────────────┘
                 Connection
```

### After: HTTP Webhook
```
┌─────────────┐         ┌──────────────┐
│   Iris      │  HTTP   │   봇 (Srv)   │
│  Router     │────────▶│   Webhook    │
│             │◀────────│   Handler    │
└─────────────┘ Stateless└──────────────┘
```

---

## 📈 성능 비교

| 항목 | WebSocket | HTTP Webhook |
|------|-----------|--------------|
| **연결 유지** | 필요 | 불필요 |
| **재연결 로직** | 복잡 | 불필요 |
| **메시지 필터링** | 봇에서 처리 | Iris에서 처리 |
| **수평 확장** | 제한적 | 로드밸런서 지원 |
| **디버깅** | 어려움 | HTTP 로그 활용 |

---

## 🔍 트러블슈팅

### 1. "IRIS_WS_URL is required" 에러
```
FATAL: IRIS_WS_URL is required
```
**해결**: 환경 변수에서 `IRIS_WS_URL` 제거, `SERVER_PORT` 추가

### 2. "Address already in use" 에러
```
ERROR: HTTP server error: listen tcp :30001: bind: address already in use
```
**해결**:
```bash
# 포트 사용 프로세스 확인
lsof -i :30001

# 또는 다른 포트 사용
SERVER_PORT=30002
```

### 3. Webhook이 호출되지 않음
- Iris `config.json` 확인
- prefix 매칭 확인 (`/홀로` 등)
- 네트워크 연결 확인 (`curl http://172.17.0.1:30001/health`)

### 4. 메시지 전송 실패
```
ERROR: Failed to send message: invalid room
```
**원인**: `req.Room` 대신 `req.JSON.ChatID` 사용 필요
**해결**: 코드에서 이미 처리됨 (`internal/server/webhook.go:66`)

---

## 📚 참고 자료

- [Iris Integration Guide](https://github.com/park285/iris-integration-guide)
- [iris-20q-service 구현 예제](../iris-20q-service/)
- [Gin Framework 문서](https://gin-gonic.com/)

---

**마이그레이션 날짜**: 2025-10-30
**롤백 태그**: `v1.0-websocket-based`
