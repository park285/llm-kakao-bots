package party.qwer.twentyq.util

import org.junit.jupiter.api.Assertions.assertEquals
import org.junit.jupiter.api.Assertions.assertFalse
import org.junit.jupiter.api.Assertions.assertNull
import org.junit.jupiter.api.Assertions.assertTrue
import org.junit.jupiter.api.Test
import party.qwer.twentyq.util.common.extensions.chunkedByLines
import party.qwer.twentyq.util.common.extensions.endsWithIgnoreCase
import party.qwer.twentyq.util.common.extensions.fillTemplate
import party.qwer.twentyq.util.common.extensions.isValidCategory
import party.qwer.twentyq.util.common.extensions.isValidQuestion
import party.qwer.twentyq.util.common.extensions.limitLines
import party.qwer.twentyq.util.common.extensions.maskSensitive
import party.qwer.twentyq.util.common.extensions.maskToken
import party.qwer.twentyq.util.common.extensions.normalizeForComparison
import party.qwer.twentyq.util.common.extensions.normalizeKakaoText
import party.qwer.twentyq.util.common.extensions.normalizeWhitespace
import party.qwer.twentyq.util.common.extensions.nullIfEmpty
import party.qwer.twentyq.util.common.extensions.parseYesNo
import party.qwer.twentyq.util.common.extensions.safeSubstring
import party.qwer.twentyq.util.common.extensions.smartTrim
import party.qwer.twentyq.util.common.extensions.startsWithIgnoreCase
import party.qwer.twentyq.util.common.extensions.toCategoryIcon
import party.qwer.twentyq.util.common.extensions.toKoreanAnswer
import party.qwer.twentyq.util.common.extensions.truncate

/**
 * StringExtensions.kt 단위 테스트
 *
 * 테스트 범위:
 * - 질문 검증 (isValidQuestion, normalizeForComparison)
 * - 텍스트 정규화 (normalizeWhitespace, smartTrim, normalizeKakaoText)
 * - 민감정보 마스킹 (maskSensitive, maskToken)
 * - 카테고리 처리 (isValidCategory, toCategoryIcon)
 * - 답변 파싱 (parseYesNo, toKoreanAnswer)
 * - 프롬프트 처리 (fillTemplate, limitLines)
 * - 유틸리티 함수들
 */
class StringExtensionsTest {
    @Test
    fun `isValidQuestion - should return true for valid Korean question`() {
        // Given: 정상적인 한글 질문
        val validQuestion = "이것은 동물인가요?"

        // When: 검증 수행
        val result = validQuestion.isValidQuestion()

        // Then: true 반환
        assertTrue(result)
    }

    @Test
    fun `isValidQuestion - should return true for valid English question`() {
        // Given: 정상적인 영문 질문
        val validQuestion = "Is this an animal?"

        // When: 검증 수행
        val result = validQuestion.isValidQuestion()

        // Then: true 반환
        assertTrue(result)
    }

    @Test
    fun `isValidQuestion - should return true for mixed Korean and English`() {
        // Given: 한영 혼합 질문
        val mixedQuestion = "이것은 AI인가요?"

        // When: 검증 수행
        val result = mixedQuestion.isValidQuestion()

        // Then: true 반환
        assertTrue(result)
    }

    @Test
    fun `isValidQuestion - should return false for too short text`() {
        // Given: 4자 이하 텍스트
        val shortText = "짧아"

        // When: 검증 수행
        val result = shortText.isValidQuestion()

        // Then: false 반환
        assertFalse(result)
    }

    @Test
    fun `isValidQuestion - should accept exactly 5 characters`() {
        // Given: 정확히 5자
        val boundaryQuestion = "5자질문?"

        // When: 검증 수행
        val result = boundaryQuestion.isValidQuestion()

        // Then: true 반환 (경계값)
        assertTrue(result)
    }

    @Test
    fun `isValidQuestion - should accept exactly 100 characters`() {
        // Given: 정확히 100자 질문
        val longQuestion = "가".repeat(100)

        // When: 검증 수행
        val result = longQuestion.isValidQuestion()

        // Then: true 반환 (경계값)
        assertTrue(result)
    }

    @Test
    fun `isValidQuestion - should reject over 100 characters`() {
        // Given: 101자 이상 질문
        val tooLong = "가".repeat(101)

        // When: 검증 수행
        val result = tooLong.isValidQuestion()

        // Then: false 반환
        assertFalse(result)
    }

    @Test
    fun `isValidQuestion - should reject special characters`() {
        // Given: 특수문자 포함 (허용되지 않는 문자)
        val specialChars = "이것은 @#$% 질문인가요?"

        // When: 검증 수행
        val result = specialChars.isValidQuestion()

        // Then: false 반환
        assertFalse(result)
    }

