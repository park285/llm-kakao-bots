package party.qwer.twentyq.model

/**
 * 체인 질문 실행 조건
 *
 * 첫 질문의 답변에 따라 나머지 질문의 실행 여부를 결정
 */
enum class ChainCondition {
    /**
     * 무조건 실행 (기본값)
     *
     * 모든 질문을 순차적으로 실행
     */
    ALWAYS,

    /**
     * 긍정 답변 시 실행
     *
     * 첫 질문의 답변이 "예" 또는 "아마도 예"일 때만 나머지 질문 실행
     */
    IF_TRUE,
}
