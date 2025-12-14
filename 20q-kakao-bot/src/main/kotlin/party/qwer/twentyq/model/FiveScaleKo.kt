package party.qwer.twentyq.model

import party.qwer.twentyq.util.common.extensions.normalizeEnumResponse

enum class FiveScaleKo {
    ALWAYS_YES,
    MOSTLY_YES,
    MOSTLY_NO,
    ALWAYS_NO,
    INVALID,
    ;

    companion object {
        // LLM 스키마용 전체 값 리스트 (INVALID 포함)
        val SCHEMA_VALUES: List<String> =
            listOf(
                "예",
                "아마도 예",
                "아마도 아니오",
                "아니오",
                "이해할 수 없는 질문입니다",
            )

        // 재시도 시 제시할 후보 목록 (INVALID 제외)
        const val RETRY_CANDIDATES: String = "예, 아마도 예, 아마도 아니오, 아니오"

        private val TOKENS: Map<String, FiveScaleKo> =
            mapOf(
                "예" to ALWAYS_YES,
                "아마도 예" to MOSTLY_YES,
                "아마도 아니오" to MOSTLY_NO,
                "아니오" to ALWAYS_NO,
                "이해할 수 없는 질문입니다" to INVALID,
            )

        fun fromText(raw: String?): FiveScaleKo? {
            if (raw.isNullOrBlank()) return null
            val cleaned =
                raw
                    .normalizeEnumResponse() // 따옴표 제거 (공통)
                    .replace("\u3000", " ") // 전각 공백
                    .replace("\u3002", ".") // 전각 마침표
                    .replace("\uff0c", ",") // 전각 쉼표
                    .removeSuffix(".")
                    .removeSuffix("!")
                    .removeSuffix("?")
                    .removeSuffix("\u3002") // 전각 마침표
                    .removeSuffix("\uff01") // 전각 느낌표
                    .removeSuffix("\uff1f") // 전각 물음표
                    .trim()

            return TOKENS[cleaned]
        }

        fun token(value: FiveScaleKo): String = TOKENS.entries.first { it.value == value }.key
    }
}
