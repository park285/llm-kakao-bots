package party.qwer.twentyq.redis

import org.redisson.api.RBucketReactive
import org.redisson.api.RExpirableReactive
import reactor.core.publisher.Mono
import java.time.Duration
import kotlinx.coroutines.reactor.awaitSingleOrNull as reactorAwaitSingleOrNull

suspend fun <T : Any> Mono<T>.awaitSingleOrNull(): T? = this.reactorAwaitSingleOrNull()

suspend fun RExpirableReactive.expireAsync(ttl: Duration): Boolean = this.expire(ttl).awaitSingleOrNull() ?: false

suspend fun <T> RBucketReactive<T>.setAsync(
    value: T,
    ttl: Duration,
) {
    this.set(value, ttl).awaitSingleOrNull()
}
