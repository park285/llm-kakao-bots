# Game Bot Admin API 문서

## 개요

Game Bot (TwentyQ, TurtleSoup) 관리를 위한 Admin API 명세서입니다.
모든 Admin API는 `admin-dashboard`를 통해 프록시되며, 인증은 gateway 레벨에서 처리됩니다.

## 접근 경로

| 서비스 | 직접 접근 (내부) | admin-dashboard 프록시 |
|:---|:---|:---|
| TwentyQ | `http://twentyq-bot:30002/admin/*` | `https://admin.capu.blog/admin/api/twentyq/admin/*` |
| TurtleSoup | `http://turtle-soup-bot:30003/admin/*` | `https://admin.capu.blog/admin/api/turtle/admin/*` |

## OpenTelemetry 통합

모든 Admin API 요청은 자동으로 OTel 추적됩니다:
- HTTP 서버에서 `otelhttp` 미들웨어 적용
- 각 요청에 대해 `trace_id`, `span_id`가 로그에 자동 주입
- Jaeger UI에서 `twentyq-bot` / `turtle-soup-bot` 서비스로 조회 가능

---

## TwentyQ Admin APIs

Base URL: `/admin` (직접) 또는 `/admin/api/twentyq/admin` (프록시)

### GET /admin/stats

통합 게임 통계를 조회합니다.

**Response:**
```json
{
  "totalGamesPlayed": 1234,
  "totalGamesCompleted": 890,
  "totalSurrenders": 344,
  "successRate": 72.12,
  "activeSessions": 5,
  "totalParticipants": 456,
  "last24HoursGames": 23
}
```

| 필드 | 타입 | 설명 |
|:---|:---|:---|
| `totalGamesPlayed` | int | 전체 플레이된 게임 수 |
| `totalGamesCompleted` | int | 성공적으로 완료된 게임 수 |
| `totalSurrenders` | int | 항복으로 종료된 게임 수 |
| `successRate` | float | 성공률 (%) |
| `activeSessions` | int | 현재 활성 세션 수 |
| `totalParticipants` | int | 총 참여자 수 (고유 사용자) |
| `last24HoursGames` | int | 최근 24시간 동안 완료된 게임 수 |

---

### GET /admin/sessions

현재 활성 게임 세션 목록을 조회합니다.

**Response:**
```json
{
  "status": "ok",
  "sessions": [
    {
      "chatId": "room123",
      "category": "인물",
      "questionCount": 5,
      "startedAt": "2026-01-02T10:00:00Z",
      "ttlSeconds": 3600
    }
  ],
  "count": 1
}
```

---

### GET /admin/sessions/{id}

진행 중인 세션 상세 정보와 Q&A 히스토리를 조회합니다.

**Path Parameters:**
- `id`: 채팅방 ID (chatId)

**Response:**
```json
{
  "status": "ok",
  "session": {
    "chatId": "room123",
    "target": "스마트폰",
    "category": "물건",
    "intro": "당신이 생각한 것은 '물건' 카테고리입니다.",
    "questionCount": 5,
    "hintCount": 1,
    "ttlSeconds": 3200
  },
  "history": [
    {
      "questionNumber": 1,
      "question": "휴대할 수 있나요?",
      "answer": "예",
      "isChain": false
    }
  ],
  "players": [
    {"userId": "user123", "sender": "홍길동"}
  ]
}
```

---

### POST /admin/sessions/{id}/hint

진행 중인 게임에 GM 힌트를 주입합니다.

**Path Parameters:**
- `id`: 채팅방 ID (chatId)

**Request:**
```json
{
  "message": "정답은 일상에서 매일 사용하는 물건입니다."
}
```

**Response:**
```json
{
  "status": "ok",
  "message": "hint injected"
}
```

---

### POST /admin/sessions/cleanup

오래된 세션을 일괄 정리합니다.

**Request:**
```json
{
  "olderThanHours": 24
}
```

**Response:**
```json
{
  "status": "ok",
  "deletedCount": 5
}
```

---

### DELETE /admin/sessions/{id}

특정 세션을 강제 종료합니다.

**Path Parameters:**
- `id`: 채팅방 ID (chatId)

**Response (성공):**
```json
{
  "status": "ok",
  "message": "session deleted"
}
```

**Response (세션 없음):**
```json
{
  "error": "SESSION_NOT_FOUND",
  "message": "session not found"
}
```

---

### GET /admin/games

게임 히스토리를 조회합니다 (페이지네이션 지원).

**Query Parameters:**
| 파라미터 | 타입 | 기본값 | 설명 |
|:---|:---|:---|:---|
| `limit` | int | 50 | 조회할 최대 게임 수 (max: 100) |
| `offset` | int | 0 | 건너뛸 게임 수 |
| `category` | string | - | 카테고리 필터 |
| `result` | string | - | 결과 필터 (success/surrender) |

