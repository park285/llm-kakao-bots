package party.qwer.twentyq.util.game.constants

object UnicodeConstants {
    // 한글 자모 범위
    const val HANGUL_JAMO_START = 0x1100
    const val HANGUL_JAMO_END = 0x11FF

    const val HANGUL_JAMO_EXTENDED_A_START = 0xA960
    const val HANGUL_JAMO_EXTENDED_A_END = 0xA97F

    const val HANGUL_JAMO_EXTENDED_B_START = 0xD7B0
    const val HANGUL_JAMO_EXTENDED_B_END = 0xD7FF

    // 한글 호환 자모
    const val HANGUL_COMPATIBILITY_JAMO_START = 0x3130
    const val HANGUL_COMPATIBILITY_JAMO_END = 0x318F

    // 한글 완성형 범위
    const val HANGUL_SYLLABLES_START = 0xAC00
    const val HANGUL_SYLLABLES_END = 0xD7AF

    // 이모지 범위 상수
    private const val EMOJI_MISC_SYMBOLS_START = 0x1F300
    private const val EMOJI_MISC_SYMBOLS_END = 0x1FAFF
    private const val EMOJI_MISC_SYMBOLS_2_START = 0x2600
    private const val EMOJI_MISC_SYMBOLS_2_END = 0x27BF
    private const val EMOJI_SUPPLEMENTAL_START = 0x1F900
    private const val EMOJI_SUPPLEMENTAL_END = 0x1F9FF
    private const val EMOJI_EMOTICONS_START = 0x1F600
    private const val EMOJI_EMOTICONS_END = 0x1F64F
    private const val EMOJI_TRANSPORT_START = 0x1F680
    private const val EMOJI_TRANSPORT_END = 0x1F6FF
    private const val EMOJI_VARIATION_SELECTORS_START = 0xFE00
    private const val EMOJI_VARIATION_SELECTORS_END = 0xFE0F
    private const val EMOJI_TECH_SYMBOLS_START = 0x2300
    private const val EMOJI_TECH_SYMBOLS_END = 0x23FF
    private const val EMOJI_STARS_START = 0x2B50
    private const val EMOJI_STARS_END = 0x2B55

    // 이모지 범위 (Guard 공통 사용)
    val EMOJI_RANGES =
        listOf(
            EMOJI_MISC_SYMBOLS_START..EMOJI_MISC_SYMBOLS_END,
            EMOJI_MISC_SYMBOLS_2_START..EMOJI_MISC_SYMBOLS_2_END,
            EMOJI_SUPPLEMENTAL_START..EMOJI_SUPPLEMENTAL_END,
            EMOJI_EMOTICONS_START..EMOJI_EMOTICONS_END,
            EMOJI_TRANSPORT_START..EMOJI_TRANSPORT_END,
            EMOJI_VARIATION_SELECTORS_START..EMOJI_VARIATION_SELECTORS_END,
            EMOJI_TECH_SYMBOLS_START..EMOJI_TECH_SYMBOLS_END,
            EMOJI_STARS_START..EMOJI_STARS_END,
        )

    // Zero Width Joiner (이모지 조합용)
    const val ZERO_WIDTH_JOINER = 0x200D
}
