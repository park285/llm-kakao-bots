package party.qwer.twentyq.util.game

/**
 * 카테고리별 금지 단어 매핑
 * LLM 프롬프트에 삽입하여 카테고리 정보 누출 방지
 */
object CategorySynonyms {
    private val CATEGORY_MAPPING =
        mapOf(
            "영화" to setOf("영화", "매체", "미디어", "콘텐츠", "작품", "영상물"),
            "음식" to setOf("음식", "먹거리", "식품", "요리", "음식물", "식재료"),
            "동물" to setOf("동물", "생물", "생명체", "짐승", "야생동물", "가축"),
            "장소" to setOf("장소", "공간", "곳", "위치", "지역", "시설"),
            "사물" to setOf("사물", "물건", "물체", "도구", "기구", "제품"),
            "식물" to setOf("식물", "생물", "생명체", "초목", "나무"),
            "직업" to setOf("직업", "직종", "일", "업무", "전문가"),
            "운동" to setOf("운동", "스포츠", "경기", "체육"),
            "교통수단" to setOf("교통수단", "이동수단", "운송수단", "탈것", "차량"),
            "건물" to setOf("건물", "건축물", "시설물", "구조물"),
            "의류" to setOf("의류", "옷", "의복", "복장", "의상"),
            "가구" to setOf("가구", "집기", "가구류"),
            "전자제품" to setOf("전자제품", "전자기기", "가전", "기기", "디바이스"),
            "악기" to setOf("악기", "연주도구", "악기류"),
            "책" to setOf("책", "서적", "도서", "출판물", "문헌"),
        )

    /**
     * 카테고리에 해당하는 금지 단어 목록 반환
     * 프롬프트에 삽입하여 LLM에게 사용 금지 지시
     */
    fun getForbiddenWords(category: String): Set<String> = CATEGORY_MAPPING[category] ?: emptySet()

    /**
     * 프롬프트용 금지 단어 문자열 생성
     */
    fun toForbiddenWordsString(category: String): String {
        val words = getForbiddenWords(category)
        return if (words.isEmpty()) {
            "(no forbidden words)"
        } else {
            words.joinToString(", ")
        }
    }
}
