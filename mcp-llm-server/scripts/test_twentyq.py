#!/usr/bin/env python3
"""Test script for twentyq MCP tools."""

import asyncio
import sys
from pathlib import Path


# Add src to path
sys.path.insert(0, str(Path(__file__).parent.parent / "src"))


async def test_hints():
    """Test hint generation with TOON format."""
    print("\n=== Test: twentyq_generate_hints ===")
    from typing import cast

    from langchain_core.prompts import ChatPromptTemplate

    from mcp_llm_server.config.settings import get_settings
    from mcp_llm_server.domains.twentyq.models import HintsOutput
    from mcp_llm_server.domains.twentyq.prompts import get_twentyq_prompts
    from mcp_llm_server.infra.gemini_client import get_gemini_client
    from mcp_llm_server.utils.toon import encode_secret

    client = get_gemini_client()
    settings = get_settings()
    prompts = get_twentyq_prompts()

    # 테스트 데이터
    target = "스마트폰"
    category = "사물"
    details = {"type": "전자기기", "usage": "통신, 인터넷"}

    # TOON 인코딩
    secret_toon = encode_secret(target, category, details)
    print(f"TOON encoded secret:\n{secret_toon}\n")

    system = prompts.hints_system(category)
    prompt = ChatPromptTemplate.from_messages(
        [
            ("system", system),
            ("human", "{secret_info}"),
        ]
    )

    model = settings.gemini.get_model("hints")
    llm = client._get_llm(model)
    structured_llm = llm.with_structured_output(HintsOutput)
    chain = prompt | structured_llm

    user_content = prompts.hints_user(secret_toon)
    print(f"Model: {model}")
    print("Calling LLM...")

    result = cast("HintsOutput", await chain.ainvoke({"secret_info": user_content}))
    print(f"Hints: {result.hints}")
    return result.hints


async def test_answer():
    """Test answer question with TOON format."""
    print("\n=== Test: twentyq_answer_question ===")
    from langchain_core.prompts import ChatPromptTemplate

    from mcp_llm_server.config.settings import get_settings
    from mcp_llm_server.domains.twentyq.models import AnswerScale
    from mcp_llm_server.domains.twentyq.prompts import get_twentyq_prompts
    from mcp_llm_server.infra.gemini_client import get_gemini_client
    from mcp_llm_server.utils.toon import encode_secret

    client = get_gemini_client()
    settings = get_settings()
    prompts = get_twentyq_prompts()

    target = "스마트폰"
    category = "사물"
    question = "전자기기인가요?"

    secret_toon = encode_secret(target, category)
    print(f"TOON: {secret_toon}")

    system = prompts.answer_system()
    prompt = ChatPromptTemplate.from_messages(
        [
            ("system", system),
            ("human", "{user_prompt}"),
        ]
    )

    model = settings.gemini.get_model("answer")
    llm = client._get_llm(model)
    chain = prompt | llm

    user_content = prompts.answer_user(secret_toon, question)
    print(f"Question: {question}")
    print(f"Model: {model}")
    print("Calling LLM...")

    result = await chain.ainvoke({"user_prompt": user_content})
    raw_text = client._extract_text(result)
    scale = AnswerScale.from_text(raw_text)

    print(f"Raw response: {raw_text}")
    print(f"Parsed scale: {scale.value if scale else 'PARSE_FAILED'}")
    return scale


async def test_verify():
    """Test verify guess."""
    print("\n=== Test: twentyq_verify_guess ===")
    from typing import cast

    from langchain_core.prompts import ChatPromptTemplate

    from mcp_llm_server.config.settings import get_settings
    from mcp_llm_server.domains.twentyq.models import VerifyOutput
    from mcp_llm_server.domains.twentyq.prompts import get_twentyq_prompts
    from mcp_llm_server.infra.gemini_client import get_gemini_client

    client = get_gemini_client()
    settings = get_settings()
    prompts = get_twentyq_prompts()

    target = "스마트폰"
    guess = "핸드폰"

    system = prompts.verify_system()
    prompt = ChatPromptTemplate.from_messages(
        [
            ("system", system),
            ("human", "{user_prompt}"),
        ]
    )

    model = settings.gemini.get_model("verify")
    llm = client._get_llm(model)
    structured_llm = llm.with_structured_output(VerifyOutput)
    chain = prompt | structured_llm

    user_content = prompts.verify_user(target, guess)
    print(f"Target: {target}, Guess: {guess}")
    print(f"Model: {model}")
    print("Calling LLM...")

    result = cast("VerifyOutput", await chain.ainvoke({"user_prompt": user_content}))
    print(f"Result: {result.result.value}")
    return result.result


