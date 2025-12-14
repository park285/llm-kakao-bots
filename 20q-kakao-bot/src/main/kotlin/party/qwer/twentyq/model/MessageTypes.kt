package party.qwer.twentyq.model

/**
 * 외부로부터 수신한 메시지
 *
 * @property chatId 채팅방 ID
 * @property userId 사용자 ID
 * @property content 메시지 내용
 * @property threadId 스레드 ID (선택)
 * @property sender 발신자 이름 (선택, 오픈톡에서는 닉네임, 일반톡에서는 null)
 */
data class InboundMessage(
    val chatId: String,
    val userId: String,
    val content: String,
    val threadId: String?,
    val sender: String? = null,
)

/** 외부로 송신할 메시지 타입 */
sealed interface OutboundMessage {
    val chatId: String
    val text: String
    val threadId: String?

    /** 처리 중 대기 메시지 */
    data class Waiting(
        override val chatId: String,
        override val text: String,
        override val threadId: String?,
    ) : OutboundMessage

    /** 최종 응답 메시지 */
    data class Final(
        override val chatId: String,
        override val text: String,
        override val threadId: String?,
    ) : OutboundMessage

    /** 오류 메시지 */
    data class Error(
        override val chatId: String,
        override val text: String,
        override val threadId: String?,
    ) : OutboundMessage
}
