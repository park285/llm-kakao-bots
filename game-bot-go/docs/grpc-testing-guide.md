# gRPC API 명세 및 테스트 가이드

> 최종 업데이트: 2025-12-30

## 개요

mcp-llm-server-go는 봇(game-bot-go)과의 내부 통신을 위해 gRPC(plaintext/H2C) 프로토콜을 사용합니다. 이 문서는 gRPC API 명세와 테스트 방법을 설명합니다.

---

## 1. 서버 정보

| 항목 | 값 |
|------|-----|
| **서비스명** | `llm.v1.LLMService` |
| **프로토콜** | gRPC over HTTP/2 Cleartext (H2C) |
| **포트** | `40528` |
| **인증** | API Key (`x-api-key` 헤더) |
| **Proto 파일** | `mcp-llm-server-go/proto/llm/v1/llm_service.proto` |

---

## 2. 사전 준비

### 2.1 grpcurl 설치

```bash
# Ubuntu/Debian
sudo apt install grpcurl

# macOS
brew install grpcurl

# Go 설치
go install github.com/fullstorydev/grpcurl/cmd/grpcurl@latest
```

### 2.2 환경 변수 설정

```bash
# API 키 (필수)
export GRPC_API_KEY="322e303ee866a7ff87d5d04427c31c4948b484e009f644b5fcc32db85e2fb18e"

# 서버 주소
export GRPC_HOST="localhost:40528"
```

---

## 3. RPC 메서드 명세

### 3.1 공통 메서드

#### GetModelConfig
현재 LLM 모델 설정을 조회합니다.

```bash
grpcurl -plaintext \
  -H "x-api-key: 322e303ee866a7ff87d5d04427c31c4948b484e009f644b5fcc32db85e2fb18e" \
  localhost:40528 llm.v1.LLMService.GetModelConfig
```

**응답 예시:**
```json
{
  "modelDefault": "gemini-3-flash-preview",
  "modelHints": "gemini-3-flash-preview",
  "modelAnswer": "gemini-3-flash-preview",
  "modelVerify": "gemini-3-flash-preview",
  "temperature": 1,
  "configuredTemperature": 0.7,
  "timeoutSeconds": 60,
  "maxRetries": 3,
  "http2Enabled": true,
  "transportMode": "h2c"
}
```

#### GuardIsMalicious
입력 텍스트의 악의적 여부를 확인합니다.

```bash
grpcurl -plaintext \
  -H "x-api-key: 322e303ee866a7ff87d5d04427c31c4948b484e009f644b5fcc32db85e2fb18e" \
  -d '{"input_text": "테스트 메시지"}' \
  localhost:40528 llm.v1.LLMService.GuardIsMalicious
```

**응답:**
```json
{
  "malicious": false
}
```

#### EndSession
세션을 종료합니다.

```bash
grpcurl -plaintext \
  -H "x-api-key: 322e303ee866a7ff87d5d04427c31c4948b484e009f644b5fcc32db85e2fb18e" \
  -d '{"session_id": "twentyq:12345678"}' \
  localhost:40528 llm.v1.LLMService.EndSession
```

---

### 3.2 TwentyQ (스무고개) 메서드

#### TwentyQGetCategories
사용 가능한 카테고리 목록을 조회합니다.

```bash
grpcurl -plaintext \
  -H "x-api-key: 322e303ee866a7ff87d5d04427c31c4948b484e009f644b5fcc32db85e2fb18e" \
  localhost:40528 llm.v1.LLMService.TwentyQGetCategories
```

**응답:**
```json
{
  "categories": ["person", "organism", "object", "place", "concept"]
}
```

#### TwentyQSelectTopic
정답 주제를 선택합니다.

```bash
grpcurl -plaintext \
  -H "x-api-key: 322e303ee866a7ff87d5d04427c31c4948b484e009f644b5fcc32db85e2fb18e" \
  -d '{
    "category": "organism",
    "banned_topics": ["사자", "호랑이"],
    "excluded_categories": []
  }' \
  localhost:40528 llm.v1.LLMService.TwentyQSelectTopic
```

**응답:**
```json
{
  "name": "코끼리",
  "category": "organism",
  "details": {"habitat": "아프리카, 아시아"}
}
```

#### TwentyQGenerateHints
정답에 대한 힌트를 생성합니다.

```bash
grpcurl -plaintext \
  -H "x-api-key: 322e303ee866a7ff87d5d04427c31c4948b484e009f644b5fcc32db85e2fb18e" \
  -d '{
    "target": "코끼리",
    "category": "organism",
    "details": {}
  }' \
  localhost:40528 llm.v1.LLMService.TwentyQGenerateHints
```

#### TwentyQAnswerQuestion
질문에 대한 답변을 생성합니다.

