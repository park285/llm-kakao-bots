# Admin UI 통합 봇 관리 페이지 승격 계획

> 작성일: 2026-01-02  
> 최종 수정: 2026-01-02  
> 상태: **Phase 4a 완료 (admin-backend 분리 + OpenAPI Pipeline)**

---

## 🚨 Critical Review 요약

### 핵심 리스크 및 대응

| 리스크 | 설명 | 대응 |
|--------|------|------|
| **SPOF (단일 실패점)** | hololive-bot 장애 시 전체 Admin 마비 | Phase 4 (인프라 분리) **필수** 격상 |
| **데이터 휘발성** | TurtleSoup Redis 데이터 재시작 시 소실 | PostgreSQL 아카이빙 선행 구현 |
| **God Container** | Admin UI가 hololive-bot에 포함되어 배포 비효율 | 독립 컨테이너 + Nginx Gateway |

### 수정된 핵심 결정

- ✅ **Phase 4 (인프라 분리)**: 선택 → **필수** 격상
- ✅ **Nginx Gateway**: 경로 기반 라우팅으로 봇별 독립성 보장
- ✅ **데이터 영속성**: TurtleSoup 게임 결과 PostgreSQL 아카이빙 선행

---

## 1. 현재 상태 분석

### 1.1 프로젝트 위치
```
/home/kapu/gemini/llm/hololive-kakao-bot-go/admin-ui/
```

### 1.2 현재 브랜딩
| 항목 | 현재 값 |
|------|---------|
| 타이틀 | "Hololive Kakao Bot Admin UI" |
| 사이드바 로고 | "Hololive Bot" |
| 배너 타이틀 | "Hololive Bot Console" |
| 헤더 서브타이틀 | "Hololive Kakao Bot Management System" |
| 도메인 | admin.capu.blog |

### 1.3 관리 대상 봇 서비스
| 서비스명 | 컨테이너명 | 용도 |
|----------|------------|------|
| hololive-bot | hololive-kakao-bot-go | 홀로라이브 VTuber 방송 알림 봇 |
| twentyq-bot | twentyq-bot | 스무고개 게임 봇 |
| turtle-soup-bot | turtle-soup-bot | 거북이 수프 (상황 추리) 게임 봇 |

### 1.4 현재 기능 현황

#### 홀로라이브 봇 전용 기능
| 탭 | 기능 | 비고 |
|----|------|------|
| 대시보드 (stats) | 멤버/알람/방 통계, 시스템 모니터링 | HoloBot 전용 |
| 방송 현황 (streams) | 라이브/예정 스트림 | HoloBot 전용 |
| 멤버 관리 (members) | VTuber 멤버 CRUD, 별칭, 채널 연동 | HoloBot 전용 |
| 마일스톤 (milestones) | 구독자 마일스톤 달성 추적 | HoloBot 전용 |
| 알람 관리 (alarms) | 방송 알림 구독자 관리 | HoloBot 전용 |
| 방 관리 (rooms) | 채팅방 ACL (화이트리스트) | HoloBot 전용 |

#### 공통 인프라 기능 (이미 통합됨)
| 탭 | 기능 | 비고 |
|----|------|------|
| 로그 (logs) | 시스템 로그 + Docker 컨테이너 실시간 로그 | 전체 봇 대상 |
| Traces (traces) | Jaeger 분산 트레이싱, SPM 메트릭 | 전체 서비스 대상 |
| 설정 (settings) | 알람 설정 + Docker 컨테이너 관리 | 전체 컨테이너 대상 |

---

## 2. 승격 목표

**"통합 봇 관리 어드민 페이지"** 로 승격하여 모든 봇 서비스를 단일 대시보드에서 관리

### 2.1 목표 브랜딩
| 항목 | 변경 후 |
|------|---------|
| 타이틀 | "Bot Admin Console"|
| 사이드바 로고 | "Bot Admin" |
| 배너 타이틀 | "Bot Management Console" |
| 헤더 서브타이틀 | "Unified Bot Management System" |

