# syntax=docker/dockerfile:1.7

FROM node:20-alpine AS ui-build
WORKDIR /app
COPY admin-ui/package.json admin-ui/pnpm-lock.yaml ./
RUN npm install -g pnpm
RUN --mount=type=cache,target=/root/.pnpm-store pnpm install --frozen-lockfile
COPY admin-ui/ ./
RUN pnpm run build

FROM golang:1.25-alpine3.23 AS build
WORKDIR /src
RUN apk add --no-cache git
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod --mount=type=cache,target=/root/.cache/go-build go mod download
COPY . .
# Copy built UI assets to admin package directory for embedding
COPY --from=ui-build /app/dist ./admin/dist
RUN --mount=type=cache,target=/go/pkg/mod --mount=type=cache,target=/root/.cache/go-build CGO_ENABLED=0 go build -tags go_json -trimpath -buildvcs=false -ldflags="-s -w -buildid=" -o /out/llm-watchdog .

FROM alpine:3.23

COPY --from=build /out/llm-watchdog /usr/local/bin/llm-watchdog
ENTRYPOINT ["llm-watchdog"]
