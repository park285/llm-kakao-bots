# Admin Dashboard

관리 대시보드 - 백엔드(Go)와 프론트엔드(React)를 포함합니다.

## 구조

```
admin-dashboard/
├── backend/          # Go 백엔드
│   ├── cmd/admin/    # 엔트리포인트
│   ├── internal/     # 내부 패키지
│   │   ├── auth/     # 인증 및 세션
│   │   ├── bootstrap/# 앱 초기화
│   │   ├── config/   # 설정
│   │   ├── docker/   # Docker 관리
│   │   ├── logs/     # 시스템 로그
│   │   ├── proxy/    # 봇 프록시
│   │   ├── server/   # HTTP 서버
│   │   └── traces/   # Jaeger 클라이언트
│   ├── Dockerfile
│   └── go.mod
└── frontend/         # React 프론트엔드
    ├── src/
    ├── package.json
    └── vite.config.ts
```

## 백엔드

### 빌드

```bash
cd backend
go build -tags=go_json -o admin ./cmd/admin
```

### 환경 변수

| 변수 | 설명 | 기본값 |
|------|------|--------|
| `PORT` | HTTP 포트 | `30090` |
| `VALKEY_URL` | Valkey 주소 | `valkey-cache:6379` |
| `JAEGER_QUERY_URL` | Jaeger Query API | `http://jaeger:16686` |
| `DOCKER_HOST` | Docker 데몬 | `tcp://docker-proxy:2375` |
| `LOG_DIR` | 로그 디렉토리 | `/app/logs` |
| `ADMIN_USER` | 관리자 ID | `admin` |
| `ADMIN_PASS_HASH` | 비밀번호 bcrypt 해시 | - |
| `SESSION_SECRET` | 세션 서명 키 | - |
| `METRICS_API_KEY` | Prometheus `/metrics` 보호 키 (Bearer 또는 `X-API-Key`) | - |
| `OTEL_ENABLED` | OpenTelemetry 활성화 | `false` |
| `OTEL_SERVICE_NAME` | OpenTelemetry 서비스명 | `admin-dashboard` |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | OTLP 엔드포인트 (Jaeger) | `jaeger:4317` |
| `OTEL_EXPORTER_OTLP_INSECURE` | OTLP insecure 사용 여부 | `true` |
| `OTEL_SAMPLE_RATE` | 샘플링 비율 (0.0~1.0) | `1.0` |

> 운영(`docker-compose.prod.yml`)에서는 `METRICS_API_KEY` 설정을 필수로 강제하며,
> Prometheus 스크랩을 위해 `./secrets/admin-dashboard-metrics.token` 파일을 `METRICS_API_KEY`로 자동 생성합니다.

## 프론트엔드

### 개발 서버

```bash
cd frontend
npm install
npm run dev
```

### 빌드

```bash
cd frontend
npm run build
```

## Docker

```bash
# 백엔드
docker build -t admin-dashboard -f backend/Dockerfile backend/

# 프론트엔드
docker build -t admin-ui -f frontend/Dockerfile frontend/
```

## API 경로

### 공통 (admin-dashboard)
- `POST /admin/api/auth/login` - 로그인
- `POST /admin/api/auth/logout` - 로그아웃
- `POST /admin/api/auth/heartbeat` - 세션 갱신
- `GET /admin/api/docker/*` - Docker 관리
- `GET /admin/api/logs/*` - 시스템 로그
- `GET /admin/api/traces/*` - Jaeger 프록시

### 도메인별 (프록시)
- `/admin/api/holo/*` → hololive-bot
- `/admin/api/twentyq/*` → twentyq-bot
- `/admin/api/turtle/*` → turtle-soup-bot
