package party.qwer.twentyq.util.common.json

import tools.jackson.core.type.TypeReference
import tools.jackson.databind.ObjectMapper

/** 자주 사용되는 TypeReference 상수 */
object JsonTypeRefs {
    /** Map<String, Any> 타입 */
    val STRING_ANY_MAP: TypeReference<Map<String, Any>> =
        object : TypeReference<Map<String, Any>>() {}
}

/** JSON description 문자열을 Map으로 파싱 (실패 시 null) */
fun ObjectMapper.parseDescriptionOrNull(json: String?): Map<String, Any>? =
    json?.let { desc ->
        runCatching {
            readValue(desc, JsonTypeRefs.STRING_ANY_MAP)
        }.getOrNull()
    }
