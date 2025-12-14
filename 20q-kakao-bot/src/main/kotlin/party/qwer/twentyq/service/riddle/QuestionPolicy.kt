package party.qwer.twentyq.service.riddle

import com.github.benmanes.caffeine.cache.Cache
import com.ibm.icu.text.Normalizer2
import org.springframework.stereotype.Component
import party.qwer.twentyq.rest.NlpRestClient
import party.qwer.twentyq.util.cache.CacheBuilders
import party.qwer.twentyq.util.common.extensions.minutes

// 질문 정책 검증 (메타질문 탐지)
@Component
class QuestionPolicy(
    private val nlpRestClient: NlpRestClient,
) {
    companion object {
        private const val CACHE_MAX_SIZE = 5_000L

        // 캐시
        private val lengthCache: Cache<String, Boolean> =
            CacheBuilders
                .expireAfterWrite(CACHE_MAX_SIZE, 30.minutes, recordStats = true)

        private val prefixCache: Cache<String, Boolean> =
            CacheBuilders
                .expireAfterWrite(CACHE_MAX_SIZE, 30.minutes, recordStats = true)

        private val suffixCache: Cache<String, Boolean> =
            CacheBuilders
                .expireAfterWrite(CACHE_MAX_SIZE, 30.minutes, recordStats = true)

        private val indexCache: Cache<String, Boolean> =
            CacheBuilders
                .expireAfterWrite(CACHE_MAX_SIZE, 30.minutes, recordStats = true)

        // 길이 패턴
        private val digitUnit = Regex("(?s)(?:^|\\s)\\d+\\s*(글자|자|음절|토큰|문자|캐릭터|character|모음|자음|초성|중성|종성|받침)")
        private val koreanNumUnit =
            Regex(
                "(?s)(한|두|세|네|다섯|여섯|일곱|여덟|아홉|열)\\s*" +
                    "(글자|자|음절|토큰|문자|캐릭터|character|모음|자음|초성|중성|종성|받침)",
            )
        private val sinoKoreanNumUnit =
            Regex("(?s)(삼|사|오|육|칠|팔|구|십|십[일이삼사오육칠팔구]|[이삼사오육칠팔구]십(?:[일이삼사오육칠팔구])?)\\s*(글자|자|음절|토큰|문자|캐릭터|character|모음|자음|초성|중성|종성|받침)")

        private val reversedKoreanNumUnit =
            Regex("(?s)(음절|글자|자|토큰|문자|캐릭터|character|모음|자음|초성|중성|종성|받침)[이가]?\\s*(한|두|세|네|다섯|여섯|일곱|여덟|아홉|열)개?")
        private val reversedSinoKoreanNumUnit =
            Regex(
                "(?s)(음절|글자|자|토큰|문자|캐릭터|character|모음|자음|초성|중성|종성|받침)[이가]?\\s*(삼|사|오|육|칠|팔|구|십|십[일이삼사오육칠팔구]|[이삼사오육칠팔구]십(?:[일이삼사오육칠팔구])?)개?",
            )
        private val howManyUnit = Regex("(?s)몇\\s*(글자|자|음절|토큰|문자|캐릭터|character|모음|자음|초성|중성|종성|받침)")

        private val definitionBasedUnit = Regex("(?s)(소리.*?단위|발음.*?단위|한\\s*번.*?단위|음.*?단위)")
        private val numComparePattern =
            Regex("(?s)(\\d+|한|두|세|네|다섯|여섯|일곱|여덟|아홉|열|삼|사|오|육|칠|팔|구|십)\\s*개?\\s*(이하|이상|초과|미만|넘|이내|안|밖|less|more|under|over)")
        private val segmentationPattern = Regex("(?s)(분절|나누|구분|쪼개|토막).*?(\\d+|한|두|세|몇).*?(개|이하|이상|초과|미만)")
        private val charCountWords = Regex("(?s)(글자수|글자 수|자릿수|자리수|자리 수)")
        private val codeLenCall = Regex("(?is)(?:len|length)\\s*\\([^)]+\\)\\s*(?:==|=|<=|>=|<|>)\\s*\\d+")
        private val englishLengthOf = Regex("(?is)length\\s+of\\s+(?:answer|target)\\s*(?:==|=|<=|>=|<|>)\\s*\\d+")
        private val inputCountPattern = Regex("(?s)(입력|타이핑|타자|치|누르).*?(\\d+|한|두|세|네|다섯|여섯|일곱|여덟|아홉|열|몇).*?(번|회|번째|이내|이하|이상|초과|미만|안)")
        private val keyboardMethodPattern =
            Regex("(?s)(키보드|두벌식|세벌식|쿼티|qwerty|dvorak|왼손|오른손).*?(입력|타이핑|타자|치).*?(\\d+|한|두|세|네|다섯|여섯|일곱|여덟|아홉|열|몇).*?(번|회|이내|이하|이상|안)")
        private val handCountPattern = Regex("(?s)(손|손가락|왼손|오른손).*?(\\d+|한|두|세|네|다섯|여섯|일곱|여덟|아홉|열|몇).*?(번|회|이내|이하|이상|안)")

        // 자모 단순 존재 질문
        private val simpleJamoPattern = Regex("(?s)(받침|초성|중성|종성)\\s*(?:있|없|가지|포함)")

        // 비표준 단위 (칸/슬롯/공간) - 양방향 매칭
        private val nonStandardUnitPattern =
            Regex(
                "(?s)(?:(?:\\d+|한|두|세|네|다섯|여섯|일곱|여덟|아홉|열|몇)\\s*(?:칸|공간|슬롯|위치|자리)|(?:칸|공간|슬롯|위치|자리)\\s*(?:\\d+|한|두|세|네|다섯|여섯|일곱|여덟|아홉|열|몇)\\s*개?)",
            )

        // 아라비아 숫자 + 개 (역순 패턴 보완)
        private val reversedDigitUnit = Regex("(?s)(음절|글자|자|토큰|문자)[이가]?\\s*\\d+\\s*개?")

        private const val KO_ANSWER_TERMS = "(정답|답변|답|타겟|타깃)"
        private val koAnswerLengthCompare =
            Regex("(?s)(?:\$KO_ANSWER_TERMS)(?:의)?\\s*(길이|글자수|글자 수|문자수|문자 수|음절수|음절 수|자릿수|자리수|자리 수)\\s*(?:은|는|이|가|==|=|<=|>=|<|>)?\\s*\\d+")
        private val rawJamoMultiChoice = Regex("(?is)(초성|쵸성|중성|종성|받침).*?[ㄱ-ㅎ]{2,}.*?(중에|중에서|중\\s*하나)")
        private val percentCompare =
            Regex("(?is)(\\d+\\s*%|\\d+\\s*퍼센트|percent).*?(이상|이하|초과|미만|over|under|more than|less than)")
        private val percentContext =
            Regex("(?is)(소유|보유|가지고|대부분|사람(?:들)?|사용|이용|인지|정답|정답률|정확도|유사|유사도|일치|확률|확신)")

        private val LENGTH_PATTERNS =
            listOf(
                digitUnit,
                koreanNumUnit,
                sinoKoreanNumUnit,
                reversedKoreanNumUnit,
                reversedSinoKoreanNumUnit,
                howManyUnit,
                definitionBasedUnit,
                numComparePattern,
                segmentationPattern,
                charCountWords,
                codeLenCall,
                englishLengthOf,
                koAnswerLengthCompare,
                inputCountPattern,
                keyboardMethodPattern,
                handCountPattern,
                simpleJamoPattern,
                nonStandardUnitPattern,
                reversedDigitUnit,
            )

        // 접두사 패턴
        private val prefixCharStart = Regex("(?is)(?:^|\\s)[\\p{L}\\p{N}][\\p{Z}\\p{P}]*으?로[\\p{Z}\\p{P}]*시작")
        private val whichLetterStart = Regex("(?s)(무슨|어떤)\\s*글자(?:로)?\\s*시작")
        private val prefixPhrases = Regex("(?s)(첫\\s*글자|첫글자|처음\\s*글자|머리\\s*글자|머리글자|초성|파열음|마찰음|파찰음|비음|유음|경음|된소리|거센소리|예사소리|평음|격음|유성음|무성음)")
        private val prefixPhonetic = Regex("(?s)[skptbdnmrlfvwzhjcq]\\s*(발음|소리|사운드|sound).*?시작")
        private val prefixEnglish = Regex("(?is)(starts\\s*with\\s*[\"']?[A-Za-z0-9]|first\\s*letter|initial\\s*(?:is|=)?)")

        private val PREFIX_PATTERNS =
            listOf(
                prefixCharStart,
                whichLetterStart,
                prefixPhrases,
                prefixPhonetic,
                prefixEnglish,
            )

        // 접미사 패턴
        private val suffixCharEnd = Regex("(?is)(?:^|\\s)[\\p{L}\\p{N}][\\p{Z}\\p{P}]*으?로[\\p{Z}\\p{P}]*끝")
        private val whichLetterEnd = Regex("(?s)(무슨|어떤)\\s*글자(?:로)?\\s*끝나")
        private val suffixPhrases = Regex("(?s)(끝\\s*글자|마지막\\s*글자|끝자리|마지막\\s*자리)")
        private val suffixEnglish = Regex("(?is)(ends\\s*with\\s*[\"']?[A-Za-z0-9]|last\\s*letter)")

        private val SUFFIX_PATTERNS =
            listOf(
                suffixCharEnd,
                whichLetterEnd,
                suffixPhrases,
                suffixEnglish,
            )

        // 인덱스 패턴
        private val nthKorean = Regex("(?s)(?:\\d+|한|두|세|네|다섯|여섯|일곱|여덟|아홉|열|열한|열두|스무)\\s*(?:번째|번)\\s*(글자|자|음절)")
        private val nthEnglish =
            Regex("(?is)(?:\\d+(?:st|nd|rd|th)?|first|second|third|fourth|fifth)\\s*(?:letter|char|character|syllable)")
        private val indexEnglish = Regex("(?is)(?:letter|char|character)\\s*(?:at|in)\\s*(?:position|index)\\s*\\d+")
        private val middleKorean = Regex("(?s)(중간\\s*글자|가운데\\s*글자|가운뎃\\s*글자|중간\\s*음절|가운데\\s*음절|중앙\\s*글자)")
        private val middleEnglish = Regex("(?is)(middle\\s*(?:letter|char|character|syllable))")
        private val nthJamoKorean = Regex("(?s)(초성|중성|종성)\\s*(?:의)?\\s*(?:몇\\s*번째|\\d+\\s*번째|\\d+\\s*번)")

        private val INDEX_PATTERNS =
            listOf(
                nthKorean,
                nthEnglish,
                indexEnglish,
                middleKorean,
                middleEnglish,
                nthJamoKorean,
            )

        // 정규화 패턴
        private val zeroWidthRegex = Regex("[\u200B-\u200D\u2060\uFE00-\uFE0F\uFEFF]")
        private val emojiAndModsRegex = Regex("""[\uFE0F\p{So}\p{Sk}]+""")
        private const val HANGUL_CLASS = "[가-힣ㄱ-ㅎㅏ-ㅣ]"
        private val hangulBetweenNoise = Regex("""(?<=\$HANGUL_CLASS)[\p{Z}\p{P}]+(?=\$HANGUL_CLASS)""")
        private val JAMO_AFTER_HANGUL_PATTERN = Regex("(?<=[가-힣])[ㄱ-ㅎㅏ-ㅣ]+")
        private val JAMO_BEFORE_HANGUL_PATTERN = Regex("[ㄱ-ㅎㅏ-ㅣ]+(?=[가-힣])")

        // 공통 정규화: NFKC + Format/Control 제거 + Zero-width 제거 + Emoji 제거 + Noise 제거
        private fun normalizeBase(q: String): String {
            val nfkc = Normalizer2.getNFKCInstance().normalize(q)
            val noCfCc =
                buildString(nfkc.length) {
                    nfkc.forEach { ch ->
                        val t = Character.getType(ch)
                        if (t != Character.FORMAT.toInt() && t != Character.CONTROL.toInt()) append(ch)
                    }
                }
            val removedZw = zeroWidthRegex.replace(noCfCc, "")
            val removedEmoji = emojiAndModsRegex.replace(removedZw, "")
            return hangulBetweenNoise.replace(removedEmoji, "")
        }

        private fun normalizeForPolicy(q: String): String {
            val base = normalizeBase(q)

            // Policy용 추가 처리: Jamo 제거
            val removed1 = JAMO_AFTER_HANGUL_PATTERN.replace(base, "")
            val removed2 = JAMO_BEFORE_HANGUL_PATTERN.replace(removed1, "")

            return removed2
        }

        private fun normalizeForBoundary(q: String): String = normalizeBase(q)

        // 캐시 키 정규화 (공백 제거)
        private fun normalizeForCache(q: String): String = q.trim()
    }

    suspend fun isAnswerLengthMetaQuestion(q: String): Boolean {
        val key = normalizeForCache(q)
        return lengthCache.getIfPresent(key) ?: run {
            val result = evaluateLengthMeta(q)
            lengthCache.put(key, result)
            result
        }
    }

    private suspend fun evaluateLengthMeta(q: String): Boolean {
        if (percentCompare.containsMatchIn(q) && percentContext.containsMatchIn(q)) {
            return true
        }
        val normalized = normalizeForPolicy(q).trim()

        return when {
            normalized.isEmpty() -> false
            LENGTH_PATTERNS.any { it.containsMatchIn(normalized) } -> true
            else ->
                runCatching {
                    val heuristics = nlpRestClient.analyzeHeuristics(q)
                    heuristics.numericQuantifier && heuristics.unitNoun
                }.getOrDefault(false)
        }
    }

    suspend fun isAnswerIndexMetaQuestion(q: String): Boolean {
        val key = normalizeForCache(q)
        return indexCache.getIfPresent(key) ?: run {
            val result = evaluateIndexMeta(q)
            indexCache.put(key, result)
            result
        }
    }

    private suspend fun evaluateIndexMeta(q: String): Boolean {
        val normalized = normalizeForPolicy(q).trim()
        if (normalized.isEmpty()) return false

        if (INDEX_PATTERNS.any { it.containsMatchIn(normalized) }) return true

        return runCatching {
            val heuristics = nlpRestClient.analyzeHeuristics(q)
            heuristics.boundaryRef && heuristics.unitNoun
        }.getOrDefault(false)
    }

    // 경계 메타
    suspend fun isAnswerBoundaryMetaQuestion(q: String): Boolean =
        rawJamoMultiChoice.containsMatchIn(q) ||
            isAnswerPrefixMetaQuestion(q) ||
            isAnswerSuffixMetaQuestion(q)

    private suspend fun isAnswerPrefixMetaQuestion(q: String): Boolean {
        val key = normalizeForCache(q)
        return prefixCache.getIfPresent(key) ?: run {
            val result = evaluatePrefixMeta(q)
            prefixCache.put(key, result)
            result
        }
    }

    private suspend fun evaluatePrefixMeta(q: String): Boolean {
        val normalized = normalizeForBoundary(q).trim()
        if (normalized.isEmpty()) return false

        return PREFIX_PATTERNS.any { it.containsMatchIn(normalized) }
    }

    private suspend fun isAnswerSuffixMetaQuestion(q: String): Boolean {
        val key = normalizeForCache(q)
        return suffixCache.getIfPresent(key) ?: run {
            val result = evaluateSuffixMeta(q)
            suffixCache.put(key, result)
            result
        }
    }

    private suspend fun evaluateSuffixMeta(q: String): Boolean {
        val normalized = normalizeForBoundary(q).trim()
        if (normalized.isEmpty()) return false

        return SUFFIX_PATTERNS.any { it.containsMatchIn(normalized) }
    }
}
