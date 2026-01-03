# Admin UI 운영 가이드

## 개요

Admin UI의 Docker 컨테이너 관리 기능과 deunhealth(워치독) 간의 상호작용을 설명합니다.

---

## 🛡️ Docker Socket Proxy 아키텍처

봇이 직접 `docker.sock`에 접근하지 않고, `docker-socket-proxy`를 통해 **제한된 API만** 사용합니다.

```
┌─────────────────┐     ┌──────────────────┐     ┌─────────────────┐
│  hololive-bot   │────▶│  docker-proxy    │────▶│  Docker Engine  │
│  (Admin UI)     │     │  (Allowlist)     │     │                 │
└─────────────────┘     └──────────────────┘     └─────────────────┘
                              │
┌─────────────────┐           │
│   deunhealth    │───────────┘
│   (워치독)       │
└─────────────────┘
```

### 허용된 API
| 권한 | 설명 |
|------|------|
| `CONTAINERS=1` | 컨테이너 목록/상태 조회 |
| `POST=1` | 컨테이너 시작/중지/재시작 |
| `LOGS=1` | 로그 조회 |

### 차단된 API (기본값)
- `BUILD`: 이미지 빌드
- `EXEC`: 컨테이너 내 명령 실행
- `IMAGES`: 이미지 관리
- `NETWORKS`: 네트워크 관리
- `VOLUMES`: 볼륨 관리
- `SWARM`: Swarm 관리

**보안 효과**: 봇이 해킹당해도 호스트 전체가 위험해지지 않음

---

## ⚔️ deunhealth vs Admin UI 충돌 해결

### 문제 상황

1. Admin UI에서 유지보수를 위해 `twentyq-bot`을 **수동 정지** 함
2. deunhealth가 "unhealthy 감지" → 즉시 **자동 재시작**
3. 결과: 봇을 끌 수 없는 좀비 상태

### 해결 방법: 워치독 먼저 정지

Admin UI에서 컨테이너를 유지보수할 때는 **반드시 다음 순서**를 따릅니다:

#### 점검 시작
```
1. Admin UI → Docker 관리 → deunhealth 정지 (Stop)
2. 원하는 컨테이너 정지/재시작/점검
3. 작업 완료
```

#### 점검 종료
```
1. 모든 컨테이너가 정상 상태인지 확인
2. Admin UI → Docker 관리 → deunhealth 시작 (Start)
```

### 주의사항

| 항목 | 설명 |
|------|------|
| ⚠️ deunhealth 정지 중 | 자동 복구가 작동하지 않음. 수동 모니터링 필요 |
| ⏰ 권장 시간 | 점검은 5분 이내로 완료 |
| 🔄 작업 후 | 반드시 deunhealth를 다시 시작할 것 |

---

## 관리 대상 컨테이너

Admin UI에서 제어 가능한 컨테이너 목록:

| 컨테이너 | 설명 | 주의사항 |
|----------|------|----------|
| `hololive-kakao-bot-go` | 메인 봇 | - |
| `mcp-llm-server` | LLM 서버 | 다른 봇의 의존성 |
| `twentyq-bot` | 스무고개 봇 | - |
| `turtle-soup-bot` | 거북이 스프 봇 | - |
| `valkey-cache` | 캐시 | 정지 시 세션 유실 |
| `valkey-mq` | 메시지 큐 | 정지 시 메시지 유실 |
| `llm-postgres` | 데이터베이스 | ⚠️ 정지 시 전체 장애 |
| `deunhealth` | 워치독 | 점검 시 먼저 정지 |
| `jaeger` | 트레이싱 | - |

---

## CLI 대안 (비상시)

Admin UI가 접근 불가한 경우 호스트에서 직접 실행:

```bash
# 워치독 정지
docker stop deunhealth

# 점검 작업...

# 워치독 재시작
docker start deunhealth
```

---

## 문제 해결

### Q: Admin UI에서 컨테이너 조작이 안 됩니다

1. `docker-proxy` 컨테이너가 실행 중인지 확인
   ```bash
   docker ps | grep docker-proxy
   ```

2. 프록시가 중지된 경우 호스트에서 직접 시작
   ```bash
   docker start docker-proxy
   ```

### Q: deunhealth가 계속 컨테이너를 살립니다

이 문서의 "점검 시작" 절차를 따르세요. deunhealth를 먼저 정지해야 합니다.