**Response:**
```json
{
  "status": "ok",
  "games": [
    {
      "sessionId": "abc123",
      "chatId": "room456",
      "category": "인물",
      "result": "success",
      "participantCount": 3,
      "questionCount": 15,
      "hintCount": 2,
      "completedAt": "2026-01-02T10:30:00Z"
    }
  ],
  "total": 1234,
  "limit": 50,
  "offset": 0
}
```

---

### GET /admin/leaderboard

리더보드를 조회합니다.

**Query Parameters:**
| 파라미터 | 타입 | 기본값 | 설명 |
|:---|:---|:---|:---|
| `limit` | int | 20 | 조회할 최대 항목 수 (max: 100) |

**Response:**
```json
{
  "status": "ok",
  "leaderboard": [
    {
      "rank": 1,
      "userId": "user123",
      "chatId": "room456",
      "totalGamesCompleted": 50,
      "successRate": 85.5,
      "bestQuestionCount": 8,
      "bestTarget": "아인슈타인"
    }
  ],
  "count": 20
}
```

---

### POST /admin/synonyms

동의어 매핑을 생성합니다 (Valkey Hash 저장).

**Request:**
```json
{
  "canonical": "스마트폰",
  "aliases": ["스맛폰", "핸드폰", "휴대폰"]
}
```

**Response:**
```json
{
  "status": "ok",
  "message": "synonym created",
  "canonical": "스마트폰",
  "aliases": ["스맛폰", "핸드폰", "휴대폰"]
}
```

---

### GET /admin/synonyms

동의어 매핑을 조회합니다.

**Query Parameters:**
| 파라미터 | 타입 | 설명 |
|:---|:---|:---|
| `query` | string | 특정 alias 조회 (없으면 전체 목록) |

**Response (전체 목록):**
```json
{
  "status": "ok",
  "synonyms": [
    {
      "canonical": "스마트폰",
      "aliases": ["스맛폰", "핸드폰"]
    }
  ],
  "count": 1
}
```

---

### POST /admin/games/{id}/audit

판정 리뷰 기록을 생성합니다.

**Path Parameters:**
- `id`: 게임 세션 ID

**Request:**
```json
{
  "questionIndex": 5,
  "verdict": "AI_WRONG",
  "reason": "동의어 미처리",
  "adminUserId": "admin123"
}
```

| 필드 | 타입 | 설명 |
|:---|:---|:---|
| `verdict` | string | `AI_CORRECT`, `AI_WRONG`, `UNCLEAR` |
| `reason` | string | 판정 이유 |

**Response:**
```json
{
  "status": "ok",
  "message": "audit recorded",
  "id": 123
}
```

---

### POST /admin/games/{id}/refund

유저 스탯을 복원합니다 (AI 오판으로 인한 항복 시).

**Path Parameters:**
- `id`: 게임 세션 ID

**Request:**
```json
{
  "userId": "user123",
  "restoreStats": true,
  "adminUserId": "admin456",
  "reason": "AI 오판으로 인한 복구"
}
```

**Response:**
```json
{
  "status": "ok",
  "message": "refund applied"
}
```

---

### GET /admin/users/stats

유저 통계 목록을 조회합니다.

**Query Parameters:**
| 파라미터 | 타입 | 기본값 | 설명 |
|:---|:---|:---|:---|
| `chatId` | string | - | 채팅방 ID 필터 |
| `limit` | int | 50 | 최대 조회 수 (max: 100) |
| `offset` | int | 0 | 건너뛸 수 |

**Response:**
```json
{
  "status": "ok",
  "stats": [
    {
      "id": "room123:user456",
      "chatId": "room123",
      "userId": "user456",
      "totalGamesStarted": 15,
      "totalGamesCompleted": 12,
      "totalSurrenders": 2,
      "bestScoreQuestionCnt": 8
    }
  ],
  "total": 150,
  "limit": 50,
  "offset": 0
}
```

---

### GET /admin/users/{id}/stats

특정 유저의 통계를 조회합니다.

**Path Parameters:**
- `id`: 유저 ID

**Query Parameters:**
| 파라미터 | 타입 | 설명 |
|:---|:---|:---|
| `chatId` | string | 채팅방 ID 필터 (선택) |

**Response:**
```json
{
  "status": "ok",
  "stats": [ ... ]
}
```

---

### DELETE /admin/users/{id}/stats

유저 통계를 리셋(삭제)합니다.

**Path Parameters:**
- `id`: 유저 ID

**Query Parameters:**
| 파라미터 | 타입 | 설명 |
|:---|:---|:---|
| `chatId` | string | 채팅방 ID 필터 (선택, 없으면 모든 채팅방) |

