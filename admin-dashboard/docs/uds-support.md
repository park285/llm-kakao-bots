# Unix Domain Socket (UDS) 지원

## 개요

gRPC 서버와 클라이언트 간 통신에서 TCP 외에 Unix Domain Socket (UDS)을 지원합니다.
동일 호스트/컨테이너 네트워크 내에서 성능 향상과 보안 강화를 제공합니다.

## 지원 대상

| 서비스 | UDS 지원 | 설명 |
|--------|----------|------|
| **gRPC (LLM ↔ Bot)** | ✅ | `mcp-llm-server` ↔ `game-bot-go` 간 gRPC 통신 |
| **Valkey Cache** | ✅ | 세션/상태 캐시 저장소 |
| **PostgreSQL** | ✅ | 메인 데이터베이스 연결 |
| **Valkey MQ** | ❌ | Iris (redroid)가 외부에 있어 TCP 유지 필요 |

## 동작 모드

### Dual-Mode (TCP + UDS 병행)

모든 서비스는 TCP와 UDS를 동시에 지원합니다:

```
┌─────────────────────────────────────────────────────────────┐
│                   Docker Compose Network                     │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│  ┌──────────────┐     ┌─────────────────┐                   │
│  │   postgres   │     │  valkey-cache   │                   │
│  │ TCP:5432     │     │ TCP:6379        │                   │
│  │ UDS:/var/run/│     │ UDS:/var/run/   │                   │
│  │   postgresql │     │   valkey/*.sock │                   │
│  └──────┬───────┘     └────────┬────────┘                   │
│         │                      │                             │
│    ┌────┴──────────────────────┴────┐                       │
│    │        Shared tmpfs volumes     │                       │
│    │  pg-socket, valkey-cache-socket │                       │
│    │       grpc-socket               │                       │
│    └───────────┬────────────────────┘                       │
│                │                                             │
│  ┌─────────────┴─────────────┐                              │
│  │      mcp-llm-server       │                              │
│  │  gRPC TCP:40528 + UDS     │──────────┐                   │
│  │  → DB via UDS             │          │                   │
│  │  → Cache via UDS          │          │                   │
│  └───────────┬───────────────┘          │                   │
│              │ UDS                       │                   │
│  ┌───────────┴───────────────┐          │ TCP               │
│  │   twentyq-bot / turtle-   │          │                   │
│  │   soup-bot                │          │                   │
│  │  → gRPC via UDS           │          │                   │
│  │  → DB via UDS             │          ▼                   │
│  │  → Cache via UDS          │    [외부 grpcurl]            │
│  └───────────────────────────┘                              │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

## 환경 변수

### gRPC 서버 (mcp-llm-server)

| 환경 변수 | 기본값 | 설명 |
|----------|-------|------|
| `GRPC_HOST` | `127.0.0.1` | TCP 바인딩 호스트 |
| `GRPC_PORT` | `40528` | TCP 포트 |
| `GRPC_ENABLED` | `true` | gRPC 서버 활성화 |
| `GRPC_SOCKET_PATH` | (비어있음) | UDS 파일 경로. 비어있으면 TCP만 사용 |

### gRPC 클라이언트 (game-bot-go)

| 환경 변수 | 예시 값 | 설명 |
|----------|--------|------|
| `LLM_BASE_URL` | `grpc://mcp-llm-server:40528` | TCP 모드 |
| `LLM_BASE_URL` | `unix:///var/run/grpc/llm.sock` | UDS 모드 |

### PostgreSQL

| 환경 변수 | 예시 값 | 설명 |
|----------|--------|------|
| `DB_HOST` | `postgres` | TCP 호스트 (fallback) |
| `DB_PORT` | `5432` | TCP 포트 (fallback) |
| `DB_SOCKET_PATH` | `/var/run/postgresql` | UDS 디렉터리. 설정되면 UDS 우선 |
| `POSTGRES_SOCKET_PATH` | `/var/run/postgresql` | hololive-bot용 (동일) |

### Valkey Cache

| 환경 변수 | 예시 값 | 설명 |
|----------|--------|------|
| `CACHE_HOST` | `valkey-cache` | TCP 호스트 (fallback) |
| `CACHE_PORT` | `6379` | TCP 포트 (fallback) |
| `CACHE_SOCKET_PATH` | `/var/run/valkey/valkey-cache.sock` | UDS 경로. 설정되면 UDS 우선 |

