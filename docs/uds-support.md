# Unix Domain Socket (UDS) 지원

## 개요

gRPC 서버와 클라이언트 간 통신에서 TCP 외에 Unix Domain Socket (UDS)을 지원합니다.
동일 호스트/컨테이너 네트워크 내에서 성능 향상과 보안 강화를 제공합니다.

## 동작 모드

### Dual-Mode (TCP + UDS 병행)

서버는 TCP와 UDS를 동시에 listen할 수 있습니다:

```
┌─────────────────────────────────────────────────────────────┐
│                     mcp-llm-server                          │
├─────────────────────────────────────────────────────────────┤
│  TCP Listener:  0.0.0.0:40528                               │
│  UDS Listener:  /var/run/grpc/llm.sock                      │
└─────────────────────────────────────────────────────────────┘
                    │                    │
          ┌─────────┘                    └─────────┐
          ▼                                        ▼
   [외부 클라이언트]                        [내부 컨테이너]
   grpcurl, 디버깅                         twentyq-bot
                                           turtle-soup-bot
```

## 환경 변수

### 서버 (mcp-llm-server)

| 환경 변수 | 기본값 | 설명 |
|----------|-------|------|
| `GRPC_HOST` | `127.0.0.1` | TCP 바인딩 호스트 |
| `GRPC_PORT` | `40528` | TCP 포트 |
| `GRPC_ENABLED` | `true` | gRPC 서버 활성화 |
| `GRPC_SOCKET_PATH` | (비어있음) | UDS 파일 경로. 비어있으면 TCP만 사용 |

### 클라이언트 (game-bot-go)

| 환경 변수 | 예시 값 | 설명 |
|----------|--------|------|
| `LLM_BASE_URL` | `grpc://mcp-llm-server:40528` | TCP 모드 |
| `LLM_BASE_URL` | `unix:///var/run/grpc/llm.sock` | UDS 모드 |

## URL 스킴

클라이언트는 URL 스킴으로 통신 방식을 결정합니다:

| 스킴 | 예시 | 설명 |
|-----|------|------|
| `grpc://` | `grpc://mcp-llm-server:40528` | TCP 연결 |
| `unix://` | `unix:///var/run/grpc/llm.sock` | UDS 연결 |

> **참고**: `unix://` 뒤에 슬래시 3개가 필요합니다 (`unix://` + `/var/run/...`)

## Docker Compose 설정

### 소켓 볼륨 정의

```yaml
volumes:
  grpc-socket:
    driver: local
    driver_opts:
      type: tmpfs
      device: tmpfs
      o: size=1m,mode=0755
```

### 서버 설정

```yaml
services:
  mcp-llm-server:
    volumes:
      - grpc-socket:/var/run/grpc
    environment:
      GRPC_HOST: 0.0.0.0
      GRPC_PORT: 40528
      GRPC_ENABLED: "true"
      GRPC_SOCKET_PATH: /var/run/grpc/llm.sock
```

### 클라이언트 설정

```yaml
services:
  twentyq-bot:
    volumes:
      - grpc-socket:/var/run/grpc:ro
    environment:
      LLM_BASE_URL: unix:///var/run/grpc/llm.sock
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

## 권장 사용 패턴

### 프로덕션 (컨테이너 간 통신)

- **서버**: TCP + UDS 모두 활성화
- **내부 클라이언트**: UDS 사용 (`unix://`)
- **외부 디버깅**: TCP 사용 (`grpc://`)

### 개발 환경

- TCP만 사용 권장 (디버깅 편의성)
- `GRPC_SOCKET_PATH` 환경 변수 생략

## 소켓 파일 경로 규칙

표준 런타임 경로 사용:

```
/var/run/grpc/llm.sock
```

- `/var/run/`: 리눅스 표준 런타임 디렉터리
- `grpc/`: gRPC 소켓 전용 서브디렉터리
- `llm.sock`: 서비스별 소켓 파일명

## 트러블슈팅

### 소켓 파일 권한 오류

```bash
# 소켓 파일 권한 확인
ls -la /var/run/grpc/

# 권한 수정 (필요시)
chmod 0660 /var/run/grpc/llm.sock
```

### 소켓 파일이 남아있는 경우

서버 비정상 종료 시 소켓 파일이 남을 수 있습니다.
서버 시작 시 기존 소켓 파일을 자동으로 삭제합니다.

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
| 2026-01-01 | 1.0.0 | 초기 UDS 지원 추가 |
