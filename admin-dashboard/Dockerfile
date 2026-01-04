# syntax=docker/dockerfile:1.7

ARG GO_VERSION=1.25.5
ARG ALPINE_VERSION=3.23

# Stage 1: Build Frontend
FROM node:22-alpine AS frontend-builder
WORKDIR /app/frontend

# 상위 디렉토리에서 frontend 복사
COPY frontend/package.json frontend/package-lock.json ./
RUN npm ci

COPY frontend/ .
RUN npm run build

# Stage 2: Build Backend (with embedded frontend)
FROM golang:${GO_VERSION}-alpine AS builder

# 빌드 시 버전 주입 (docker-compose args 또는 --build-arg로 전달)
ARG VERSION=dev

WORKDIR /build

COPY backend/go.mod backend/go.sum* ./
RUN go mod download

COPY backend/ .

# 프론트엔드 빌드 결과물을 backend 내부로 복사 (go:embed용)
# internal/static/admin-ui/dist에 복사하여 embed.go에서 참조
COPY --from=frontend-builder /app/frontend/dist ./internal/static/admin-ui/dist

RUN CGO_ENABLED=0 GOOS=linux go build -tags=go_json -ldflags="-s -w -X main.Version=${VERSION}" -o admin ./cmd/admin

# Stage 3: Production Runtime (Single Binary)
FROM alpine:${ALPINE_VERSION}

RUN apk add --no-cache ca-certificates tini tzdata && \
    cp /usr/share/zoneinfo/Asia/Seoul /etc/localtime && \
    echo "Asia/Seoul" > /etc/timezone && \
    cp /usr/share/zoneinfo/Asia/Seoul /tmp/Seoul && \
    apk del --no-cache tzdata && \
    mkdir -p /usr/share/zoneinfo/Asia && \
    mv /tmp/Seoul /usr/share/zoneinfo/Asia/Seoul

RUN addgroup -g 1000 appgroup && \
    adduser -u 1000 -G appgroup -s /bin/sh -D appuser

WORKDIR /app

# 단일 바이너리만 복사 (프론트엔드는 바이너리에 임베딩됨)
COPY --from=builder /build/admin .

RUN chown -R appuser:appgroup /app

USER appuser

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:${PORT:-30090}/health || exit 1

EXPOSE 30090

ENTRYPOINT ["/sbin/tini", "--"]
CMD ["./admin"]
