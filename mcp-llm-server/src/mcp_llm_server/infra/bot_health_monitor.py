"""Bot health monitor embedded in the LLM server."""

from __future__ import annotations

import asyncio
import contextlib
import logging
import os
import re
import shlex
import subprocess
import urllib.error
import urllib.parse
import urllib.request
from dataclasses import dataclass
from pathlib import Path


log = logging.getLogger(__name__)
HTTP_OK_MIN = 200
HTTP_OK_MAX = 300


@dataclass(frozen=True)
class BotHealthTarget:
    """Health check target with restart policy."""

    name: str
    url: str
    restart_containers: list[str]

    def endpoint_label(self) -> str:
        return f"http:{self.url}"


@dataclass(frozen=True)
class BotHealthConfig:
    """Configuration for bot health monitoring."""

    targets: list[BotHealthTarget]
    restart_cmd: list[str]
    docker_socket: str
    interval_seconds: int
    max_failures: int
    timeout_seconds: float
    startup_grace_seconds: int
    enabled: bool

    @classmethod
    def from_env(cls) -> BotHealthConfig:
        # 기본값: 서버 자체 헬스 엔드포인트(h2c)
        http_host = os.getenv("HTTP_HOST", "127.0.0.1").strip() or "127.0.0.1"
        http_port = os.getenv("HTTP_PORT", "40527").strip() or "40527"
        default_url = f"http://{http_host}:{http_port}/health/ready"
        enabled_flag = os.getenv("BOT_HEALTH_ENABLED", "true").lower() == "true"

        urls_raw = os.getenv("BOT_HEALTH_URLS", "").strip()
        if urls_raw:
            urls = [item for item in re.split(r"[,\s]+", urls_raw) if item]
        else:
            url = os.getenv("BOT_HEALTH_URL", default_url).strip()
            urls = [url] if url else []

        # 도커 컨테이너 내부에는 docker CLI가 없으므로 기본값 비활성
        restart_cmd_raw = os.getenv("BOT_RESTART_CMD", "").strip()
        restart_cmd = shlex.split(restart_cmd_raw) if restart_cmd_raw else []
        restart_targets_raw = os.getenv("BOT_RESTART_CONTAINERS", "").strip()
        restart_containers_raw = [
            item for item in re.split(r"[,\s]+", restart_targets_raw) if item
        ]
        docker_socket = os.getenv(
            "BOT_DOCKER_SOCKET",
            "/var/run/docker.sock",
        ).strip()

        interval_seconds = max(1, int(os.getenv("BOT_HEALTH_INTERVAL_SECONDS", "60")))
        max_failures = max(1, int(os.getenv("BOT_HEALTH_MAX_FAILURES", "5")))
        timeout_seconds = float(os.getenv("BOT_HEALTH_TIMEOUT_SECONDS", "3"))
        startup_grace_seconds = max(
            0, int(os.getenv("BOT_HEALTH_STARTUP_GRACE_SECONDS", "15"))
        )

        targets = [_build_target(url, restart_containers_raw) for url in urls]
        enabled = enabled_flag and bool(targets)

        return cls(
            targets=targets,
            restart_cmd=restart_cmd,
            docker_socket=docker_socket,
            interval_seconds=interval_seconds,
            max_failures=max_failures,
            timeout_seconds=timeout_seconds,
            startup_grace_seconds=startup_grace_seconds,
            enabled=enabled,
        )


def _build_target(url: str, restart_containers_raw: list[str]) -> BotHealthTarget:
    parsed = urllib.parse.urlparse(url)
    host = (parsed.hostname or "").strip()
    name = host if host else url
    if parsed.path and parsed.path != "/":
        name = f"{name}{parsed.path}"

    derived_container = host
    if restart_containers_raw:
        restart_containers = (
            [derived_container]
            if derived_container and derived_container in restart_containers_raw
            else restart_containers_raw
        )
    else:
        restart_containers = [derived_container] if derived_container else []

    return BotHealthTarget(
        name=name,
        url=url,
        restart_containers=restart_containers,
    )


