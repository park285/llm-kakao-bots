# syntax=docker/dockerfile:1.7

FROM python:3.13-slim AS base
WORKDIR /app

# System deps for kiwipiepy/ahocorasick builds
RUN apt-get update && \
    apt-get install -y --no-install-recommends build-essential curl && \
    rm -rf /var/lib/apt/lists/*

COPY pyproject.toml uv.lock README.md ./
COPY src ./src

# Install project (script entrypoint: mcp-llm-server)
RUN pip install --no-cache-dir --upgrade pip && \
    pip install --no-cache-dir .

EXPOSE 40527

ENV HTTP_HOST=0.0.0.0 \
    HTTP_PORT=40527 \
    HTTP2_ENABLED=true

VOLUME ["/run/mcp-llm"]

ENTRYPOINT ["mcp-llm-server"]
