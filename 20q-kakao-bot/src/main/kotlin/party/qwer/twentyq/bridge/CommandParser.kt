package party.qwer.twentyq.bridge

import party.qwer.twentyq.model.Command
import party.qwer.twentyq.model.UsagePeriod
import party.qwer.twentyq.rest.NlpRestClient
import party.qwer.twentyq.util.game.constants.GameConstants

class CommandParser(
    private val prefix: String,
    private val nlpRestClient: NlpRestClient,
) {
    private val chainedQuestionParser = ChainedQuestionCommandParser(nlpRestClient)

    suspend fun parse(msg: String?): Command? {
        if (msg.isNullOrBlank()) return null
        val t = msg.trim()
        val p = Regex.escape(prefix)

        return parseAdmin(t)
            ?: parseStart(t, p)
            ?: parseHint(t, p)
            ?: parseUsage(t, p)
            ?: parseUserStats(t, p)
            ?: parseSimple(t, p)
            ?: chainedQuestionParser.parse(t, p)
            ?: parseAsk(t, p)
    }

    private fun parseAdmin(t: String): Command? {
        // Prefix 없이 사용 가능한 관리자 명령어
        val adminPairs =
            listOf(
                Regex("^/살자해라$", RegexOption.IGNORE_CASE) to Command.AdminRestartAll,
            )
        return adminPairs.firstOrNull { it.first.containsMatchIn(t) }?.second
    }

    private fun parseStart(
        t: String,
        p: String,
    ): Command? {
        // 공백으로 구분된 여러 카테고리 캡처 지원
        val startRe = Regex("^$p\\s*(?:start|시작)(?:\\s+(.+))?$", RegexOption.IGNORE_CASE)
        val m = startRe.find(t) ?: return null
        val categoriesRaw = m.groups[1]?.value?.trim()
        val categories =
            categoriesRaw
                ?.split("\\s+".toRegex())
                ?.map { it.trim() }
                ?.filter { it.isNotBlank() }
                ?.takeIf { it.isNotEmpty() }
        return Command.Start(categories)
    }

    private fun parseHint(
        t: String,
        p: String,
    ): Command? {
        val hintRe = Regex("^$p\\s*(?:hint|힌트)(?:\\s+(\\d+))?$", RegexOption.IGNORE_CASE)
        val m = hintRe.find(t) ?: return null
        val count = m.groups[1]?.value?.toIntOrNull() ?: 1
        return Command.Hints(count.coerceIn(GameConstants.MIN_HINT_REQUEST, GameConstants.MAX_HINT_REQUEST))
    }

    private fun parseSimple(
        t: String,
        p: String,
    ): Command? {
        val helpRe = Regex("^$p\\s*$")
        if (helpRe.containsMatchIn(t)) return Command.Help

        val simplePairs =
            listOf(
                Regex("^$p\\s*(?:surrender|하남자)$", RegexOption.IGNORE_CASE) to Command.Surrender,
                Regex("^$p\\s*(?:agree|동의)$", RegexOption.IGNORE_CASE) to Command.Agree,
                Regex("^$p\\s*(?:reject|거부)$", RegexOption.IGNORE_CASE) to Command.Reject,
                Regex("^$p\\s*(?:status|상태)$", RegexOption.IGNORE_CASE) to Command.Status,
                Regex("^$p\\s*(?:admin\\s+force-end|관리자\\s+강제종료)$", RegexOption.IGNORE_CASE) to Command.AdminForceEnd,
                Regex("^$p\\s*(?:admin\\s+clear-all|관리자\\s+전체삭제)$", RegexOption.IGNORE_CASE) to Command.AdminClearAll,
                Regex(
                    "^$p\\s*(?:admin\\s+refresh-cache|관리자\\s+캐싱)$",
                    RegexOption.IGNORE_CASE,
                ) to Command.AdminRefreshCache,
                Regex("^$p\\s*(?:살자해라|restart-all)$", RegexOption.IGNORE_CASE) to Command.AdminRestartAll,
                Regex("^$p\\s*(?:뒤졌냐|살아있냐|핑크)$", RegexOption.IGNORE_CASE) to Command.HealthCheck,
                Regex("^$p\\s*(?:모델|model)$", RegexOption.IGNORE_CASE) to Command.ModelInfo,
            )
        return simplePairs.firstOrNull { it.first.containsMatchIn(t) }?.second
    }

    /**
     * 전적 조회 명령어 파싱
     * "/스자 전적" - 본인 전적
     * "/스자 전적 닉네임" - 다른 사용자 전적
     * "/스자 전적 룸" - 방 전적 (전체)
     * "/스자 전적 룸 일간|주간|월간" - 방 전적 (기간별)
     */
    private fun parseUserStats(
        t: String,
        p: String,
    ): Command? {
        // 방 전적 조회 먼저 확인
        val roomStatsRe = Regex("^$p\\s*전적\\s+룸(?:\\s+(일간|주간|월간))?$", RegexOption.IGNORE_CASE)
        val roomMatch = roomStatsRe.find(t)
        if (roomMatch != null) {
            val period = roomMatch.groups[1]?.value?.trim()
            return Command.UserStats(targetNickname = null, roomPeriod = period ?: "")
        }

        // 개인 전적 조회
        val statsRe = Regex("^$p\\s*전적(?:\\s+(.+))?$", RegexOption.IGNORE_CASE)
        val match = statsRe.find(t) ?: return null
        val targetNickname = match.groups[1]?.value?.trim()
        return Command.UserStats(targetNickname = targetNickname, roomPeriod = null)
    }

    private fun parseUsage(
        t: String,
        p: String,
    ): Command? {
        // 기간 키워드
        val periodKeywords = "오늘|주간|월간|today|weekly|monthly"
        // 모델 키워드: flash, 2.5pro, 3.0pro 등
        val modelKeywords = "flash|2\\.5pro|3\\.0pro|pro"

        val usageRe =
            Regex(
                "^$p\\s*(?:사용량|usage)(?:\\s+($periodKeywords))?(?:\\s+($modelKeywords))?$",
                RegexOption.IGNORE_CASE,
            )
        val m = usageRe.find(t) ?: return null

        val periodStr = m.groups[1]?.value?.lowercase()
        val period =
            when (periodStr) {
                "오늘", "today" -> UsagePeriod.TODAY
                "주간", "weekly" -> UsagePeriod.WEEKLY
                "월간", "monthly" -> UsagePeriod.MONTHLY
                else -> UsagePeriod.TODAY
            }

        val modelStr = m.groups[2]?.value?.lowercase()
        val modelOverride =
            when (modelStr) {
                "flash" -> "flash-25"
                "2.5pro" -> "pro-25"
                "3.0pro", "pro" -> "pro-30"
                else -> null
            }

        return Command.AdminUsage(period, modelOverride)
    }

    private fun parseAsk(
        t: String,
        p: String,
    ): Command? {
        val patterns =
            listOf(
                Regex("^$p\\s+(정답\\s+.+)$", RegexOption.IGNORE_CASE),
                Regex("^$p\\s*(?:ask|\\?|질문)\\s+(.+)$", RegexOption.IGNORE_CASE),
                Regex("^$p\\s+(.+)$"),
            )
        for (re in patterns) {
            val m = re.find(t)
            val q =
                m
                    ?.groups
                    ?.get(1)
                    ?.value
                    ?.trim()
            if (!q.isNullOrBlank()) return Command.Ask(q)
        }
        return null
    }
}
