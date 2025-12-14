package party.qwer.twentyq.bridge

import org.springframework.stereotype.Component
import party.qwer.twentyq.redis.LockCoordinator
import party.qwer.twentyq.redis.ProcessingLockService

/**
 * 락 관련 의존성 묶음
 */
@Component
internal class LockingSupport(
    val lockCoordinator: LockCoordinator,
    val processingLockService: ProcessingLockService,
)