    @Test
    fun `normalizeForComparison - should normalize to lowercase`() {
        // Given: 대소문자 혼합 텍스트
        val mixedCase = "This Is A Question"

        // When: 정규화 수행
        val result = mixedCase.normalizeForComparison()

        // Then: 소문자로 변환
        assertEquals("thisisaquestion", result)
    }

    @Test
    fun `normalizeForComparison - should remove all non-alphanumeric characters`() {
        // Given: 특수문자 및 공백 포함
        val withSpecialChars = "이것은 질문인가요? !"

        // When: 정규화 수행
        val result = withSpecialChars.normalizeForComparison()

        // Then: 한글과 영숫자만 남음
        assertEquals("이것은질문인가요", result)
    }

    @Test
    fun `normalizeForComparison - should handle empty string`() {
        // Given: 빈 문자열
        val empty = ""

        // When: 정규화 수행
        val result = empty.normalizeForComparison()

        // Then: 빈 문자열 반환
        assertEquals("", result)
    }

    @Test
    fun `normalizeWhitespace - should replace multiple spaces with single space`() {
        // Given: 연속된 공백
        val multipleSpaces = "여러    공백이    있는    문장"

        // When: 정규화 수행
        val result = multipleSpaces.normalizeWhitespace()

        // Then: 단일 공백으로 변환
        assertEquals("여러 공백이 있는 문장", result)
    }

    @Test
    fun `normalizeWhitespace - should trim leading and trailing spaces`() {
        // Given: 앞뒤 공백
        val withSpaces = "   앞뒤 공백   "

        // When: 정규화 수행
        val result = withSpaces.normalizeWhitespace()

        // Then: 앞뒤 공백 제거
        assertEquals("앞뒤 공백", result)
    }

    @Test
    fun `normalizeWhitespace - should handle tabs and newlines`() {
        // Given: 탭과 줄바꿈
        val withWhitespace = "탭\t과\n줄바꿈"

        // When: 정규화 수행
        val result = withWhitespace.normalizeWhitespace()

        // Then: 단일 공백으로 통일
        assertEquals("탭 과 줄바꿈", result)
    }

    @Test
    fun `smartTrim - should trim each line and remove blank lines`() {
        // Given: 여러 줄 텍스트
        val multiline =
            """
            첫 번째 줄  
            
            두 번째 줄  
                세 번째 줄
            """.trimIndent()

        // When: smartTrim 수행
        val result = multiline.smartTrim()

        // Then: 각 줄 trim, 빈 줄 제거
        assertEquals("첫 번째 줄\n두 번째 줄\n세 번째 줄", result)
    }

    @Test
    fun `normalizeKakaoText - should remove zero-width characters`() {
        // Given: Zero-width 문자 포함
        val withZeroWidth = "텍스트\u200B여기\u200C있음\u200D"

        // When: 정규화 수행
        val result = withZeroWidth.normalizeKakaoText()

        // Then: Zero-width 문자 제거
        assertEquals("텍스트여기있음", result)
    }

    @Test
    fun `normalizeKakaoText - should remove emoji`() {
        // Given: 이모지 포함 텍스트
        val withEmoji = "안녕하세요 🎉 반갑습니다 👋"

        // When: 정규화 수행
        val result = withEmoji.normalizeKakaoText()

        // Then: 이모지 제거, 공백 정규화
        assertEquals("안녕하세요 반갑습니다", result)
    }

    @Test
    fun `maskSensitive - should mask fully when 4 characters or less`() {
        // Given: 4자 이하 텍스트
        val shortText = "1234"

        // When: 마스킹 수행
        val result = shortText.maskSensitive()

        // Then: 전체 마스킹
        assertEquals("****", result)
    }

    @Test
    fun `maskSensitive - should show first 2 and last 2 characters when 5 or more`() {
        // Given: 5자 이상 텍스트
        val longText = "12345678"

        // When: 마스킹 수행
        val result = longText.maskSensitive()

        // Then: 앞 2자 + 마스킹 + 뒤 2자
        assertEquals("12****78", result)
    }

    @Test
    fun `maskSensitive - should handle empty string`() {
        // Given: 빈 문자열
        val empty = ""

        // When: 마스킹 수행
        val result = empty.maskSensitive()

        // Then: 빈 문자열 반환
        assertEquals("", result)
    }