### 2.2 관리 범위 확장
```
┌────────────────────────────────────────────────────┐
│              Bot Admin Console                      │
├────────────────────────────────────────────────────┤
│  ┌─────────────┐ ┌─────────────┐ ┌─────────────┐   │
│  │ Hololive Bot│ │ TwentyQ Bot │ │TurtleSoup Bot│   │
│  │  (방송 알림) │ │ (스무고개)  │ │(상황 추리)  │   │
│  └─────────────┘ └─────────────┘ └─────────────┘   │
│                                                     │
│  ┌──────────────────────────────────────────────┐  │
│  │  공통 기능: 로그, Traces, Docker 관리        │  │
│  └──────────────────────────────────────────────┘  │
└────────────────────────────────────────────────────┘
```

---

## 3. 승격 작업 계획

### Phase 1: 브랜딩 변경 (Low Effort)
**예상 소요: 30분**

#### 3.1.1 파일 변경 목록

| 파일 | 변경 내용 |
|------|----------|
| `index.html` | `<title>` 태그with SEO 메타 태그 수정 |
| `AppLayout.tsx` | 사이드바 로고 텍스트, 헤더 서브타이틀 변경 |
| `StatsTab.tsx` | 배너 타이틀 변경 |
| `README.md` | 프로젝트 설명 업데이트 |

#### 3.1.2 변경 상세

**`index.html`**
```html
<!-- Before -->
<title>Hololive Bot Admin</title>

<!-- After -->
<title>Bot Admin Console</title>
```

**`AppLayout.tsx` (line 80-82)**
```tsx
// Before
<span className="text-lg font-bold text-slate-800 tracking-tight">
    Hololive Bot
</span>

// After
<span className="text-lg font-bold text-slate-800 tracking-tight">
    Bot Admin
</span>
```

**`AppLayout.tsx` (line 176-178)**
```tsx
// Before
<p className="text-xs text-slate-400 font-medium mt-0.5">
    Hololive Kakao Bot Management System
</p>

// After
<p className="text-xs text-slate-400 font-medium mt-0.5">
    Unified Bot Management System
</p>
```

**`StatsTab.tsx` (line 72-74)**
```tsx
// Before
<h1 className="text-3xl font-bold text-slate-800 tracking-tight">
    Hololive Bot Console
</h1>

// After
<h1 className="text-3xl font-bold text-slate-800 tracking-tight">
    Bot Management Console
</h1>
```

---

### Phase 2: 네비게이션 재구성 (Medium Effort)
**예상 소요: 1-2시간**

#### 3.2.1 봇별 섹션 분리

현재 navItems 구조:
```tsx
const navItems = [
    { id: 'stats', label: '대시보드', ... },
    { id: 'streams', label: '방송 현황', ... },
    { id: 'members', label: '멤버 관리', ... },
    { id: 'milestones', label: '마일스톤', ... },
    { id: 'alarms', label: '알람 관리', ... },
    { id: 'rooms', label: '방 관리', ... },
    { id: 'traces', label: 'Traces', ... },
    { id: 'logs', label: '로그', ... },
    { id: 'settings', label: '설정', ... },
]
```

제안 구조 (Option A - 그룹화):
```tsx
const navGroups = [
    {
        label: 'Overview',
        items: [
            { id: 'stats', label: '통합 대시보드', icon: LayoutDashboard },
        ]
    },
    {
        label: 'Hololive Bot',
        items: [
            { id: 'streams', label: '방송 현황', icon: Radio },
            { id: 'members', label: '멤버 관리', icon: Users },
            { id: 'milestones', label: '마일스톤', icon: Trophy },
            { id: 'alarms', label: '알람 관리', icon: Bell },
            { id: 'rooms', label: '방 관리', icon: MessageSquare },
        ]
    },
    {
        label: 'Game Bots',  // TwentyQ, TurtleSoup
        items: [
            // 향후 게임 봇별 관리 기능 추가 시
        ]
    },
    {
        label: 'Infrastructure',
        items: [
            { id: 'traces', label: 'Traces', icon: Activity },
            { id: 'logs', label: '로그', icon: ScrollText },
            { id: 'settings', label: '설정', icon: Settings },
        ]
    },
]
```

