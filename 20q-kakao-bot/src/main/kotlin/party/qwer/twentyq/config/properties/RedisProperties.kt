package party.qwer.twentyq.config.properties

/**
 * Redis TTL 기본값 설정
 *
 * @property sessionTtlMinutes 세션 TTL (기본 30분)
 * @property veryLongTtlMinutes 매우 긴 TTL (기본 720분)
 * @property vectorEmbeddingMaxSize 벡터 임베딩 캐시 최대 크기 (기본 100,000)
 * @property vectorEmbeddingTtlHours 벡터 임베딩 캐시 TTL (기본 24시간)
 */
data class RedisDefaults(
    val sessionTtlMinutes: Long = 30,
    val veryLongTtlMinutes: Long = 720,
    val vectorEmbeddingMaxSize: Long = 100_000,
    val vectorEmbeddingTtlHours: Long = 24,
)

/**
 * Valkey Message Queue 설정
 *
 * @property host Redis 호스트
 * @property port Redis 포트
 * @property password Redis 비밀번호
 * @property timeout 연결 타임아웃 (ms)
 * @property consumerGroup Consumer 그룹 이름
 * @property consumerName Consumer 이름
 * @property streamKey Inbound 메시지 스트림 키
 * @property replyStreamKey Outbound 응답 스트림 키
 * @property maxQueueProcessIterations 큐 처리 최대 반복 횟수 (무한 루프 방지)
 */
data class ValkeyMQ(
    val host: String = "localhost",
    val port: Int = 1833,
    val password: String? = null,
    val timeout: Int = 5000,
    val connectionPoolSize: Int = 64,
    val connectionMinIdleSize: Int = 10,
    val consumerGroup: String = "20q-bot-group",
    val consumerName: String = "consumer-1",
    val streamKey: String = "kakao:20q",
    val replyStreamKey: String = "kakao:bot:reply",
    val maxQueueProcessIterations: Int = 100,
)