    @Test
    fun `maskSensitive - should handle exactly 5 characters`() {
        // Given: 정확히 5자
        val fiveChars = "abcde"

        // When: 마스킹 수행
        val result = fiveChars.maskSensitive()

        // Then: 앞2 + 마스킹1 + 뒤2
        assertEquals("ab*de", result)
    }

    @Test
    fun `maskToken - should mask fully when 10 characters or less`() {
        // Given: 10자 이하 토큰
        val shortToken = "short-key"

        // When: 마스킹 수행
        val result = shortToken.maskToken()

        // Then: 전체 마스킹
        assertEquals("*********", result)
    }

    @Test
    fun `maskToken - should show first 6 and last 4 when over 10 characters`() {
        // Given: 긴 API 토큰
        val apiKey = "AIzaSyArCv7_jikqCeVVNFsslLeivp26Ogt1L-c"

        // When: 마스킹 수행
        val result = apiKey.maskToken()

        // Then: 앞6 + ... + 뒤4
        assertEquals("AIzaSy...1L-c", result)
    }

    @Test
    fun `maskToken - should handle empty token`() {
        // Given: 빈 토큰
        val empty = ""

        // When: 마스킹 수행
        val result = empty.maskToken()

        // Then: 빈 문자열 반환
        assertEquals("", result)
    }

    @Test
    fun `isValidCategory - should return true for valid categories`() {
        // Given: 유효한 카테고리들
        val validCategories = listOf("인물", "사물", "동물", "장소", "음식", "추상")

        // When/Then: 모두 true 반환
        validCategories.forEach { category ->
            assertTrue(category.isValidCategory(), "카테고리 '$category'는 유효해야 함")
        }
    }

    @Test
    fun `isValidCategory - should return false for invalid category`() {
        // Given: 유효하지 않은 카테고리
        val invalidCategory = "무효한카테고리"

        // When: 검증 수행
        val result = invalidCategory.isValidCategory()

        // Then: false 반환
        assertFalse(result)
    }

    @Test
    fun `toCategoryIcon - should map categories to correct icons`() {
        // Given/When/Then: 각 카테고리의 아이콘 검증
        assertEquals("👤", "인물".toCategoryIcon())
        assertEquals("📦", "사물".toCategoryIcon())
        assertEquals("🐾", "동물".toCategoryIcon())
        assertEquals("📍", "장소".toCategoryIcon())
        assertEquals("🍽️", "음식".toCategoryIcon())
        assertEquals("💭", "추상".toCategoryIcon())
    }

    @Test
    fun `toCategoryIcon - should return question mark for unknown category`() {
        // Given: 알 수 없는 카테고리
        val unknownCategory = "알수없음"

        // When: 아이콘 조회
        val result = unknownCategory.toCategoryIcon()

        // Then: 물음표 반환
        assertEquals("❓", result)
    }

    @Test
    fun `parseYesNo - should return true for affirmative answers`() {
        // Given: 긍정 답변들
        val yesAnswers = listOf("네", "예", "yes", "y", "YES", "맞아", "맞습니다")

        // When/Then: 모두 true 반환
        yesAnswers.forEach { answer ->
            assertEquals(true, answer.parseYesNo(), "답변 '$answer'는 true여야 함")
        }
    }

    @Test
    fun `parseYesNo - should return false for negative answers`() {
        // Given: 부정 답변들
        val noAnswers = listOf("아니", "아니오", "no", "n", "NO", "아니야", "틀려")

        // When/Then: 모두 false 반환
        noAnswers.forEach { answer ->
            assertEquals(false, answer.parseYesNo(), "답변 '$answer'는 false여야 함")
        }
    }

    @Test
    fun `parseYesNo - should return null for ambiguous answer`() {
        // Given: 모호한 답변
        val ambiguous = "모르겠어요"

        // When: 파싱 수행
        val result = ambiguous.parseYesNo()

        // Then: null 반환
        assertNull(result)
    }

    @Test
    fun `parseYesNo - should trim whitespace before parsing`() {
        // Given: 공백 포함 답변
        val withSpaces = "  네  "

        // When: 파싱 수행
        val result = withSpaces.parseYesNo()

        // Then: true 반환 (공백 무시)
        assertEquals(true, result)
    }

    @Test
    fun `toKoreanAnswer - should convert true to 네`() {
        // Given: yes 답변
        val yesAnswer = "yes"

        // When: 한국어로 변환
        val result = yesAnswer.toKoreanAnswer()

        // Then: "네" 반환
        assertEquals("네", result)
    }

    @Test
    fun `toKoreanAnswer - should convert false to 아니오`() {
        // Given: no 답변
        val noAnswer = "no"

        // When: 한국어로 변환
        val result = noAnswer.toKoreanAnswer()

        // Then: "아니오" 반환
        assertEquals("아니오", result)
    }