---

### Phase 3: 통합 대시보드 개선 (High Effort)
**예상 소요: 3-5시간**

#### 3.3.1 봇별 상태 카드 추가

현재 StatsTab은 Hololive Bot 전용입니다. 통합 대시보드로 확장:

```tsx
// 봇별 상태 표시
const botServices = [
    {
        name: 'Hololive Bot',
        container: 'hololive-kakao-bot-go',
        icon: <Play />,
        color: 'sky',
        stats: { members, alarms, rooms }
    },
    {
        name: 'TwentyQ Bot',
        container: 'twentyq-bot',
        icon: <HelpCircle />,
        color: 'purple',
        stats: { activeSessions, totalGames }  // 신규 API 필요
    },
    {
        name: 'TurtleSoup Bot',
        container: 'turtle-soup-bot',
        icon: <Soup />,  // Lucide에서 적절한 아이콘
        color: 'emerald',
        stats: { activeSessions, totalGames }  // 신규 API 필요
    },
]
```

#### 3.3.2 필요한 백엔드 API

Game Bot들의 상태를 조회하려면 각 봇에서 Admin API를 노출해야 합니다:

| 엔드포인트 | 용도 | 현재 상태 |
|------------|------|-----------|
| `GET /health` | 헬스 체크 | 이미 존재 |
| `GET /admin/stats` | 봇별 통계 | 신규 필요 |
| `GET /admin/sessions` | 활성 세션 목록 | 신규 필요 |

---

### Phase 4: 프로젝트 구조 변경 (Optional, High Effort)
**예상 소요: 4-8시간**

현재 admin-ui는 `hololive-kakao-bot-go/admin-ui`에 위치해 hololive-bot Docker 이미지에 포함됩니다.

#### Option A: 현재 위치 유지 (권장)
- 장점: 변경 최소화, Docker 빌드 변경 불필요
- 단점: 디렉토리 구조가 의미와 맞지 않음

#### Option B: 독립 프로젝트로 분리
```
/home/kapu/gemini/llm/
├── admin-ui/           # 새 위치
├── hololive-kakao-bot-go/
├── game-bot-go/
└── mcp-llm-server-go/
```
- 장점: 논리적 구조 개선

---

### Phase 5: 게임 봇 도메인 특화 관리 기능 (High Effort)
**예상 소요: 8-16시간** (백엔드 + 프론트엔드)

홀로라이브 봇처럼 게임 봇들(TwentyQ, TurtleSoup)도 도메인에 특화된 관리 기능을 제공합니다.

---

#### 5.1 TwentyQ Bot (스무고개) 관리 기능

##### 5.1.1 현재 데이터 모델 (PostgreSQL)

| 테이블 | 설명 | 주요 필드 |
|--------|------|-----------|
| `game_sessions` | 게임 세션 기록 | session_id, chat_id, category, result, question_count, hint_count, completed_at |
| `game_logs` | 참여자별 기록 | chat_id, user_id, sender, category, question_count, result, target |
| `user_stats` | 사용자 통계 집계 | total_games_started, total_games_completed, total_surrenders, best_score_* |
| `user_nickname_map` | 닉네임 매핑 | chat_id, user_id, last_sender |

##### 5.1.2 제안 관리 기능

**TwentyQTab.tsx** - 스무고개 관리 탭

| 섹션 | 기능 | 설명 |
|------|------|------|
| **대시보드** | 통계 요약 | 총 게임 수, 성공률, 평균 질문 수, 활성 세션 수 |
| **활성 세션** | 실시간 현황 | 현재 진행 중인 게임 목록, 강제 종료 기능 |
| **게임 기록** | 히스토리 | 최근 완료된 게임 목록 (필터: 채팅방, 결과, 카테고리) |
| **사용자 통계** | 리더보드 | 상위 플레이어, 최소 질문 기록 |
| **카테고리 관리** | 토픽 관리 | 카테고리별 사용 빈도, 성공률 분석 |

##### 5.1.3 🧠 지식 베이스 관리 (Dictionary CMS) - 심화 기능

