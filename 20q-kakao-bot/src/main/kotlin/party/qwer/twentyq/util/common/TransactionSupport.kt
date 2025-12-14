package party.qwer.twentyq.util.common

import kotlinx.coroutines.Job
import kotlinx.coroutines.reactive.awaitSingle
import kotlinx.coroutines.reactor.mono
import org.slf4j.Logger
import org.springframework.dao.OptimisticLockingFailureException
import org.springframework.transaction.reactive.TransactionalOperator
import kotlin.coroutines.coroutineContext

/**
 * R2DBC 트랜잭션 실행 (코루틴 지원)
 * - 블록 내 모든 작업이 하나의 트랜잭션으로 실행
 * - 예외 발생 시 자동 롤백
 */
suspend fun <T : Any> TransactionalOperator.executeAndAwait(block: suspend () -> T): T =
    this.transactional(mono(coroutineContext.minusKey(Job)) { block() }).awaitSingle()

/**
 * Optimistic locking 실패 시 재시도하는 트랜잭션 실행
 * @param maxRetries 최대 재시도 횟수 (기본: 3)
 * @param logTag 로깅용 태그
 * @param log 로거
 * @param block 트랜잭션 내에서 실행할 작업
 */
suspend fun <T : Any> TransactionalOperator.executeWithOptimisticRetry(
    maxRetries: Int = OPTIMISTIC_LOCK_MAX_RETRIES,
    logTag: String,
    log: Logger,
    block: suspend () -> T,
): Result<T> {
    var lastException: Throwable? = null

    repeat(maxRetries) { attempt ->
        runCatching { executeAndAwait(block) }
            .onSuccess { return Result.success(it) }
            .onFailure { ex ->
                lastException = ex
                if (ex !is OptimisticLockingFailureException) {
                    return Result.failure(ex)
                }
                log.warn(
                    "{} optimistic lock conflict, attempt={}/{}, retrying...",
                    logTag,
                    attempt + 1,
                    maxRetries,
                )
            }
    }

    return Result.failure(
        lastException ?: Exception("Unknown error after $maxRetries attempts"),
    )
}

private const val OPTIMISTIC_LOCK_MAX_RETRIES = 3
