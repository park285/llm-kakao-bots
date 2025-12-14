package party.qwer.twentyq.service.riddle

import org.springframework.stereotype.Component
import party.qwer.twentyq.redis.tracking.HintCountStore
import party.qwer.twentyq.redis.tracking.TopicHistoryStore
import party.qwer.twentyq.util.game.GameMessageProvider

@Component
data class AnswerSuccessConfig(
    val topicHistoryStore: TopicHistoryStore,
    val hintCountStore: HintCountStore,
    val gameMessageProvider: GameMessageProvider,
)