AI 판정 오류 교정 및 게임 밸런싱을 위한 도구입니다.

**A. 동의어(Synonym) 매핑**
- 문제: 유저가 "스맛폰"이라 답했는데 AI가 모른다고 판정
- 해결: 관리자가 `스맛폰 = 스마트폰` 매핑을 추가하여 즉시 정답 처리

```
POST /admin/synonyms
{ "aliases": ["스맛폰", "손전화"], "canonical": "스마트폰" }

GET  /admin/synonyms?query=스맛폰
→ { "canonical": "스마트폰", "aliases": ["스맛폰", "손전화"] }
```

**B. 난이도 티어링**
- 각 정답 단어별 유저 승률 분석 → S/A/B/C 등급 자동 산정
- 초보자 방에는 쉬운 단어(C등급)만 출제되도록 밸런싱

```
GET  /admin/difficulty?minGames=10
→ [{ "target": "사과", "winRate": 0.85, "tier": "C" },
   { "target": "양자역학", "winRate": 0.12, "tier": "S" }]
```

##### 5.1.4 🕵️ 게임 리플레이 & 디버거

판정 논란 해결 및 CS 대응 도구입니다.

**A. 타임라인 뷰**
- 단순 텍스트 로그가 아닌 채팅방 형태의 UI로 게임 복기
- 질문 → AI 판정 → 유저 반응 흐름을 시각적으로 표현

**B. 판단 감사(Audit) & 판정 번복(Refund)**
- AI가 "아니오"라고 대답했을 때, 실제 판단이 옳았는지 관리자 검토
- 오판 확인 시 "판정 번복" → 유저 스탯 복구

```
POST /admin/games/{gameId}/audit
{ "questionIndex": 5, "verdict": "AI_WRONG", "reason": "동의어 미인식" }

POST /admin/games/{gameId}/refund
{ "userId": "user123", "restoreStats": true }
→ 해당 유저의 questionCount, wrongGuessCount 등 복구
```

##### 5.1.3 필요한 백엔드 Admin API

```
# twentyq-bot 서비스에 추가할 엔드포인트

GET  /admin/stats
     → { totalGames, completedGames, successRate, avgQuestions, activeSessions }

GET  /admin/sessions
     → [{ sessionId, chatId, category, questionCount, startedAt, status }]

DELETE /admin/sessions/{sessionId}
     → 강제 종료

GET  /admin/games?limit=50&offset=0&result=CORRECT&category=인물
     → [{ sessionId, chatId, result, questionCount, target, completedAt }]

GET  /admin/leaderboard?type=best_score&limit=10
     → [{ chatId, userId, sender, bestScoreQuestionCnt, target, achievedAt }]

GET  /admin/categories
     → [{ category, totalGames, successRate, avgQuestions }]
```

---

#### 5.2 TurtleSoup Bot (거북이 수프) 관리 기능

##### 5.2.1 현재 데이터 구조 (Valkey/Redis)

| 키 패턴 | 설명 | 데이터 |
|---------|------|--------|
| `tssession:{sessionId}` | 게임 세션 | GameState (puzzle, questionCount, hintsUsed, isSolved) |
| `tsdedup:{hash}` | 퍼즐 중복 방지 | Set of puzzle hashes |
| `tslock:{sessionId}` | 동시성 락 | Distributed lock |
| `tsvote:{sessionId}` | 항복 투표 | Vote state |

##### 5.2.2 제안 관리 기능

**TurtleSoupTab.tsx** - 거북이 수프 관리 탭

| 섹션 | 기능 | 설명 |
|------|------|------|
| **대시보드** | 통계 요약 | 총 게임 수, 해결률, 평균 질문 수, 평균 힌트 사용 |
| **활성 세션** | 실시간 현황 | 현재 진행 중인 게임 (채팅방, 시작 시간, 질문 수) |
| **퍼즐 관리** | 토픽 분석 | 카테고리/테마별 분포, 난이도별 해결률 |
| **세션 관리** | 유지보수 | 오래된 세션 정리, 강제 종료 |

##### 5.2.3 📝 시나리오 에디터 (Scenario CMS) - 심화 기능