async def test_normalize():
    """Test normalize question."""
    print("\n=== Test: twentyq_normalize_question ===")
    from typing import cast

    from langchain_core.prompts import ChatPromptTemplate

    from mcp_llm_server.config.settings import get_settings
    from mcp_llm_server.domains.twentyq.models import NormalizeOutput
    from mcp_llm_server.domains.twentyq.prompts import get_twentyq_prompts
    from mcp_llm_server.infra.gemini_client import get_gemini_client

    client = get_gemini_client()
    settings = get_settings()
    prompts = get_twentyq_prompts()

    # 오타가 있는 질문
    question = "전자기기인가요ㅛㅛ"

    system = prompts.normalize_system()
    prompt = ChatPromptTemplate.from_messages(
        [
            ("system", system),
            ("human", "{question}"),
        ]
    )

    model = settings.gemini.get_model()
    llm = client._get_llm(model)
    structured_llm = llm.with_structured_output(NormalizeOutput)
    chain = prompt | structured_llm

    user_content = prompts.normalize_user(question)
    print(f"Original: {question}")
    print(f"Model: {model}")
    print("Calling LLM...")

    result = cast("NormalizeOutput", await chain.ainvoke({"question": user_content}))
    print(f"Normalized: {result.normalized}")
    return result.normalized


async def test_synonym():
    """Test synonym check."""
    print("\n=== Test: twentyq_check_synonym ===")
    from typing import cast

    from langchain_core.prompts import ChatPromptTemplate

    from mcp_llm_server.config.settings import get_settings
    from mcp_llm_server.domains.twentyq.models import SynonymOutput
    from mcp_llm_server.domains.twentyq.prompts import get_twentyq_prompts
    from mcp_llm_server.infra.gemini_client import get_gemini_client

    client = get_gemini_client()
    settings = get_settings()
    prompts = get_twentyq_prompts()

    # 동의어 테스트 케이스들
    test_cases = [
        ("스마트폰", "핸드폰", "EQUIVALENT"),  # 동의어
        ("컴퓨터", "노트북", "NOT_EQUIVALENT"),  # 상위-하위 관계
    ]

    system = prompts.synonym_system()
    model = settings.gemini.get_model()
    llm = client._get_llm(model)
    structured_llm = llm.with_structured_output(SynonymOutput)

    for target, guess, expected in test_cases:
        prompt = ChatPromptTemplate.from_messages(
            [
                ("system", system),
                ("human", "{user_prompt}"),
            ]
        )
        chain = prompt | structured_llm

        user_content = prompts.synonym_user(target, guess)
        print(f"\nTarget: {target}, Guess: {guess} (expected: {expected})")
        print("Calling LLM...")

        result = cast(
            "SynonymOutput", await chain.ainvoke({"user_prompt": user_content})
        )
        status = "PASS" if result.result.value == expected else "FAIL"
        print(f"Result: {result.result.value} [{status}]")

    return True


async def main():
    """Run all tests."""
    print("=" * 60)
    print("Twenty Questions MCP Tools Test (TOON Format)")
    print("=" * 60)

    try:
        await test_hints()
        await test_answer()
        await test_verify()
        await test_normalize()
        await test_synonym()
        print("\n" + "=" * 60)
        print("All tests completed!")
        print("=" * 60)
    except Exception as e:
        print(f"\nError: {e}")
        import traceback

        traceback.print_exc()
        return 1

    return 0


if __name__ == "__main__":
    exit(asyncio.run(main()))
