# syntax=docker/dockerfile:1.7

ARG GO_VERSION=1.25.5
ARG ALPINE_VERSION=3.23

FROM golang:${GO_VERSION}-alpine${ALPINE_VERSION} AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod --mount=type=cache,target=/root/.cache/go-build go mod download

COPY . .

RUN --mount=type=cache,target=/go/pkg/mod --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 GOEXPERIMENT=greenteagc \
    go build -tags go_json -trimpath -buildvcs=false -ldflags="-s -w -buildid=" -o /dist/bin/bot ./cmd/bot && \
    mkdir -p /dist/logs /dist/admin-ui /dist/internal/domain /dist/internal/adapter && \
    cp -r admin-ui/dist /dist/admin-ui/ && \
    cp -r internal/domain/data /dist/internal/domain/ && \
    cp -r internal/adapter/templates /dist/internal/adapter/ && \
    cp config.example.json config.json official_japanese_names.json /dist/

FROM alpine:${ALPINE_VERSION}

RUN apk add --no-cache ca-certificates tzdata && \
    cp /usr/share/zoneinfo/Asia/Seoul /etc/localtime && \
    echo "Asia/Seoul" > /etc/timezone

RUN addgroup -g 1000 appuser && \
    adduser -D -u 1000 -G appuser appuser

WORKDIR /app

COPY --from=builder --link --chown=appuser:appuser /dist ./

ENV GOGC=200 \
    GOMEMLIMIT=450MiB \
    GIN_MODE=release

EXPOSE 30001

HEALTHCHECK --interval=30s --timeout=5s --start-period=5s --retries=3 \
    CMD wget -q --spider http://127.0.0.1:30001/health || exit 1

USER appuser

CMD ["./bin/bot"]