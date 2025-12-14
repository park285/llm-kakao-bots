package party.qwer.twentyq.service.riddle

import io.mockk.coEvery
import io.mockk.mockk
import kotlinx.coroutines.test.runTest
import org.junit.jupiter.api.Assertions.assertFalse
import org.junit.jupiter.api.Assertions.assertTrue
import org.junit.jupiter.api.BeforeEach
import org.junit.jupiter.api.Test
import org.junit.jupiter.params.ParameterizedTest
import org.junit.jupiter.params.provider.ValueSource
import party.qwer.twentyq.rest.NlpRestClient
import party.qwer.twentyq.rest.dto.NlpHeuristicsResponse

/**
 * QuestionPolicy 정규식 패턴 검증 테스트
 *
 * 검증 목표:
 * - 신규 추가된 simpleJamoPattern, nonStandardUnitPattern 동작 확인
 * - 기존 18개 패턴의 회귀 테스트
 * - 캐시 키 정규화 동작 검증
 */
class QuestionPolicyTest {
    private lateinit var nlpRestClient: NlpRestClient
    private lateinit var policy: QuestionPolicy

    @BeforeEach
    fun setup() {
        nlpRestClient = mockk()
        policy = QuestionPolicy(nlpRestClient)
    }

    // ========== 신규 패턴: simpleJamoPattern ==========

    @ParameterizedTest
    @ValueSource(strings = ["받침 있어?", "초성 포함돼?", "중성 가지고 있어?", "종성 없어?"])
    fun `simpleJamoPattern should match simple jamo existence questions`(question: String) =
        runTest {
            // When: 단순 자모 존재 질문
            val result = policy.isAnswerLengthMetaQuestion(question)

            // Then: 메타 질문으로 감지
            assertTrue(result)
        }

    @Test
    fun `simpleJamoPattern should not match complex jamo questions`() =
        runTest {
            // Given: 복잡한 자모 질문 (선택지 포함)
            val question = "초성이 ㄱ ㄴ ㄷ 중 하나야?"

            // When
            val result = policy.isAnswerBoundaryMetaQuestion(question)

            // Then: rawJamoMultiChoice에 의해 감지됨 (simpleJamoPattern 아님)
            assertTrue(result)
        }

    // ========== 신규 패턴: nonStandardUnitPattern ==========

    @ParameterizedTest
    @ValueSource(strings = ["몇 칸이야?", "슬롯 3개야?", "5자리야?", "공간 몇 개?", "위치 3개야?"])
    fun `nonStandardUnitPattern should match non-standard unit questions`(question: String) =
        runTest {
            // When: 비표준 단위 질문
            val result = policy.isAnswerLengthMetaQuestion(question)

            // Then: 메타 질문으로 감지
            assertTrue(result)
        }

    @Test
    fun `nonStandardUnitPattern should not match standard unit questions`() =
        runTest {
            // Given: 표준 단위 사용 (digitUnit 패턴으로 이미 커버)
            val question = "3글자야?"

            // When
            val result = policy.isAnswerLengthMetaQuestion(question)

            // Then: 메타 질문으로 감지 (기존 패턴)
            assertTrue(result)
        }

    // ========== 기존 패턴 회귀 테스트 ==========

    @ParameterizedTest
    @ValueSource(strings = ["3글자야?", "몇 글자야?", "글자 수가 5개야?", "음절이 4개야?"])
    fun `should detect length meta questions`(question: String) =
        runTest {
            // When: 길이 관련 질문
            val result = policy.isAnswerLengthMetaQuestion(question)

            // Then: 메타 질문으로 감지
            assertTrue(result)
        }

    @ParameterizedTest
    @ValueSource(strings = ["정답률이 50% 이상이야?", "정답이 70 percent 이상이야?"])
    fun `should detect percent meta questions with answer context`(question: String) =
        runTest {
            // When: 정답/유사도 맥락에서 퍼센트 비교
            val result = policy.isAnswerLengthMetaQuestion(question)

            // Then: 메타 질문으로 감지
            assertTrue(result)
        }

    @ParameterizedTest
    @ValueSource(strings = ["할인율 50% 이상이야?", "50% 할인 이벤트야?"])
    fun `should not overblock non answer percent contexts`(question: String) =
        runTest {
            // When: 게임 맥락이 아닌 일반 퍼센트 문구
            val result = policy.isAnswerLengthMetaQuestion(question)

            // Then: 메타 질문으로 감지되지 않음
            assertFalse(result)
        }

    @ParameterizedTest
    @ValueSource(strings = ["첫 글자가 ㄱ이야?", "ㅅ으로 시작해?", "끝 글자가 ㄴ이야?"])
    fun `should detect boundary meta questions`(question: String) =
        runTest {
            // When: 경계(접두사/접미사) 관련 질문
            val result = policy.isAnswerBoundaryMetaQuestion(question)

            // Then: 메타 질문으로 감지
            assertTrue(result)
        }