    @Test
    fun `toKoreanAnswer - should keep ambiguous answer as is`() {
        // Given: 모호한 답변
        val ambiguous = "잘 모르겠어요"

        // When: 변환 시도
        val result = ambiguous.toKoreanAnswer()

        // Then: 원본 유지
        assertEquals("잘 모르겠어요", result)
    }

    @Test
    fun `fillTemplate - should replace placeholders with params`() {
        // Given: 플레이스홀더가 있는 템플릿
        val template = "정답은 \${answer}입니다. 카테고리는 \${category}입니다."
        val params = mapOf("answer" to "고양이", "category" to "동물")

        // When: 치환 수행
        val result = template.fillTemplate(params)

        // Then: 플레이스홀더가 값으로 치환됨
        assertEquals("정답은 고양이입니다. 카테고리는 동물입니다.", result)
    }

    @Test
    fun `fillTemplate - should handle numeric values`() {
        // Given: 숫자 파라미터
        val template = "질문 \${count}번째"
        val params = mapOf("count" to 10)

        // When: 치환 수행
        val result = template.fillTemplate(params)

        // Then: 숫자가 문자열로 치환됨
        assertEquals("질문 10번째", result)
    }

    @Test
    fun `fillTemplate - should handle empty params`() {
        // Given: 빈 파라미터
        val template = "파라미터 없음 \${missing}"
        val params = emptyMap<String, Any>()

        // When: 치환 수행
        val result = template.fillTemplate(params)

        // Then: 플레이스홀더 유지
        assertEquals("파라미터 없음 \${missing}", result)
    }

    @Test
    fun `limitLines - should limit to specified number of lines`() {
        // Given: 5줄 텍스트
        val multiline = "줄1\n줄2\n줄3\n줄4\n줄5"

        // When: 3줄로 제한
        val result = multiline.limitLines(3)

        // Then: 처음 3줄만 반환
        assertEquals("줄1\n줄2\n줄3", result)
    }

    @Test
    fun `limitLines - should return all lines when limit is greater`() {
        // Given: 3줄 텍스트
        val multiline = "줄1\n줄2\n줄3"

        // When: 10줄로 제한 (실제보다 많음)
        val result = multiline.limitLines(10)

        // Then: 모든 줄 반환
        assertEquals("줄1\n줄2\n줄3", result)
    }

    @Test
    fun `startsWithIgnoreCase - should ignore case when checking prefix`() {
        // Given: 대소문자 혼합 텍스트
        val text = "Hello World"

        // When/Then: 대소문자 무시하고 prefix 확인
        assertTrue(text.startsWithIgnoreCase("hello"))
        assertTrue(text.startsWithIgnoreCase("HELLO"))
        assertTrue(text.startsWithIgnoreCase("HeLLo"))
        assertFalse(text.startsWithIgnoreCase("world"))
    }

    @Test
    fun `endsWithIgnoreCase - should ignore case when checking suffix`() {
        // Given: 대소문자 혼합 텍스트
        val text = "Hello World"

        // When/Then: 대소문자 무시하고 suffix 확인
        assertTrue(text.endsWithIgnoreCase("world"))
        assertTrue(text.endsWithIgnoreCase("WORLD"))
        assertTrue(text.endsWithIgnoreCase("WoRLd"))
        assertFalse(text.endsWithIgnoreCase("hello"))
    }

    @Test
    fun `safeSubstring - should prevent index out of bounds`() {
        // Given: 짧은 문자열
        val text = "안녕"

        // When: 범위를 벗어나는 substring 시도
        val result = text.safeSubstring(0, 10)

        // Then: 문자열 끝까지만 반환
        assertEquals("안녕", result)
    }

    @Test
    fun `safeSubstring - should handle negative start index`() {
        // Given: 문자열
        val text = "안녕하세요"

        // When: 음수 startIndex
        val result = text.safeSubstring(-5, 3)

        // Then: 0부터 시작
        assertEquals("안녕하", result)
    }

    @Test
    fun `truncate - should add ellipsis when text is too long`() {
        // Given: 긴 텍스트
        val longText = "이것은 매우 긴 텍스트입니다"

        // When: 11자로 truncate
        val result = longText.truncate(11)

        // Then: 11자 (take(8) + "..." = 11자)
        assertEquals("이것은 매우 긴...", result)
    }

    @Test
    fun `truncate - should return original when within limit`() {
        // Given: 짧은 텍스트
        val shortText = "짧은 텍스트"

        // When: 20자로 truncate
        val result = shortText.truncate(20)

        // Then: 원본 그대로 반환
        assertEquals("짧은 텍스트", result)
    }