시나리오(스토리)가 게임의 핵심입니다. DB나 JSON 직접 수정은 위험합니다.

**A. 시나리오 작성 폼**
- 문제(Scenario), 진상(Truth), 핵심 힌트를 위한 전용 에디터
- Markdown 지원, 미리보기

**B. 스포일러 방지 (Blur)**
- 관리자 화면에서도 '진상' 텍스트는 기본 흐림 처리
- 클릭해야만 표시 → 방송 송출 사고 방지

**C. 상태 관리 (Workflow)**
```
Draft (작성중) → Test (테스트) → Published (배포)
```
- 미완성 문제가 실서비스에 노출되는 것 방지
- 테스트 채팅방에서만 Draft 시나리오 사용 가능

```
POST /admin/scenarios
{ "title": "...", "scenario": "...", "truth": "...", "hints": [...], "status": "draft" }

PATCH /admin/scenarios/{id}/status
{ "status": "published" }

GET  /admin/scenarios?status=draft
→ [{ id, title, status, createdAt, author }]
```

##### 5.2.4 ⚡ 실시간 GM 개입 (God Mode) - 심화 기능

AI가 상황을 못 맞히거나(환각), 유저들이 답답해할 때 관리자가 직접 개입합니다.

**A. 힌트 주입 (Inject Hint)**
- 관리자가 작성한 텍스트를 봇이 말한 것처럼 채팅방에 전송
- "시스템 힌트" 표시로 구분

```
POST /admin/sessions/{sessionId}/inject
{ "type": "hint", "message": "핵심 단서: 날씨를 생각해보세요" }
→ 채팅방에 "[힌트] 날씨를 생각해보세요" 전송
```

**B. LLM 생각 엿보기 (Trace)**
- AI가 힌트/판정할 때의 내부 프롬프트/추론 로그(Chain of Thought) 실시간 확인
- Jaeger Trace와 연계

```
GET  /admin/sessions/{sessionId}/llm-trace
→ { "prompt": "...", "response": "...", "reasoning": "...", "latencyMs": 1234 }
```

##### 5.2.5 🗄️ 데이터 영속성 (PostgreSQL 아카이빙)

**⚠️ 선행 필수 작업** - Redis 휘발성 문제 해결

| 시점 | 데이터 | 저장소 |
|------|--------|--------|
| 게임 진행 중 | 세션 상태 | Redis (실시간 접근) |
| 게임 종료 시 | 게임 결과/통계 | **PostgreSQL** (영구 보존) |

```go
// 게임 종료 시 비동기 아카이빙
func (s *GameService) archiveToPostgres(ctx context.Context, state GameState) error {
    record := GameArchive{
        SessionID:     state.SessionID,
        ChatID:        state.ChatID,
        Category:      state.Puzzle.Category,
        Difficulty:    state.Puzzle.Difficulty,
        QuestionCount: state.QuestionCount,
        HintsUsed:     state.HintsUsed,
        IsSolved:      state.IsSolved,
        CompletedAt:   time.Now(),
    }
    return s.db.Create(&record).Error
}
```

##### 5.2.3 필요한 백엔드 Admin API

```
# turtle-soup-bot 서비스에 추가할 엔드포인트

GET  /admin/stats
     → { totalGames, solvedGames, solveRate, avgQuestions, avgHints, activeSessions }

GET  /admin/sessions
     → [{ sessionId, chatId, userId, category, difficulty, questionCount, hintsUsed, startedAt }]

DELETE /admin/sessions/{sessionId}
     → 강제 종료

GET  /admin/puzzles/stats
     → { byCategory: {...}, byDifficulty: {...}, byTheme: {...} }

POST /admin/sessions/cleanup?olderThan=24h
     → 오래된 세션 일괄 정리
```

---

#### 5.3 통합 게임 관리 페이지 구조