**Response:**
```json
{
  "status": "ok",
  "message": "user stats reset",
  "deletedCount": 3
}
```

---

### GET /admin/audits

오디트 로그 목록을 조회합니다.

**Query Parameters:**
| 파라미터 | 타입 | 기본값 | 설명 |
|:---|:---|:---|:---|
| `sessionId` | string | - | 세션 ID 필터 |
| `limit` | int | 50 | 최대 조회 수 (max: 100) |
| `offset` | int | 0 | 건너뛸 수 |

**Response:**
```json
{
  "status": "ok",
  "logs": [
    {
      "id": 1,
      "sessionId": "sess123",
      "questionIndex": 5,
      "verdict": "AI_WRONG",
      "reason": "동의어 미처리",
      "adminUserId": "admin123",
      "createdAt": "2026-01-02T10:30:00Z"
    }
  ],
  "total": 25,
  "limit": 50,
  "offset": 0
}
```

---

### GET /admin/refunds

리펀드 로그 목록을 조회합니다.

**Query Parameters:**
| 파라미터 | 타입 | 기본값 | 설명 |
|:---|:---|:---|:---|
| `sessionId` | string | - | 세션 ID 필터 |
| `userId` | string | - | 유저 ID 필터 |
| `limit` | int | 50 | 최대 조회 수 (max: 100) |
| `offset` | int | 0 | 건너뛸 수 |

**Response:**
```json
{
  "status": "ok",
  "logs": [
    {
      "id": 1,
      "sessionId": "sess123",
      "userId": "user456",
      "adminUserId": "admin123",
      "reason": "AI 오판으로 인한 스탯 복원",
      "createdAt": "2026-01-02T10:35:00Z"
    }
  ],
  "total": 10,
  "limit": 50,
  "offset": 0
}
```

---

## TurtleSoup Admin APIs

Base URL: `/admin` (직접) 또는 `/admin/api/turtle/admin` (프록시)

### GET /admin/stats

통합 게임 통계를 조회합니다.

**Response:**
```json
{
  "activeSessions": 3,
  "totalSolved": 0,
  "totalFailed": 0,
  "solveRate": 0,
  "avgQuestions": 0,
  "avgHintsPerGame": 0,
  "last24HoursSolve": 0
}
```

> **Note:** 현재 PostgreSQL 연동이 없어 활성 세션 수만 조회됩니다. 향후 업데이트 예정.

---

### GET /admin/sessions

현재 활성 퍼즐 세션 목록을 조회합니다.

**Response:**
```json
{
  "status": "ok",
  "sessions": [
    {
      "sessionId": "sess123",
      "chatId": "room456",
      "puzzleId": "puzzle789",
      "questionCount": 10,
      "hintCount": 1,
      "startedAt": "2026-01-02T10:00:00Z",
      "ttlSeconds": 7200
    }
  ],
  "count": 1
}
```

---

### DELETE /admin/sessions/{id}

특정 세션을 강제 종료합니다.

**Path Parameters:**
- `id`: 세션 ID

**Response (성공):**
```json
{
  "status": "ok",
  "message": "session deleted"
}
```

---

### POST /admin/sessions/cleanup

지정된 시간보다 오래된 세션을 정리합니다.

**Request:**
```json
{
  "olderThanHours": 24
}
```

**Response:**
```json
{
  "deletedCount": 5,
  "message": "cleanup completed"
}
```

---

### POST /admin/sessions/{id}/inject

GM 모드: 특정 세션에 힌트/메시지를 주입합니다.

**Path Parameters:**
- `id`: 세션 ID

**Request:**
```json
{
  "message": "이것은 중요한 힌트입니다.",
  "asBot": true
}
```

| 필드 | 타입 | 설명 |
|:---|:---|:---|
| `message` | string | 주입할 메시지 |
| `asBot` | bool | `true`면 봇 메시지, `false`면 시스템 메시지 |

**Response:**
```json
{
  "status": "ok",
  "message": "hint injected"
}
```

---

### GET /admin/puzzles

퍼즐 목록을 조회합니다.

**Query Parameters:**
| 파라미터 | 타입 | 기본값 | 설명 |
|:---|:---|:---|:---|
| `status` | string | - | 상태 필터 (draft, test, published) |
| `limit` | int | 50 | 최대 조회 수 (max: 100) |
| `offset` | int | 0 | 건너뛸 수 |

**Response:**
```json
{
  "status": "ok",
  "puzzles": [
    {
      "id": 1,
      "title": "전화기 미스터리",
      "scenario": "한 남자가 전화기를 바라보며 울고 있다.",
      "solution": "...",
      "category": "MYSTERY",
      "difficulty": 3,
      "status": "published",
      "playCount": 10,
      "solveCount": 7
    }
  ],
  "total": 15,
  "limit": 50,
  "offset": 0
}
```