## Docker Compose 설정

### 볼륨 정의

```yaml
volumes:
  grpc-socket:
    driver: local
    driver_opts:
      type: tmpfs
      device: tmpfs
      o: size=1m,mode=0777,uid=1000
  valkey-cache-socket:
    driver: local
    driver_opts:
      type: tmpfs
      device: tmpfs
      o: size=1m,mode=0777
  pg-socket:
    driver: local
    driver_opts:
      type: tmpfs
      device: tmpfs
      o: size=1m,mode=0777
```

### PostgreSQL 설정

```yaml
services:
  postgres:
    volumes:
      - pg-data-v18:/var/lib/postgresql/data
      - pg-socket:/var/run/postgresql  # 소켓 볼륨 마운트
```

> PostgreSQL은 기본적으로 `/var/run/postgresql` 디렉터리에 소켓 파일을 생성합니다.

### 클라이언트 설정

```yaml
services:
  mcp-llm-server:
    volumes:
      - pg-socket:/var/run/postgresql:ro
    environment:
      DB_SOCKET_PATH: /var/run/postgresql
      
  twentyq-bot:
    volumes:
      - grpc-socket:/var/run/grpc:ro
      - valkey-cache-socket:/var/run/valkey:ro
      - pg-socket:/var/run/postgresql:ro
    environment:
      LLM_BASE_URL: unix:///var/run/grpc/llm.sock
      CACHE_SOCKET_PATH: /var/run/valkey/valkey-cache.sock
      DB_SOCKET_PATH: /var/run/postgresql
```

## 장점

| 항목 | 설명 |
|-----|------|
| **성능** | 커널 TCP 스택 우회로 latency 10-30% 감소 |
| **보안** | 파일 시스템 권한으로 접근 제어 |
| **리소스** | 포트 소진 문제 없음, 파일 디스크립터만 사용 |
| **오버헤드** | No checksums, no routing, no port management |

## 제약사항

| 항목 | 설명 |
|-----|------|
| **동일 호스트 필수** | UDS는 같은 머신/컨테이너 namespace에서만 동작 |
| **Docker 볼륨** | 소켓 파일 공유를 위한 shared volume 필요 |
| **디버깅 어려움** | `grpcurl`, wireshark 등 TCP 기반 도구 사용 불가 |
| **외부 접근 불가** | Iris (redroid) 등 외부 컨테이너는 TCP 사용 필요 |

## 권장 사용 패턴

### 프로덕션 (컨테이너 간 통신)

- **서버**: TCP + UDS 모두 활성화 (Dual-Mode)
- **내부 클라이언트**: UDS 사용 (`unix://`, `SocketPath`)
- **외부 디버깅**: TCP 사용 (`grpc://`, Host:Port)

### 개발 환경

- TCP만 사용 권장 (디버깅 편의성)
- `*_SOCKET_PATH` 환경 변수 생략

## 트러블슈팅

### 소켓 파일 권한 오류

```bash
# 소켓 파일 권한 확인
ls -la /var/run/grpc/
ls -la /var/run/postgresql/
ls -la /var/run/valkey/

# 권한 수정 (필요시)
chmod 0660 /var/run/grpc/llm.sock
```

### PostgreSQL 소켓 연결 확인

```bash
# 컨테이너 내에서 소켓 파일 확인
docker exec llm-postgres ls -la /var/run/postgresql/

# 소켓으로 직접 연결 테스트
docker exec llm-postgres psql -h /var/run/postgresql -U twentyq_app -d twentyq -c "SELECT 1"
```

### 연결 실패

```bash
# 소켓 파일 존재 확인
test -S /var/run/grpc/llm.sock && echo "Socket exists" || echo "Socket not found"

# 서버 프로세스 확인
ss -lx | grep llm.sock
```

## 변경 내역

| 날짜 | 버전 | 설명 |
|-----|------|------|
| 2026-01-01 | 1.1.0 | PostgreSQL, Valkey Cache UDS Dual-Mode 지원 추가 |
| 2026-01-01 | 1.0.0 | 초기 gRPC UDS 지원 추가 |
