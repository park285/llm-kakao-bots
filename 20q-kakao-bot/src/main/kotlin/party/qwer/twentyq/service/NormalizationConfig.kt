package party.qwer.twentyq.service

import org.springframework.beans.factory.annotation.Value
import org.springframework.stereotype.Component
import party.qwer.twentyq.config.AppProperties

/** 정규화 서비스 설정 */
@Component
data class NormalizationConfig(
    val appProperties: AppProperties,
    @param:Value("\${app.normalize.cache.enabled:true}")
    val cacheEnabled: Boolean,
)