    @Test
    fun `truncate - should use custom suffix`() {
        // Given: 긴 텍스트
        val longText = "긴 텍스트입니다"

        // When: 커스텀 suffix로 truncate (6자로 제한)
        val result = longText.truncate(6, suffix = ">>>")

        // Then: 커스텀 suffix 사용 (take(3) + ">>>" = 6자)
        assertEquals("긴 텍>>>", result)
    }

    @Test
    fun `nullIfEmpty - should return null for empty string`() {
        // Given: 빈 문자열
        val empty = ""

        // When: nullIfEmpty 호출
        val result = empty.nullIfEmpty()

        // Then: null 반환
        assertNull(result)
    }

    @Test
    fun `nullIfEmpty - should return string for non-empty`() {
        // Given: 비어있지 않은 문자열
        val nonEmpty = "내용 있음"

        // When: nullIfEmpty 호출
        val result = nonEmpty.nullIfEmpty()

        // Then: 원본 문자열 반환
        assertEquals("내용 있음", result)
    }

    @Test
    fun `ifEmpty - should return default when empty`() {
        // Given: 빈 문자열
        val empty = ""

        // When: ifEmpty 호출
        val result = empty.ifEmpty { "기본값" }

        // Then: 기본값 반환
        assertEquals("기본값", result)
    }

    @Test
    fun `ifEmpty - should return original when not empty`() {
        // Given: 비어있지 않은 문자열
        val nonEmpty = "원본"

        // When: ifEmpty 호출
        val result = nonEmpty.ifEmpty { "기본값" }

        // Then: 원본 반환
        assertEquals("원본", result)
    }

    @Test
    fun `chunkedByLines - should not split when text is within maxLength`() {
        // Given: maxLength 이내의 짧은 텍스트
        val shortText = "짧은 텍스트"

        // When: 분할 수행
        val result = shortText.chunkedByLines(500)

        // Then: 분할 없이 단일 chunk 반환
        assertEquals(1, result.size)
        assertEquals("짧은 텍스트", result[0])
    }

    @Test
    fun `chunkedByLines - should split multiline text by lines`() {
        // Given: maxLength 초과하는 여러 줄 텍스트
        val multiline = "첫 번째 줄입니다.\n두 번째 줄입니다.\n세 번째 줄입니다."

        // When: 20자로 분할
        val result = multiline.chunkedByLines(20)

        // Then: 줄 단위로 분할됨
        assertTrue(result.size >= 2, "최소 2개 chunk로 분할되어야 함")
        assertTrue(result.all { it.length <= 20 }, "모든 chunk는 20자 이하")
    }

    @Test
    fun `chunkedByLines - should truncate single line exceeding maxLength`() {
        // Given: maxLength 초과하는 단일 긴 줄
        val longLine = "가".repeat(100)

        // When: 10자로 분할
        val result = longLine.chunkedByLines(10)

        // Then: 10자로 잘려서 단일 chunk
        assertEquals(1, result.size)
        assertEquals(10, result[0].length)
    }

    @Test
    fun `chunkedByLines - should handle empty string`() {
        // Given: 빈 문자열
        val empty = ""

        // When: 분할 수행
        val result = empty.chunkedByLines(500)

        // Then: 빈 리스트 반환
        assertEquals(0, result.size)
    }

    @Test
    fun `chunkedByLines - should handle text exactly at maxLength`() {
        // Given: 정확히 maxLength인 텍스트
        val exactLength = "가".repeat(10)

        // When: 10자로 분할
        val result = exactLength.chunkedByLines(10)

        // Then: 단일 chunk (분할 불필요)
        assertEquals(1, result.size)
        assertEquals(10, result[0].length)
    }

    @Test
    fun `chunkedByLines - should combine multiple short lines into single chunk`() {
        // Given: 짧은 줄 여러 개 (합쳐도 maxLength 이내)
        val shortLines = "줄1\n줄2\n줄3"

        // When: 100자로 분할
        val result = shortLines.chunkedByLines(100)

        // Then: 단일 chunk로 합쳐짐
        assertEquals(1, result.size)
        assertEquals("줄1\n줄2\n줄3", result[0])
    }

    @Test
    fun `chunkedByLines - should use default maxLength of 500`() {
        // Given: 500자 이내 텍스트
        val text = "가".repeat(400)

        // When: 기본 maxLength(500) 사용
        val result = text.chunkedByLines()

        // Then: 단일 chunk 반환
        assertEquals(1, result.size)
    }
}