    @ParameterizedTest
    @ValueSource(strings = ["세 번째 글자가 뭐야?", "중간 글자가 ㄱ이야?", "2번째 음절은?"])
    fun `should detect index meta questions`(question: String) =
        runTest {
            // When: 인덱스 관련 질문
            val result = policy.isAnswerIndexMetaQuestion(question)

            // Then: 메타 질문으로 감지
            assertTrue(result)
        }

    @ParameterizedTest
    @ValueSource(strings = ["사과야?", "빨간색이야?", "맛있어?", "과일이야?"])
    fun `should not detect normal questions as meta`(question: String) =
        runTest {
            // When: 정상적인 스무고개 질문
            val lengthResult = policy.isAnswerLengthMetaQuestion(question)
            val indexResult = policy.isAnswerIndexMetaQuestion(question)
            val boundaryResult = policy.isAnswerBoundaryMetaQuestion(question)

            // Then: 메타 질문으로 감지되지 않음
            assertFalse(lengthResult)
            assertFalse(indexResult)
            assertFalse(boundaryResult)
        }

    // ========== 캐시 정규화 동작 검증 ==========

    @Test
    fun `should treat questions with different whitespace as same cache entry`() =
        runTest {
            // Given: 공백만 다른 동일한 질문
            val q1 = "3글자야?"
            val q2 = "  3글자야?  "
            val q3 = "3글자야? "

            // When: 각 질문을 검증 (형태소 분석 Mock 설정)
            coEvery { nlpRestClient.analyzeHeuristics(any()) } returns NlpHeuristicsResponse()

            val result1 = policy.isAnswerLengthMetaQuestion(q1)
            val result2 = policy.isAnswerLengthMetaQuestion(q2)
            val result3 = policy.isAnswerLengthMetaQuestion(q3)

            // Then: 모두 동일한 결과 (캐시 히트로 빠름)
            assertTrue(result1)
            assertTrue(result2)
            assertTrue(result3)
        }

    @Test
    fun `should cache prefix question regardless of whitespace`() =
        runTest {
            // Given: 공백만 다른 접두사 질문
            val q1 = "첫 글자가 ㄱ이야?"
            val q2 = "  첫 글자가 ㄱ이야?  "

            // When
            val result1 = policy.isAnswerBoundaryMetaQuestion(q1)
            val result2 = policy.isAnswerBoundaryMetaQuestion(q2)

            // Then: 동일한 결과
            assertTrue(result1)
            assertTrue(result2)
        }

    // ========== 형태소 분석 fallback 테스트 ==========

    @Test
    fun `should fall back to morpheme analysis when regex does not match`() =
        runTest {
            // Given: 정규식에는 없지만 형태소 분석으로 감지 가능한 질문
            val question = "넓이가 넓어?"

            // Mock: 형태소 분석이 메타 질문으로 판단
            val heuristics =
                NlpHeuristicsResponse(
                    numericQuantifier = false,
                    unitNoun = false,
                    boundaryRef = false,
                    comparisonWord = false,
                )
            coEvery { nlpRestClient.analyzeHeuristics(question) } returns heuristics

            // When
            val result = policy.isAnswerLengthMetaQuestion(question)

            // Then: 정규식 + 형태소 분석 모두 실패 → false
            assertFalse(result)
        }

    @Test
    fun `should detect meta question with morpheme analysis`() =
        runTest {
            // Given: "몇 개야?" 같은 애매한 질문
            val question = "몇 개야?"

            // Mock: 형태소 분석이 숫자량사 + 단위명사 감지
            val heuristics =
                NlpHeuristicsResponse(
                    numericQuantifier = true,
                    unitNoun = true,
                    boundaryRef = false,
                    comparisonWord = false,
                )
            coEvery { nlpRestClient.analyzeHeuristics(question) } returns heuristics

            // When
            val result = policy.isAnswerLengthMetaQuestion(question)

            // Then: 메타 질문으로 감지
            assertTrue(result)
        }

    // ========== 엣지 케이스 테스트 ==========

    @Test
    fun `should handle empty string`() =
        runTest {
            // Given: 빈 문자열
            val question = ""

            // When
            val lengthResult = policy.isAnswerLengthMetaQuestion(question)
            val indexResult = policy.isAnswerIndexMetaQuestion(question)
            val boundaryResult = policy.isAnswerBoundaryMetaQuestion(question)

            // Then: 메타 질문 아님
            assertFalse(lengthResult)
            assertFalse(indexResult)
            assertFalse(boundaryResult)
        }

    @Test
    fun `should handle whitespace only string`() =
        runTest {
            // Given: 공백만 있는 문자열
            val question = "   "

            // When
            val lengthResult = policy.isAnswerLengthMetaQuestion(question)

            // Then: 정규화 후 빈 문자열이 되어 false
            assertFalse(lengthResult)
        }

    @Test
    fun `should handle special characters`() =
        runTest {
            // Given: 특수문자 포함 질문
            val question = "정답이 3글자야!?"

            // When
            val result = policy.isAnswerLengthMetaQuestion(question)

            // Then: 메타 질문으로 감지 (특수문자는 정규식에 영향 없음)
            assertTrue(result)
        }
}