class BotHealthMonitor:
    """Periodically checks bot health and triggers restart on repeated failures."""

    def __init__(self, config: BotHealthConfig) -> None:
        self._config = config
        self._failures: dict[str, int] = {target.name: 0 for target in config.targets}
        self._task: asyncio.Task[None] | None = None

    @property
    def enabled(self) -> bool:
        return self._config.enabled

    async def start(self) -> None:
        if not self._config.enabled:
            log.info(
                "Bot health monitor disabled (BOT_HEALTH_ENABLED=false or no target)"
            )
            return

        if self._task is None:
            self._task = asyncio.create_task(self._run(), name="bot-health-monitor")
            log.info(
                "Bot health monitor started targets=%s interval=%ss failures=%s",
                [target.endpoint_label() for target in self._config.targets],
                self._config.interval_seconds,
                self._config.max_failures,
            )

    async def stop(self) -> None:
        if self._task is None:
            return
        self._task.cancel()
        with contextlib.suppress(asyncio.CancelledError):
            await self._task
        self._task = None
        log.info("Bot health monitor stopped")

    async def _run(self) -> None:
        if self._config.startup_grace_seconds > 0:
            log.info(
                "BOT_HEALTH_GRACE_WAIT seconds=%s target=%s",
                self._config.startup_grace_seconds,
                [target.endpoint_label() for target in self._config.targets],
            )
            await asyncio.sleep(self._config.startup_grace_seconds)

        while True:
            try:
                for target in self._config.targets:
                    healthy = await self._ping(target)
                    if healthy:
                        self._failures[target.name] = 0
                        continue

                    self._failures[target.name] = self._failures.get(target.name, 0) + 1
                    log.warning(
                        "BOT_HEALTH_FAIL consecutive=%s threshold=%s target=%s",
                        self._failures[target.name],
                        self._config.max_failures,
                        target.endpoint_label(),
                    )
                    if self._failures[target.name] >= self._config.max_failures:
                        await self._restart(target)
                        self._failures[target.name] = 0
            except asyncio.CancelledError:
                raise
            except Exception as exc:  # pragma: no cover - safety net
                log.exception(
                    "BOT_HEALTH_MONITOR_ERROR targets=%s error=%s",
                    [target.endpoint_label() for target in self._config.targets],
                    exc,
                )
            await asyncio.sleep(self._config.interval_seconds)

    async def _ping(self, target: BotHealthTarget) -> bool:
        if target.url:
            return await asyncio.to_thread(self._ping_http_sync, target.url)
        log.warning("BOT_HEALTH_SKIP reason=no_target")
        return False

    def _ping_http_sync(self, url: str) -> bool:
        req = urllib.request.Request(url, method="GET")
        try:
            with urllib.request.urlopen(
                req, timeout=self._config.timeout_seconds
            ) as resp:
                status_code = int(resp.getcode())
                return HTTP_OK_MIN <= status_code < HTTP_OK_MAX
        except (urllib.error.URLError, TimeoutError, OSError) as exc:
            log.warning("BOT_HEALTH_HTTP_FAIL url=%s err=%s", url, exc)
            return False

    async def _restart(self, target: BotHealthTarget) -> None:
        log.warning(
            "BOT_RESTART_TRIGGER threshold=%s target=%s",
            self._config.max_failures,
            target.endpoint_label(),
        )
        if not self._config.restart_cmd:
            restarted = self._restart_containers_via_docker(target.restart_containers)
            if not restarted:
                log.warning(
                    "BOT_RESTART_SKIP reason=command_missing target=%s",
                    target.endpoint_label(),
                )
                return
        else:
            # 재시작 명령이 지정된 경우 우선 실행, 실패 시 docker 경로도 시도
            first = self._config.restart_cmd[0]
            if Path(first).is_absolute() and not Path(first).exists():
                log.warning(
                    "BOT_RESTART_SKIP reason=command_not_found cmd=%s target=%s",
                    " ".join(self._config.restart_cmd),
                    target.endpoint_label(),
                )
                return
            exit_code = await asyncio.to_thread(
                self._run_subprocess,
                self._config.restart_cmd,
            )
            if exit_code != 0:
                log.warning(
                    "BOT_RESTART_CMD_FAIL cmd=%s exit_code=%s target=%s",
                    " ".join(self._config.restart_cmd),
                    exit_code,
                    target.endpoint_label(),
                )
                # fallback to docker restart
                self._restart_containers_via_docker(target.restart_containers)
            else:
                log.info(
                    "BOT_RESTART_CMD_OK cmd=%s target=%s",
                    " ".join(self._config.restart_cmd),
                    target.endpoint_label(),
                )
            return

    @staticmethod
    def _run_subprocess(cmd: list[str]) -> int:
        completed = subprocess.run(cmd, check=False)
        return completed.returncode

    def _restart_containers_via_docker(self, containers: list[str]) -> bool:
        if not containers:
            return False

        socket_path = self._config.docker_socket
        if not Path(socket_path).exists():
            log.warning(
                "BOT_RESTART_SKIP reason=docker_socket_missing socket=%s",
                socket_path,
            )
            return False

        restarted = False
        for container in containers:
            cmd = [
                "curl",
                "-fsS",
                "--unix-socket",
                socket_path,
                "--max-time",
                "30",
                "-X",
                "POST",
                f"http://localhost/containers/{container}/restart",
            ]
            result = subprocess.run(
                cmd,
                check=False,
                capture_output=True,
            )
            if result.returncode == 0:
                restarted = True
                log.info(
                    "BOT_RESTART_DOCKER_OK container=%s",
                    container,
                )
            else:
                stderr = result.stderr.decode("utf-8", "ignore").strip()
                log.warning(
                    "BOT_RESTART_DOCKER_FAIL container=%s code=%s stderr=%s",
                    container,
                    result.returncode,
                    stderr[:200],
                )

        return restarted