```bash
grpcurl -plaintext \
  -H "x-api-key: 322e303ee866a7ff87d5d04427c31c4948b484e009f644b5fcc32db85e2fb18e" \
  -d '{
    "chat_id": "12345678",
    "namespace": "twentyq",
    "target": "코끼리",
    "category": "organism",
    "question": "그것은 동물인가요?",
    "details": {}
  }' \
  localhost:40528 llm.v1.LLMService.TwentyQAnswerQuestion
```

**응답:**
```json
{
  "scale": "yes",
  "rawText": "네, 그것은 동물입니다."
}
```

#### TwentyQVerifyGuess
정답 추측을 검증합니다.

```bash
grpcurl -plaintext \
  -H "x-api-key: 322e303ee866a7ff87d5d04427c31c4948b484e009f644b5fcc32db85e2fb18e" \
  -d '{
    "target": "코끼리",
    "guess": "코끼리"
  }' \
  localhost:40528 llm.v1.LLMService.TwentyQVerifyGuess
```

#### TwentyQNormalizeQuestion
질문을 정규화합니다.

```bash
grpcurl -plaintext \
  -H "x-api-key: 322e303ee866a7ff87d5d04427c31c4948b484e009f644b5fcc32db85e2fb18e" \
  -d '{"question": "그거 동물임?"}' \
  localhost:40528 llm.v1.LLMService.TwentyQNormalizeQuestion
```

#### TwentyQCheckSynonym
동의어 여부를 확인합니다.

```bash
grpcurl -plaintext \
  -H "x-api-key: 322e303ee866a7ff87d5d04427c31c4948b484e009f644b5fcc32db85e2fb18e" \
  -d '{
    "target": "코끼리",
    "guess": "elephant"
  }' \
  localhost:40528 llm.v1.LLMService.TwentyQCheckSynonym
```

---

### 3.3 TurtleSoup (바다거북) 메서드

#### TurtleSoupGeneratePuzzle
새로운 퍼즐을 생성합니다.

```bash
grpcurl -plaintext \
  -H "x-api-key: 322e303ee866a7ff87d5d04427c31c4948b484e009f644b5fcc32db85e2fb18e" \
  -d '{
    "category": "mystery",
    "difficulty": 3,
    "theme": "일상"
  }' \
  localhost:40528 llm.v1.LLMService.TurtleSoupGeneratePuzzle
```

#### TurtleSoupGetRandomPuzzle
데이터베이스에서 랜덤 퍼즐을 조회합니다.

```bash
grpcurl -plaintext \
  -H "x-api-key: 322e303ee866a7ff87d5d04427c31c4948b484e009f644b5fcc32db85e2fb18e" \
  -d '{"difficulty": 2}' \
  localhost:40528 llm.v1.LLMService.TurtleSoupGetRandomPuzzle
```

#### TurtleSoupRewriteScenario
시나리오를 재작성합니다.

```bash
grpcurl -plaintext \
  -H "x-api-key: 322e303ee866a7ff87d5d04427c31c4948b484e009f644b5fcc32db85e2fb18e" \
  -d '{
    "title": "창문",
    "scenario": "한 남자가 창문을 열었다가 죽었습니다.",
    "solution": "잠수함 안에서 창문을 열었기 때문입니다.",
    "difficulty": 3
  }' \
  localhost:40528 llm.v1.LLMService.TurtleSoupRewriteScenario
```

#### TurtleSoupAnswerQuestion
플레이어 질문에 답변합니다.

```bash
grpcurl -plaintext \
  -H "x-api-key: 322e303ee866a7ff87d5d04427c31c4948b484e009f644b5fcc32db85e2fb18e" \
  -d '{
    "chat_id": "12345678",
    "namespace": "turtlesoup",
    "scenario": "한 남자가 창문을 열었다가 죽었습니다.",
    "solution": "잠수함 안에서 창문을 열었기 때문입니다.",
    "question": "그 남자는 실내에 있었나요?"
  }' \
  localhost:40528 llm.v1.LLMService.TurtleSoupAnswerQuestion
```

**응답:**
```json
{
  "answer": "yes",
  "rawText": "네, 맞습니다.",
  "questionCount": 1, 
  "history": [{"question": "그 남자는 실내에 있었나요?", "answer": "yes"}]
}
```

#### TurtleSoupValidateSolution
정답 검증을 수행합니다.

```bash
grpcurl -plaintext \
  -H "x-api-key: 322e303ee866a7ff87d5d04427c31c4948b484e009f644b5fcc32db85e2fb18e" \
  -d '{
    "chat_id": "12345678",
    "namespace": "turtlesoup",
    "solution": "잠수함 안에서 창문을 열었기 때문입니다.",
    "player_answer": "잠수함에서 창문을 열어서"
  }' \
  localhost:40528 llm.v1.LLMService.TurtleSoupValidateSolution
```

#### TurtleSoupGenerateHint
힌트를 생성합니다.

