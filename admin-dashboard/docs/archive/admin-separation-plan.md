# Admin 분리 계획

> 작성일: 2026-01-02  
> 상태: **Phase 4d 진행 중, Phase 4c 완료**  
> 우선순위: **최우선**

---

## 1. 목표

현재 `hololive-kakao-bot-go`에 포함된 Admin 기능을 독립 서비스로 분리:
- **admin-backend-go/backend**: 공통 백엔드 (인증, Docker, Logs, Traces)
- **admin-backend-go/frontend**: 프론트엔드 (React SPA)

---

## 2. 현재 상태 (Phase 4d 진행 중)

```
admin-backend-go/              # 생성 완료
├── backend/                   # Go 백엔드
│   ├── cmd/admin/main.go
│   ├── internal/
│   │   ├── auth/              # 세션 관리, Rate Limiting
│   │   ├── bootstrap/         # 앱 초기화
│   │   ├── config/            # 설정
│   │   ├── docker/            # Docker 관리
│   │   ├── logging/           # 통합 로깅 (tint + lumberjack + OTel)
│   │   ├── logs/              # 시스템 로그 읽기
│   │   ├── proxy/             # 봇 프록시
│   │   ├── server/            # HTTP 서버 + SSR
│   │   ├── ssr/               # SSR 데이터 주입
│   │   ├── telemetry/         # OpenTelemetry 통합
│   │   └── traces/            # Jaeger 클라이언트
│   ├── Makefile
│   ├── VERSION
│   └── go.mod
├── frontend/                  # 복사 완료 (from hololive-bot/admin-ui)
│   ├── src/
│   ├── package.json
│   └── vite.config.ts
├── Dockerfile                 # 통합 빌드 (frontend + backend)
└── README.md
```

---

## 3. 분리 진행 상황

### Phase 4a: admin-backend 생성 완료

| 순서 | 작업 | 상태 |
|------|------|------|
| 1 | `admin-backend-go/backend/` 디렉토리 생성 | 완료 |
| 2 | `go.mod` 초기화 | 완료 |
| 3 | `internal/auth/` - 세션 관리 | 완료 |
| 4 | `internal/docker/` - Docker 서비스 | 완료 |
| 5 | `internal/logs/` - 로그 서비스 | 완료 |
| 6 | `internal/traces/` - Jaeger 클라이언트 | 완료 |
| 7 | `internal/proxy/` - 봇 프록시 | 완료 |
| 8 | `internal/server/` - HTTP 서버 | 완료 |
| 9 | `cmd/admin/main.go` - 엔트리포인트 | 완료 |
| 10 | `Dockerfile` 생성 | 완료 |
| 11 | `Makefile` 생성 (시맨틱 버전 관리) | 완료 |
| 12 | 빌드 + lint 통과 | 완료 |

### Phase 4b: admin-ui 복사 완료

| 순서 | 작업 | 상태 |
|------|------|------|
| 1 | `frontend/` 디렉토리로 복사 | 완료 |
| 2 | `node_modules`, `dist` 제거 | 완료 |

### Phase 4c: hololive-bot 정리 **완료**

| 순서 | 작업 | 상태 |
|------|------|------|
| 1 | `admin-ui/` 디렉토리 전체 삭제 | 완료 |
| 2 | `internal/service/docker/` 삭제 | 완료 |
| 3 | `internal/service/jaeger/` 삭제 | 완료 |
| 4 | `internal/server/admin_docker.go` 삭제 | 완료 |
| 5 | `internal/server/admin_traces.go` 삭제 | 완료 |
| 6 | `internal/server/admin_syslogs.go` 삭제 | 완료 |
| 7 | `internal/server/ssr_data.go` 삭제 | 완료 |
| 8 | `internal/app/admin_router.go` 수정 (도메인 전용으로) | 완료 |
| 9 | `internal/app/providers.go` 수정 (Docker/Jaeger 제거) | 완료 |
| 10 | `internal/app/bootstrap.go` 수정 | 완료 |
| 11 | Dockerfile에서 frontend-builder 스테이지 제거 | 완료 |
| 12 | 빌드 테스트 | 완료 |

### Phase 4d: docker-compose 업데이트

| 순서 | 작업 | 상태 |
|------|------|------|
| 1 | `admin-backend` 서비스 추가 | 완료 |
| 2 | Cloudflare Tunnel 설정 업데이트 (수동) | [ ] |
| 3 | 통합 테스트 | [ ] |

---

## 4. API 경로 변경

### After (분리 완료 후)
```
# admin-backend (공통)
/admin/api/auth/login
/admin/api/auth/logout
/admin/api/auth/heartbeat
/admin/api/docker/*
/admin/api/logs/*
/admin/api/traces/*

# admin-backend → hololive-bot (프록시)
/admin/api/holo/members/*
/admin/api/holo/alarms/*
/admin/api/holo/rooms/*
/admin/api/holo/streams/*
/admin/api/holo/milestones/*
/admin/api/holo/settings/*

# admin-backend → twentyq-bot (프록시)
/admin/api/twentyq/*

# admin-backend → turtle-soup-bot (프록시)
/admin/api/turtle/*
```

---

## 5. 버전 관리

모든 Go 프로젝트에 시맨틱 버전 관리 표준 적용 완료:

| 프로젝트 | VERSION 파일 | 상태 |
|----------|-------------|------|
| admin-backend-go | 1.0.0 | 완료 |
| hololive-kakao-bot-go | 1.0.0 | 완료 |
| game-bot-go | 1.0.0 | 완료 |
| mcp-llm-server-go | 1.0.0 | 완료 |

---

## 6. 라이브러리 통일

### 로깅 (logging)
- **tint**: 컬러 slog 핸들러
- **lumberjack**: 로그 로테이션
- **combined.log**: 통합 로그 파일
- **OTel 상관관계**: trace_id/span_id 자동 추가

### 텔레메트리 (telemetry)
- **OpenTelemetry SDK**: TracerProvider
- **OTLP gRPC Exporter**: Jaeger 4317
- **ParentBased Sampler**: 분산 추적 연속성 보장

---

## 7. 다음 단계

1. **문서 정리**: API 경로/참조 최신화
2. **Cloudflare Tunnel 업데이트**: admin.capu.blog → admin-backend:30090
3. **통합 테스트**: 전체 시스템 동작 확인
4. **배포**: docker compose up -d --build admin-backend

---

## 8. 의존성 다이어그램

```
Cloudflare Tunnel
       ↓
   admin-backend (포트 30090)
       ├── valkey-cache (세션)
       ├── docker-proxy (컨테이너 관리)
       ├── jaeger (트레이스)
       └── logs volume
       
admin-backend (프록시)
       ├── hololive-bot:30001
       ├── twentyq-bot:30081
       └── turtle-soup-bot:30082
```
