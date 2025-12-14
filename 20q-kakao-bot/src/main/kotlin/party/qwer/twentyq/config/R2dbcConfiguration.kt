package party.qwer.twentyq.config

import io.r2dbc.postgresql.PostgresqlConnectionConfiguration
import io.r2dbc.postgresql.PostgresqlConnectionFactory
import io.r2dbc.spi.ConnectionFactory
import org.springframework.beans.factory.annotation.Value
import org.springframework.context.annotation.Bean
import org.springframework.context.annotation.Configuration
import org.springframework.data.r2dbc.config.AbstractR2dbcConfiguration
import java.time.Duration

/**
 * R2DBC PostgreSQL 설정
 */
@Configuration
class R2dbcConfiguration(
    @param:Value("\${DB_HOST:localhost}") private val host: String,
    @param:Value("\${DB_PORT:5432}") private val port: Int,
    @param:Value("\${DB_NAME:twentyq}") private val database: String,
    @param:Value("\${DB_USER:twentyq_app}") private val username: String,
    @param:Value("\${DB_PASSWORD:}") private val password: String,
) : AbstractR2dbcConfiguration() {
    companion object {
        private const val CONNECTION_TIMEOUT_SECONDS = 5L
    }

    @Bean
    override fun connectionFactory(): ConnectionFactory =
        PostgresqlConnectionFactory(
            PostgresqlConnectionConfiguration
                .builder()
                .host(host)
                .port(port)
                .database(database)
                .username(username)
                .password(password)
                .connectTimeout(Duration.ofSeconds(CONNECTION_TIMEOUT_SECONDS))
                .build(),
        )
}
