package io.github.kapu.turtlesoup.models

import io.github.kapu.turtlesoup.utils.MessageKeys

/** 게임 명령어 sealed class */
sealed class Command {
    /** 게임 시작 (난이도 지정 가능) */
    data class Start(val difficulty: Int? = null, val hasInvalidInput: Boolean = false) : Command()

    /** 질문 (예/아니오 질문) */
    data class Ask(val question: String) : Command()

    /** 정답 제출 */
    data class Answer(val answer: String) : Command()

    /** 힌트 요청 */
    data object Hint : Command()

    /** 제시문 재표시 */
    data object Problem : Command()

    /** 항복 (플레이어 수에 따라 자동 조건 변경) */
    data object Surrender : Command()

    /** 항복 투표 동의 */
    data object Agree : Command()

    /** 요약(정리) 요청 */
    data object Summary : Command()

    /** 도움말 */
    data object Help : Command()

    /** 알 수 없는 명령어 */
    data object Unknown : Command()
}

/** 대기 메시지 키 반환 (대기 메시지가 필요 없으면 null) */
val Command.waitingMessageKey: String?
    get() =
        when (this) {
            is Command.Start -> MessageKeys.START_WAITING
            is Command.Ask -> MessageKeys.PROCESSING_THINKING
            is Command.Hint -> MessageKeys.PROCESSING_GENERATING_HINT
            is Command.Answer -> MessageKeys.PROCESSING_VALIDATING
            else -> null
        }

/** 락이 필요한 명령어 여부 */
val Command.requiresLock: Boolean
    get() =
        when (this) {
            is Command.Help, is Command.Unknown -> false
            else -> true
        }
