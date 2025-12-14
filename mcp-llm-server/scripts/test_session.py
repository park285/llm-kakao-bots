#!/usr/bin/env python3
"""Test script for MCP session functionality."""

import asyncio
import pathlib
import sys


sys.path.insert(0, str(pathlib.Path(__file__).parent.parent / "src"))


async def test_session_lifecycle():
    """Test session create, chat, info, end."""
    print("\n=== Test: Session Lifecycle ===")

    from mcp_llm_server.infra.gemini_client import get_gemini_client
    from mcp_llm_server.infra.session_manager import get_session_manager

    manager = get_session_manager()
    client = get_gemini_client()
    session_id = "test-session-001"

    # 1. Create session
    print(f"\n1. Creating session: {session_id}")
    session = await manager.get_or_create_session(
        session_id=session_id,
        model="gemini-2.5-flash-preview-09-2025",
        system_prompt="You are a helpful assistant for a Korean riddle game.",
    )
    print(f"   Created: {session.session_id}, model={session.model}")

    # 2. First message
    print("\n2. First message: '안녕하세요'")
    session.add_message("user", "안녕하세요")
    history = session.get_history_as_dicts()[:-1]
    response1 = await client.chat(
        "안녕하세요", session.system_prompt, history, session.model
    )
    session.add_message("assistant", response1)
    print(f"   Response: {response1[:100]}...")

    # 3. Second message (context should be maintained)
    print("\n3. Second message: '방금 제가 뭐라고 했죠?'")
    session.add_message("user", "방금 제가 뭐라고 했죠?")
    history = session.get_history_as_dicts()[:-1]
    response2 = await client.chat(
        "방금 제가 뭐라고 했죠?", session.system_prompt, history, session.model
    )
    session.add_message("assistant", response2)
    print(f"   Response: {response2[:100]}...")

    # 4. Session info
    print("\n4. Session info:")
    info = manager.get_session_info(session_id)
    print(f"   Message count: {info.get('message_count', 0)}")
    print(f"   Model: {info.get('model')}")

    # 5. End session
    print("\n5. Ending session")
    removed = await manager.end_session(session_id)
    print(f"   Removed: {removed}")

    # 6. Verify session is gone
    print("\n6. Verifying session is removed")
    info_after = manager.get_session_info(session_id)
    print(f"   Session exists: {info_after is not None}")

    return True


async def test_twentyq_with_session():
    """Test twentyq tools with session context."""
    print("\n=== Test: TwentyQ Answer with Session ===")

    from mcp_llm_server.domains.twentyq.prompts import get_twentyq_prompts
    from mcp_llm_server.infra.gemini_client import get_gemini_client
    from mcp_llm_server.infra.session_manager import get_session_manager
    from mcp_llm_server.utils.toon import encode_secret

    manager = get_session_manager()
    client = get_gemini_client()
    prompts = get_twentyq_prompts()
    session_id = "twentyq-game-001"

    # Create session with answer system prompt
    system = prompts.answer_system()
    session = await manager.get_or_create_session(
        session_id=session_id,
        model="gemini-2.5-flash-preview-09-2025",
        system_prompt=system,
    )

    secret_toon = encode_secret("스마트폰", "사물", {"type": "전자기기"})
    questions = [
        "전자기기인가요?",
        "손에 들 수 있나요?",
        "통화할 수 있나요?",
    ]

    print("Secret: 스마트폰 (TOON encoded)")
    print(f"Session: {session_id}\n")

    for i, q in enumerate(questions, 1):
        user_content = prompts.answer_user(secret_toon, q)
        session.add_message("user", user_content)
        history = session.get_history_as_dicts()[:-1]
        response = await client.chat(user_content, system, history, session.model)
        session.add_message("assistant", response)
        print(f"Q{i}: {q} → {response.strip()}")

    # Cleanup
    await manager.end_session(session_id)
    print("\nSession ended.")
    return True


async def main():
    """Run all session tests."""
    print("=" * 60)
    print("MCP Session Functionality Test")
    print("=" * 60)

    try:
        await test_session_lifecycle()
        await test_twentyq_with_session()
        print("\n" + "=" * 60)
        print("All session tests completed!")
        print("=" * 60)
    except Exception as e:
        print(f"\nError: {e}")
        import traceback

        traceback.print_exc()
        return 1

    return 0


if __name__ == "__main__":
    exit(asyncio.run(main()))
