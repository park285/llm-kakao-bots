"""MCP LLM Server - LangChain Gemini, Injection Guard, Korean NLP"""

__version__ = "0.1.0"

# Client exports for chat bots
from mcp_llm_server.client import LLMClient


__all__ = ["LLMClient", "__version__"]
