package party.qwer.twentyq.util.game.constants

// 입력 검증 상수
object ValidationConstants {
    // 질문 길이 제한
    const val MIN_QUESTION_LENGTH = 5
    const val MAX_QUESTION_LENGTH = 100

    // 마스킹 길이
    const val MIN_MASK_REVEAL_LENGTH = 4 // 4자 이하는 전체 마스킹
    const val TOKEN_MASK_MIN_LENGTH = 10 // 10자 이하 토큰은 전체 마스킹
    const val MASK_PREFIX_LENGTH = 2 // 일반 마스킹 앞 2자
    const val MASK_SUFFIX_LENGTH = 2 // 일반 마스킹 뒤 2자
    const val TOKEN_PREFIX_LENGTH = 6 // 토큰 마스킹 앞 6자
    const val TOKEN_SUFFIX_LENGTH = 4 // 토큰 마스킹 뒤 4자
    const val API_KEY_MASK_LENGTH = 10 // API 키 로깅용 앞 10자

    // 카카오톡 메시지 제한
    const val KAKAO_MESSAGE_MAX_LENGTH = 500

    // 캐시 크기
    const val META_QUESTION_CACHE_SIZE = 5_000L // MetaQuestionValidator 캐시

    // 쿼리 제한
    const val MAX_STATS_QUERY_LIMIT = 1000 // 통계 조회 최대 레코드 수
}
