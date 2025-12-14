package io.github.kapu.turtlesoup.utils

import io.github.kapu.turtlesoup.config.AccessConfig
import io.kotest.core.spec.style.StringSpec
import io.kotest.matchers.shouldBe

class AccessControlTest : StringSpec({
    "passthrough 모드면 차단 목록도 무시한다" {
        val control = createControl(passthrough = true, blockedUsers = listOf("user-1"))

        control.getDenialReason("user-1", "chat-1") shouldBe null
    }

    "차단된 사용자는 거부한다" {
        val control = createControl(blockedUsers = listOf("user-1"))

        control.getDenialReason("user-1", "chat-1") shouldBe MessageKeys.ERROR_USER_BLOCKED
    }

    "차단된 채팅방은 거부한다" {
        val control = createControl(blockedChats = listOf("chat-1"))

        control.getDenialReason("user-1", "chat-1") shouldBe MessageKeys.ERROR_CHAT_BLOCKED
    }

    "허용 리스트가 설정된 경우 목록 밖 채팅은 거부한다" {
        val control = createControl(allowedChats = listOf("chat-allowed"))

        control.getDenialReason("user-1", "chat-denied") shouldBe MessageKeys.ERROR_ACCESS_DENIED
    }

    "허용 리스트에 있으면 통과한다" {
        val control = createControl(allowedChats = listOf("chat-allowed"))

        control.getDenialReason("user-1", "chat-allowed") shouldBe null
    }
}) {
    companion object {
        private fun createControl(
            passthrough: Boolean = false,
            allowedChats: List<String> = emptyList(),
            blockedChats: List<String> = emptyList(),
            blockedUsers: List<String> = emptyList(),
        ): AccessControl =
            AccessControl(
                AccessConfig(
                    enabled = true,
                    allowedChatIds = allowedChats,
                    blockedChatIds = blockedChats,
                    blockedUserIds = blockedUsers,
                    passthrough = passthrough,
                ),
            )
    }
}