---

### POST /admin/puzzles

새 퍼즐을 생성합니다.

**Request:**
```json
{
  "title": "전화기 미스터리",
  "scenario": "한 남자가 전화기를 바라보며 울고 있다.",
  "solution": "그 남자는 국제전화를 걸어 오랜 친구의 목소리를 들었다. 친구가 암에 걸렸다는 소식을 듣고 울고 있다.",
  "category": "MYSTERY",
  "difficulty": 3,
  "hints": ["전화는 해외에서 왔다", "친구 관련이다"],
  "authorId": "admin123"
}
```

**Response:**
```json
{
  "status": "ok",
  "message": "puzzle created",
  "puzzle": { ... }
}
```

---

### GET /admin/puzzles/{id}

단일 퍼즐을 조회합니다.

**Path Parameters:**
- `id`: 퍼즐 ID

**Response:**
```json
{
  "status": "ok",
  "puzzle": { ... }
}
```

---

### PUT /admin/puzzles/{id}

퍼즐을 수정합니다.

**Path Parameters:**
- `id`: 퍼즐 ID

**Request:**
```json
{
  "title": "수정된 제목",
  "status": "published"
}
```

**Response:**
```json
{
  "status": "ok",
  "message": "puzzle updated",
  "puzzle": { ... }
}
```

---

### DELETE /admin/puzzles/{id}

퍼즐을 삭제합니다.

**Path Parameters:**
- `id`: 퍼즐 ID

**Response:**
```json
{
  "status": "ok",
  "message": "puzzle deleted"
}
```

---

### GET /admin/puzzles/stats

퍼즐 전체 통계를 조회합니다.

**Response:**
```json
{
  "status": "ok",
  "stats": {
    "totalPuzzles": 15,
    "publishedCount": 10,
    "draftCount": 5,
    "totalPlays": 120,
    "totalSolves": 85,
    "overallSolveRate": 70.83
  },
  "categoryStats": [
    {
      "category": "MYSTERY",
      "totalGames": 80,
      "solveCount": 60,
      "solveRate": 75.0
    }
  ]
}
```

---

### GET /admin/archives

게임 아카이브(완료된 게임 기록)를 조회합니다.

**Query Parameters:**
| 파라미터 | 타입 | 기본값 | 설명 |
|:---|:---|:---|:---|
| `result` | string | - | 결과 필터 (solved, surrendered, timeout) |
| `limit` | int | 50 | 최대 조회 수 (max: 100) |
| `offset` | int | 0 | 건너뛸 수 |

**Response:**
```json
{
  "status": "ok",
  "archives": [
    {
      "id": 1,
      "sessionId": "sess123",
      "chatId": "room456",
      "puzzleId": 5,
      "questionCount": 15,
      "hintsUsed": 2,
      "result": "solved",
      "startedAt": "2026-01-02T10:00:00Z",
      "completedAt": "2026-01-02T10:30:00Z"
    }
  ],
  "total": 100,
  "limit": 50,
  "offset": 0
}
```

---

## 에러 응답 형식

모든 에러는 표준 형식으로 반환됩니다:

```json
{
  "error": "ERROR_CODE",
  "message": "human-readable error message"
}
```

### 에러 코드

| 코드 | HTTP 상태 | 설명 |
|:---|:---|:---|
| `INVALID_REQUEST` | 400 | 잘못된 요청 (필수 파라미터 누락 등) |
| `SESSION_NOT_FOUND` | 404 | 세션을 찾을 수 없음 |
| `PUZZLE_NOT_FOUND` | 404 | 퍼즐을 찾을 수 없음 |
| `GAME_NOT_FOUND` | 404 | 게임을 찾을 수 없음 |
| `SYNONYM_NOT_FOUND` | 404 | 동의어를 찾을 수 없음 |
| `INTERNAL_ERROR` | 500 | 내부 서버 오류 |

---

## 변경 이력

| 날짜 | 버전 | 변경 내용 |
|:---|:---|:---|
| 2026-01-03 | 1.2.0 | TwentyQ Phase 5: 세션 상세조회, GM 힌트 주입, 세션 정리, 유저 통계 관리, 오디트/리펀드 로그 조회 |
| 2026-01-03 | 1.1.0 | TwentyQ 추가 API (게임 상세, 동의어 삭제, 카테고리 통계, 닉네임) 추가 |
| 2026-01-03 | 1.1.0 | TurtleSoup Puzzle CMS, Archives API 전체 구현 |
| 2026-01-02 | 1.0.0 | 최초 작성 |
