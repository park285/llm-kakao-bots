#!/usr/bin/env python3
"""Lightweight health monitor for MCP LLM server and bot.

Features:
- Polls health endpoints at fixed interval.
- On 5 consecutive failures, triggers restart command.
- Handles both server and bot (if BOT_HEALTH_URL/BOT_RESTART_CMD provided).
"""

from __future__ import annotations

import os
import shlex
import subprocess
import sys
import time
import urllib.parse
import urllib.request
from dataclasses import dataclass
from pathlib import Path
from socket import AF_UNIX, SOCK_STREAM, socket
from urllib.error import URLError

from dotenv import load_dotenv


PROJECT_ROOT = Path(__file__).resolve().parent.parent


@dataclass
class Target:
    """헬스 체크 대상."""

    name: str
    url: str
    path: str
    uds_path: str | None
    restart_cmd: list[str]
    timeout: float

    def endpoint_label(self) -> str:
        if self.uds_path:
            return f"uds:{self.uds_path}{self.path}"
        return f"http:{self.url}"


def _timestamp() -> str:
    return time.strftime("%Y-%m-%d %H:%M:%S")


def _log(message: str) -> None:
    sys.stdout.write(f"[{_timestamp()}] {message}\n")
    sys.stdout.flush()


def _ping_http(url: str, timeout: float) -> bool:
    HTTP_OK_MIN = 200
    HTTP_OK_MAX = 300
    req = urllib.request.Request(url, method="GET")
    try:
        with urllib.request.urlopen(req, timeout=timeout) as resp:
            return HTTP_OK_MIN <= resp.getcode() < HTTP_OK_MAX
    except (URLError, TimeoutError) as exc:
        _log(f"{url} ping 실패: {exc}")
        return False


def _ping_uds(uds_path: str, path: str, timeout: float) -> bool:
    HTTP_OK_MIN = 200
    HTTP_OK_MAX = 300
    request = f"GET {path} HTTP/1.1\r\nHost: localhost\r\nConnection: close\r\n\r\n"
    try:
        with socket(AF_UNIX, SOCK_STREAM) as sock:
            sock.settimeout(timeout)
            sock.connect(uds_path)
            sock.sendall(request.encode("ascii"))
            response = b""
            while True:
                chunk = sock.recv(4096)
                if not chunk:
                    break
                response += chunk
    except OSError as exc:
        _log(f"{uds_path} UDS ping 실패: {exc}")
        return False

    status_line = response.split(b"\r\n", 1)[0].decode("ascii", "ignore")
    parts = status_line.split()
    min_status_parts = 2
    if len(parts) < min_status_parts:
        _log(f"{uds_path} UDS ping 실패: malformed response '{status_line}'")
        return False
    try:
        status = int(parts[1])
    except ValueError:
        _log(f"{uds_path} UDS ping 실패: invalid status '{parts[1]}'")
        return False
    return HTTP_OK_MIN <= status < HTTP_OK_MAX


def _ping(target: Target) -> bool:
    if target.uds_path:
        return _ping_uds(target.uds_path, target.path, target.timeout)
    return _ping_http(target.url, target.timeout)


def _restart(cmd: list[str], name: str) -> None:
    if not cmd:
        _log(f"{name} 재시작 명령 미설정, 건너뜀")
        return

    _log(f"{name} 재시작 시도: {' '.join(cmd)}")
    subprocess.run(cmd, cwd=PROJECT_ROOT, check=False)


def monitor(targets: list[Target], interval_seconds: int, max_failures: int) -> None:
    failures: dict[str, int] = {target.name: 0 for target in targets}
    target_map = {target.name: target for target in targets}

    while True:
        for name, target in target_map.items():
            ok = _ping(target)
            if ok:
                failures[name] = 0
                continue

            failures[name] += 1
            _log(
                f"{name} 연속 실패 {failures[name]}/{max_failures} "
                f"({target.endpoint_label()})"
            )
            if failures[name] >= max_failures:
                _restart(target.restart_cmd, name)
                failures[name] = 0

        time.sleep(interval_seconds)


def _get_env_cmd(key: str, default: str) -> list[str]:
    raw = os.getenv(key, default).strip()
    return shlex.split(raw) if raw else []


def main() -> None:
    load_dotenv()

    bot_enabled = os.getenv("BOT_HEALTH_ENABLED", "true").lower() == "true"

    server_url = os.getenv("SERVER_HEALTH_URL", "http://127.0.0.1:40527/health/ready")
    uds_path = os.getenv("SERVER_HEALTH_UDS_PATH") or os.getenv("HTTP_UDS_PATH", "")
    bot_default_url = "http://127.0.0.1:30003/actuator/health"
    bot_url = os.getenv("BOT_HEALTH_URL", bot_default_url)
    bot_uds_path = os.getenv("BOT_HEALTH_UDS_PATH", "")
    bot_health_path = urllib.parse.urlparse(bot_url).path or "/actuator/health"
    server_restart_cmd = _get_env_cmd(
        "SERVER_RESTART_CMD", str(PROJECT_ROOT / "scripts" / "restart.sh")
    )
    bot_restart_cmd = _get_env_cmd(
        "BOT_RESTART_CMD", "/home/kapu/gemini/llm/20q-kakao-bot/bot-restart.sh"
    )
    interval_seconds = int(os.getenv("HEALTH_INTERVAL_SECONDS", "60"))
    max_failures = int(os.getenv("HEALTH_MAX_FAILURES", "5"))
    timeout_seconds = float(os.getenv("HEALTH_TIMEOUT_SECONDS", "3"))

    parsed = urllib.parse.urlparse(server_url)
    health_path = parsed.path or "/health/ready"

    targets: list[Target] = [
        Target(
            name="llm-server",
            url=server_url,
            path=health_path,
            uds_path=uds_path if uds_path else None,
            restart_cmd=server_restart_cmd,
            timeout=timeout_seconds,
        )
    ]

    if bot_enabled:
        targets.append(
            Target(
                name="bot-20q",
                url=bot_url,
                path=bot_health_path,
                uds_path=bot_uds_path if bot_uds_path else None,
                restart_cmd=bot_restart_cmd,
                timeout=timeout_seconds,
            )
        )
        targets.append(
            Target(
                name="bot-turtle",
                url="http://127.0.0.1:40808/health",
                path="/health",
                uds_path=None,
                restart_cmd=[
                    "/home/kapu/gemini/llm/turtle-soup-bot/bot-restart.sh",
                ],
                timeout=timeout_seconds,
            )
        )
    else:
        _log("봇 헬스 모니터 비활성화: BOT_HEALTH_ENABLED=false")

    _log(
        "헬스 모니터 시작: "
        f"targets={[f'{t.name}={t.endpoint_label()}' for t in targets]}, "
        f"interval={interval_seconds}s, "
        f"max_failures={max_failures}, "
        f"timeout={timeout_seconds}s",
    )
    monitor(targets, interval_seconds=interval_seconds, max_failures=max_failures)


if __name__ == "__main__":
    main()
