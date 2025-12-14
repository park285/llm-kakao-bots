package party.qwer.twentyq.model

enum class RiddleCategory(
    val koreanName: String,
    val description: String,
) {
    ANY("자유", "모든 주제"),
    ORGANISM("생물", "생물 관련"),
    FOOD("음식", "음식 관련"),
    OBJECT("사물", "사물 관련"),
    PLACE("장소", "장소 관련"),
    CONCEPT("개념", "추상적 개념"),
    MOVIE("영화", "영화 관련"),
    ;

    companion object {
        fun fromString(value: String?): RiddleCategory {
            if (value.isNullOrBlank()) return ANY
            return entries.find {
                it.name.equals(value, ignoreCase = true) ||
                    it.koreanName == value
            } ?: ANY
        }
    }
}