```
/dashboard/games                  → 게임 봇 통합 대시보드
/dashboard/games/twentyq          → TwentyQ 상세 관리
/dashboard/games/twentyq/sessions → 활성 세션
/dashboard/games/twentyq/history  → 게임 기록
/dashboard/games/twentyq/stats    → 통계/리더보드
/dashboard/games/turtlesoup       → TurtleSoup 상세 관리
/dashboard/games/turtlesoup/sessions
/dashboard/games/turtlesoup/puzzles
```

##### 5.3.1 제안 네비게이션 구조

```tsx
const navGroups = [
    {
        label: 'Overview',
        items: [
            { id: 'stats', label: '통합 대시보드', icon: LayoutDashboard },
        ]
    },
    {
        label: 'Hololive Bot',
        items: [
            { id: 'streams', label: '방송 현황', icon: Radio },
            { id: 'members', label: '멤버 관리', icon: Users },
            { id: 'milestones', label: '마일스톤', icon: Trophy },
            { id: 'alarms', label: '알람 관리', icon: Bell },
            { id: 'rooms', label: '방 관리', icon: MessageSquare },
        ]
    },
    {
        label: 'Game Bots',
        items: [
            { id: 'games', label: '게임 대시보드', icon: Gamepad2 },
            { id: 'twentyq', label: '스무고개', icon: HelpCircle },
            { id: 'turtlesoup', label: '거북이 수프', icon: Soup },
        ]
    },
    {
        label: 'Infrastructure',
        items: [
            { id: 'traces', label: 'Traces', icon: Activity },
            { id: 'logs', label: '로그', icon: ScrollText },
            { id: 'settings', label: '설정', icon: Settings },
        ]
    },
]
```

---

#### 5.4 구현 우선순위

| 순서 | 작업 | 소요 시간 | 의존성 |
|------|------|----------|--------|
| 1 | TwentyQ Admin API 구현 | 2-3시간 | DB 스키마 이미 존재 |
| 2 | TwentyQ 프론트엔드 탭 | 3-4시간 | API 필요 |
| 3 | TurtleSoup Admin API 구현 | 2-3시간 | Redis 패턴 사용 |
| 4 | TurtleSoup 프론트엔드 탭 | 3-4시간 | API 필요 |
| 5 | 통합 게임 대시보드 | 2-3시간 | 양쪽 API 필요 |

---

#### 5.5 API 프록시 고려사항

현재 Admin UI는 `hololive-bot`의 Admin API를 통해 모든 요청을 처리합니다.  
게임 봇 API를 추가하려면 두 가지 접근법이 있습니다:

**Option A: hololive-bot에서 프록시** (권장)
```
Admin UI → hololive-bot → twentyq-bot
                        → turtle-soup-bot
```
- 장점: 단일 API 엔드포인트, 인증 통합
- 단점: hololive-bot에 프록시 코드 추가 필요

**Option B: 직접 호출**
```
Admin UI → twentyq-bot (별도 인증)
        → turtle-soup-bot (별도 인증)
```
- 장점: 구현 간단
- 단점: CORS 설정, 인증 분산, 포트 노출 필요

---

## 4. 수정된 실행 계획 (Revised Roadmap)

> **admin-backend 신규 컨테이너**로 공통 백엔드 분리 확정.

