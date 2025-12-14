"""Logging configuration using loguru.

MCP 서버는 stdio 통신을 사용하므로 stdout/stderr 로깅 주의 필요.
파일 로깅만 사용하고, enqueue=True로 비동기 처리.
"""

import logging
import os
import sys
from dataclasses import dataclass
from pathlib import Path
from typing import TYPE_CHECKING, Any

from loguru import logger

from mcp_llm_server.config.settings import get_settings


if TYPE_CHECKING:
    from loguru import Logger


log = logger.bind(name=__name__)


# 기본 로그 디렉토리
DEFAULT_LOG_DIR = Path(__file__).parent.parent.parent.parent / "logs"

# loguru 포맷 (공백 최소화)
LOG_FORMAT = (
    "<green>{time:YYYY-MM-DD HH:mm:ss.SSS}</green>|"
    "<level>{level:5}</level>|"
    "<cyan>{name}</cyan>:<cyan>{function}</cyan>:<cyan>{line}</cyan>|"
    "<level>{message}</level>"
)

LOG_FORMAT_FILE = "{time:YYYY-MM-DD HH:mm:ss.SSS}|{level:5}|{extra[name]}|{message}"

# 로거명 매핑 (uvicorn.error 명칭 혼동 방지)
LOGGER_NAME_MAP: dict[str, str] = {
    "uvicorn.error": "uvicorn.server",
}

UVICORN_METHOD_INDEX = 1
UVICORN_PATH_INDEX = 2
UVICORN_STATUS_INDEX = 4


@dataclass
class FileLogOptions:
    rotation: str = "10 MB"
    retention: str = "7 days"
    compression: str = "gz"
    json_logs: bool = False
    file_name: str | None = None


class InterceptHandler(logging.Handler):
    """표준 logging을 loguru로 리다이렉트하는 핸들러.

    LangChain 등 외부 라이브러리의 logging 호출도 loguru로 통합.
    """

    def emit(self, record: logging.LogRecord) -> None:
        # loguru level 매핑
        try:
            level: str | int = logger.level(record.levelname).name
        except ValueError:
            level = record.levelno

        message = record.getMessage()
        if record.name == "uvicorn.access":
            try:
                args_raw = record.args or ()
                args: tuple[Any, ...] = args_raw if isinstance(args_raw, tuple) else ()
                method = (
                    args[UVICORN_METHOD_INDEX]
                    if len(args) > UVICORN_METHOD_INDEX
                    else getattr(record, "method", "-")
                )
                path = (
                    args[UVICORN_PATH_INDEX]
                    if len(args) > UVICORN_PATH_INDEX
                    else getattr(record, "path", "-")
                )
                status = (
                    args[UVICORN_STATUS_INDEX]
                    if len(args) > UVICORN_STATUS_INDEX
                    else getattr(record, "status_code", "-")
                )
                transport = "h2c" if get_settings().http.http2_enabled else "http1"
                message = f"{transport} {method} {path} {status}"
            except (IndexError, AttributeError, TypeError):
                message = record.getMessage()

        target_name = LOGGER_NAME_MAP.get(record.name, record.name)
        logger.bind(name=target_name).opt(exception=record.exc_info).log(level, message)


def setup_logging(
    log_dir: Path | str | None = None,
    log_level: str | None = None,
    *,
    file_options: FileLogOptions | None = None,
) -> None:
    """로깅 설정 초기화.

    Args:
        log_dir: 로그 디렉토리 경로 (None이면 기본값 사용)
        log_level: 로그 레벨 (DEBUG, INFO, WARNING, ERROR)
        file_options: 파일 로깅 옵션 (회전/보관/압축/JSON/파일명)
    """
    settings = get_settings()
    options = file_options or FileLogOptions(
        rotation=settings.logging.rotation,
        retention=settings.logging.retention,
        compression=settings.logging.compression,
        json_logs=settings.logging.json_logs,
    )
    # 로그 레벨 정규화 (loguru는 대문자 필요)
    effective_level = (log_level or settings.logging.level).upper()

    # 로그 디렉토리 설정
    log_path = (
        Path(log_dir or settings.logging.log_dir)
        if (log_dir or settings.logging.log_dir)
        else DEFAULT_LOG_DIR
    )
    log_path.mkdir(parents=True, exist_ok=True)

    # 기존 핸들러 제거
    logger.remove()

    # 기본 extra 값 설정 (root 표기 방지)
    logger.configure(extra={"name": "mcp-llm-server"})

    # 로그 파일 이름 결정 (테스트 시 분리)
    target_log_file = options.file_name or "mcp-llm-server.log"
    if os.getenv("PYTEST_CURRENT_TEST"):
        target_log_file = "mcp-llm-server-test.log"

    # stderr 핸들러 (개발용, MCP 서버에서는 비활성화 권장)
    # MCP는 stdio 사용하므로 stderr도 주의 필요
    if settings.logging.console_enabled:
        logger.add(
            sys.stderr,
            format=LOG_FORMAT,
            level=effective_level,
            colorize=True,
            enqueue=True,
        )

    # 단일 파일 핸들러 (SSOT)
    logger.add(
        log_path / target_log_file,
        format=LOG_FORMAT_FILE,
        level=effective_level,
        rotation=options.rotation,
        retention=options.retention,
        compression=options.compression,
        enqueue=True,  # 비동기, thread-safe
        serialize=options.json_logs,
    )

    # 표준 logging 라이브러리 인터셉트
    logging.basicConfig(handlers=[InterceptHandler()], level=0, force=True)

    # 외부 라이브러리 로그 레벨 조정
    for lib_logger in ["httpx", "httpcore", "urllib3", "asyncio"]:
        logging.getLogger(lib_logger).setLevel(logging.WARNING)

    # redisvl 로거 레벨 조정 (Index already exists 로그 억제)
    for lib_logger in ["redisvl", "redisvl.index", "redisvl.index.index"]:
        logging.getLogger(lib_logger).setLevel(logging.WARNING)

    # uvicorn, redisvl, langgraph 로거도 인터셉트하여 포맷 통일
    for lib_logger in [
        "uvicorn",
        "uvicorn.access",
        "uvicorn.error",
        "hypercorn",
        "hypercorn.access",
        "hypercorn.error",
        "redisvl",
        "redisvl.index",
        "redisvl.index.index",
        "langgraph",
        "langgraph.checkpoint",
        "langgraph.checkpoint.redis",
        "langgraph.checkpoint.redis.aio",
    ]:
        lib_log = logging.getLogger(lib_logger)
        lib_log.handlers = [InterceptHandler()]
        lib_log.propagate = False

    log.info(
        "Logging initialized: dir={}, level={}, rotation={}, retention={}",
        log_path,
        effective_level,
        options.rotation,
        options.retention,
    )


def get_logger(name: str) -> "Logger":
    """모듈별 로거 반환.

    Args:
        name: 모듈 이름 (보통 __name__)

    Returns:
        loguru logger with bound name
    """
    return logger.bind(name=name)