```bash
grpcurl -plaintext \
  -H "x-api-key: 322e303ee866a7ff87d5d04427c31c4948b484e009f644b5fcc32db85e2fb18e" \
  -d '{
    "chat_id": "12345678",
    "namespace": "turtlesoup",
    "scenario": "한 남자가 창문을 열었다가 죽었습니다.",
    "solution": "잠수함 안에서 창문을 열었기 때문입니다.",
    "level": 1
  }' \
  localhost:40528 llm.v1.LLMService.TurtleSoupGenerateHint
```

---

### 3.4 사용량 조회 메서드

#### GetDailyUsage
당일 사용량을 조회합니다.

```bash
grpcurl -plaintext \
  -H "x-api-key: 322e303ee866a7ff87d5d04427c31c4948b484e009f644b5fcc32db85e2fb18e" \
  localhost:40528 llm.v1.LLMService.GetDailyUsage
```

**응답:**
```json
{
  "usageDate": "2025-12-30",
  "inputTokens": "15234",
  "outputTokens": "8721",
  "totalTokens": "23955",
  "reasoningTokens": "0",
  "requestCount": "42",
  "model": "gemini-3-flash-preview"
}
```

#### GetRecentUsage
최근 N일 사용량을 조회합니다.

```bash
grpcurl -plaintext \
  -H "x-api-key: 322e303ee866a7ff87d5d04427c31c4948b484e009f644b5fcc32db85e2fb18e" \
  -d '{"days": 7}' \
  localhost:40528 llm.v1.LLMService.GetRecentUsage
```

#### GetTotalUsage
총 누적 사용량을 조회합니다.

```bash
grpcurl -plaintext \
  -H "x-api-key: 322e303ee866a7ff87d5d04427c31c4948b484e009f644b5fcc32db85e2fb18e" \
  -d '{"days": 30}' \
  localhost:40528 llm.v1.LLMService.GetTotalUsage
```

---

## 4. 에러 코드

| gRPC Code | 상황 | 해결 방법 |
|-----------|------|-----------|
| `Unauthenticated` | API 키 누락 또는 불일치 | `-H "x-api-key: <KEY>"` 헤더 추가 |
| `Unavailable` | 서버 연결 불가 | gRPC 서버 실행 상태 확인 |
| `DeadlineExceeded` | 요청 타임아웃 | 타임아웃 설정 확인 |
| `Internal` | 서버 내부 오류 | 서버 로그 확인 |
| `InvalidArgument` | 잘못된 요청 파라미터 | 요청 필드 확인 |

---

## 5. 헬퍼 스크립트

### 5.1 환경 설정 스크립트

```bash
#!/bin/bash
# grpc-env.sh

export GRPC_API_KEY="322e303ee866a7ff87d5d04427c31c4948b484e009f644b5fcc32db85e2fb18e"
export GRPC_HOST="localhost:40528"

# 별칭 설정
alias grpc='grpcurl -plaintext -H "x-api-key: 322e303ee866a7ff87d5d04427c31c4948b484e009f644b5fcc32db85e2fb18e" localhost:40528'
```

사용 방법:
```bash
source grpc-env.sh
grpc llm.v1.LLMService.GetModelConfig
grpc llm.v1.LLMService.GetDailyUsage
```

### 5.2 서비스 목록 조회

```bash
# 사용 가능한 서비스 목록
grpcurl -plaintext -H "x-api-key: 322e303ee866a7ff87d5d04427c31c4948b484e009f644b5fcc32db85e2fb18e" localhost:40528 list

# 서비스 내 메서드 목록
grpcurl -plaintext -H "x-api-key: 322e303ee866a7ff87d5d04427c31c4948b484e009f644b5fcc32db85e2fb18e" localhost:40528 list llm.v1.LLMService

# 메서드 상세 정보
grpcurl -plaintext -H "x-api-key: 322e303ee866a7ff87d5d04427c31c4948b484e009f644b5fcc32db85e2fb18e" localhost:40528 describe llm.v1.LLMService.GetModelConfig
```

---

## 6. 문제 해결

### 6.1 연결 오류

```bash
# gRPC 서버 상태 확인
docker logs mcp-llm-server 2>&1 | grep grpc_server_start

# 포트 리스닝 확인
ss -tlnp | grep 40528
```

### 6.2 인증 오류

```bash
# API 키 확인
docker exec mcp-llm-server printenv | grep HTTP_API_KEY

# 올바른 헤더 형식
# ✅ 올바름: -H "x-api-key: <KEY>"
# ❌ 틀림: -H "Authorization: <KEY>"
```

### 6.3 Docker 컨테이너 내부에서 테스트

```bash
# twentyq-bot에서 mcp-llm-server로 연결 테스트
docker exec twentyq-bot wget -q -O- http://mcp-llm-server:40527/health
```

---

## 변경 이력

| 날짜 | 버전 | 변경 내용 |
|------|------|----------|
| 2025-12-30 | 1.0 | 초기 문서 작성 |