| 순서 | Phase | 작업명 | 핵심 내용 | 필수/선택 | 상태 |
|------|-------|--------|----------|-----------|------|
| 1 | Phase 1 | 브랜딩 변경 | 타이틀변경 | **필수** | ⬜ |
| 2 | Phase 4a | **admin-backend 생성** | 인증, Docker, Logs, Traces 분리 | **필수** | ✅ 완료 |
| 2.1 | OpenAPI | **OpenAPI Pipeline** | swag + openapi-generator | **필수** | ✅ 완료 |
| 3 | Phase 4b | **admin-ui 분리** | 프론트엔드 독립 컨테이너 | **필수** | ⬜ |
| 4 | Phase 4c | **hololive-bot 정리** | 공통 코드 제거, /api/holo/* 추가 | **필수** | ⬜ |
| 5 | Backend | 게임 봇 Admin API | twentyq, turtle-soup에 `/admin/*` | **필수** | ✅ 완료 |
| 6 | Backend | 데이터 영속성 | TurtleSoup PostgreSQL 아카이빙 | **필수** (선행) | ⬜ |
| 7 | Phase 5 | CMS API 백엔드 | 동의어, 오디트, 리펀드 API | 권장 | ✅ 완료 |
| 8 | Phase 2 | 네비게이션 | 서비스별 메뉴 구성 (사이드바 그룹화) | 권장 | ⬜ |
| 9 | Phase 3 | 통합 대시보드 | 전체 봇 상태를 한눈에 보는 메인 화면 | 선택 | ⬜ |

### 4.1 인증 통합 전략

**Cloudflare Tunnel 유지** -

현재처럼 Cloudflare Tunnel을 통한 서빙을 유지합니다.  
admin-ui 컨테이너가 내부적으로 다른 봇들에게 프록시합니다.

```
                    ┌─────────────────────────────────────────┐
Cloudflare Tunnel   │                                         │
(admin.capu.blog)   │              Docker Network             │
        │           │                                         │
        ▼           │   ┌─────────────┐                       │
   admin-ui:     ───┼──►│ Static SPA  │                       │
        │           │   └─────────────┘                       │
        │           │         │                               │
        │           │   /admin/api/*                          │
        │           │         ▼                               │
        │           │   ┌─────────────┐   ┌─────────────┐     │
        │           │   │  hololive   │   │   twentyq   │     │
        │           │   │  :30001     │   │   :30081    │     │
        │           │   └─────────────┘   └─────────────┘     │
        │           │                     ┌─────────────┐     │
        │           │                     │ turtle-soup │     │
        │           │                     │   :30082    │     │
        │           │                     └─────────────┘     │
        │           └─────────────────────────────────────────┘
```

### 4.2 라우팅 구조 (hololive-bot 프록시)

현재 hololive-bot이 Admin UI를 호스팅하고 있으므로, 게임 봇 API도 hololive-bot에서 프록시:

```go
// hololive-bot/internal/admin/proxy.go
func RegisterGameBotProxies(mux *http.ServeMux, cfg ProxyConfig) {
    // TwentyQ Bot Admin API
    twentyqProxy := httputil.NewSingleHostReverseProxy(
        &url.URL{Scheme: "http", Host: "twentyq-bot:30081"},
    )
    mux.Handle("/admin/api/twentyq/", 
        http.StripPrefix("/admin/api/twentyq", twentyqProxy))
    
    // TurtleSoup Bot Admin API
    turtleProxy := httputil.NewSingleHostReverseProxy(
        &url.URL{Scheme: "http", Host: "turtle-soup-bot:30082"},
    )
    mux.Handle("/admin/api/turtle/", 
        http.StripPrefix("/admin/api/turtle", turtleProxy))
}
```

### 4.3 Shared Secret 인증

모든 봇 컨테이너가 공유하는 `SESSION_SECRET` (구: `ADMIN_SECRET_KEY`):

```yaml
# docker-compose.prod.yml
x-admin-secret: &admin-secret
  SESSION_SECRET: ${SESSION_SECRET:?required}

services:
  hololive-bot:
    environment:
      <<: *admin-secret
  twentyq-bot:
    environment:
      <<: *admin-secret
  turtle-soup-bot:
    environment:
      <<: *admin-secret
```

```go
// 각 봇의 Admin Middleware
func AdminAuthMiddleware(secret string) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            if r.Header.Get("X-Admin-Secret") != secret {
                http.Error(w, "Unauthorized", http.StatusUnauthorized)
                return
            }
            next.ServeHTTP(w, r)
        })
    }
}
```

---

## 4.4 백엔드 분리 전략 (확정)

> **상세 계획**: [admin-separation-plan.md](./admin-separation-plan.md)

**Option 2: 공통 백엔드도 분리 (admin-backend 신규 컨테이너)** ✅

### 요약

| 서비스 | 담당 |
|--------|------|
| **admin-backend** (신규) | 인증, Docker, Logs, Traces |
| **hololive-bot** | 멤버, 알람, 방, 스트림, 마일스톤 |
| **twentyq-bot** | 세션, 통계, 사전CMS |
| **turtle-soup-bot** | 세션, 퍼즐, 시나리오CMS |

---

## 5. 변경 영향 범위

### 5.1 변경 필요 파일 (Phase 1 기준)

| 파일 | 변경 유형 |
|------|----------|
| `index.html` | 브랜딩 |
| `src/layouts/AppLayout.tsx` | 브랜딩 |
| `src/components/StatsTab.tsx` | 브랜딩 |
| `README.md` | 문서화 |

### 5.2 영향 없음 → 수정됨

**Phase 4 (백엔드 분리) 시 변경 필요:**
- Docker 빌드 변경 없음
- 라우팅 변경 없음
- 인증 로직 변경 없음

---

## 6. 확정된 결정 사항

### 6.1 브랜드명
**"Bot Admin Console"** ✅ 확정

### 6.2 프로젝트 구조
**독립 프로젝트로 분리** ✅ 확정

```
/home/kapu/gemini/llm/
├── admin-ui/               # ← 새 위치 (독립 프로젝트)
├── hololive-kakao-bot-go/
├── game-bot-go/
└── mcp-llm-server-go/
```

### 6.3 분리 작업 체크리스트

| 순서 | 작업 | 상태 |
|------|------|------|
| 1 | admin-backend-go 프로젝트 생성 | ✅ 완료 |
| 2 | 인증, Docker, Logs, Traces 백엔드 이전 | ✅ 완료 |
| 3 | 봇 프록시 설정 (holo, twentyq, turtle) | ✅ 완료 |
| 4 | OpenAPI Pipeline 구축 (swag + openapi-generator) | ✅ 완료 |
| 5 | 라우터 도메인별 분리 | ✅ 완료 |
| 6 | admin-ui 디렉토리 이동 (`frontend/` 위치) | ✅ 완료 |
| 7 | **Game Bot Admin API 백엔드 구현** | ✅ 완료 |
| 8 | docker-compose.prod.yml에 admin-backend 서비스 추가 | [ ] |
| 9 | hololive-kakao-bot-go 공통 코드 제거 | [ ] |
| 10 | 브랜딩 변경 (Phase 1) | [ ] |
| 11 | 빌드 및 배포 테스트 | [ ] |

### 6.4 실행 범위 (미결정)

아래 중 선택 필요:
- [ ] Phase 1만 (브랜딩만 변경)
- [ ] Phase 1 + 2 (브랜딩 + 네비게이션)
- [ ] Phase 1 + 2 + 5 (브랜딩 + 네비게이션 + 게임 봇 관리)
- [ ] 전체 Phase (장기 계획)

---

## Appendix A: 현재 파일 구조

```
admin-ui/
├── src/
│   ├── api/              # API 클라이언트 (공통)
│   ├── components/
│   │   ├── dashboard/    # 대시보드 전용 (SystemStatsChart 등)
│   │   ├── docker/       # Docker 관리 (공통)
│   │   ├── traces/       # Jaeger 트레이싱 (공통)
│   │   ├── ui/           # 재사용 UI 컴포넌트
│   │   ├── StatsTab.tsx       # ← HoloBot 전용
│   │   ├── StreamsTab.tsx     # ← HoloBot 전용
│   │   ├── MembersTab.tsx     # ← HoloBot 전용
│   │   ├── MilestonesTab.tsx  # ← HoloBot 전용
│   │   ├── AlarmsTab.tsx      # ← HoloBot 전용
│   │   ├── RoomsTab.tsx       # ← HoloBot 전용
│   │   ├── LogsTab.tsx        # 공통
│   │   ├── TracesTab.tsx      # 공통
│   │   └── SettingsTab.tsx    # 공통
│   ├── layouts/
│   │   └── AppLayout.tsx
│   ├── pages/
│   │   └── LoginPage.tsx
│   └── ...
└── package.json
```

---

## Appendix B: 분리 후 Docker 구성

> **상세 계획**: [admin-separation-plan.md](./admin-separation-plan.md)

- Cloudflare Tunnel 유지 
- admin-backend + admin-ui 독립 컨테이너
- 각 봇별 도메인 전용 API
