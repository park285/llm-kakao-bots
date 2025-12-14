package io.github.kapu.turtlesoup.service

import io.github.kapu.turtlesoup.config.PuzzleConfig
import io.github.kapu.turtlesoup.config.PuzzleConstants
import io.github.kapu.turtlesoup.config.PuzzleDedupConstants
import io.github.kapu.turtlesoup.models.Puzzle
import io.github.kapu.turtlesoup.models.PuzzleCategory
import io.github.kapu.turtlesoup.models.PuzzleGenerationRequest
import io.github.kapu.turtlesoup.redis.PuzzleDedupStore
import io.github.kapu.turtlesoup.rest.LlmRestClient
import io.github.kapu.turtlesoup.utils.PuzzleGenerationException
import io.github.kapu.turtlesoup.utils.RedisException
import io.github.oshai.kotlinlogging.KotlinLogging
import io.ktor.client.plugins.ResponseException
import kotlinx.serialization.SerializationException
import java.io.IOException
import java.security.MessageDigest
import java.util.Locale

class PuzzleService(
    private val restClient: LlmRestClient,
    private val puzzleConfig: PuzzleConfig,
    private val dedupStore: PuzzleDedupStore,
) {
    suspend fun generatePuzzle(
        request: PuzzleGenerationRequest,
        chatId: String,
    ): Puzzle {
        val effectiveRequest =
            PuzzleGenerationRequest(
                category = request.category ?: PuzzleCategory.MYSTERY,
                difficulty = request.difficulty ?: PuzzleConstants.DEFAULT_DIFFICULTY,
                theme = request.theme.orEmpty(),
            )
        var lastError: Exception? = null

        repeat(PuzzleDedupConstants.MAX_GENERATION_RETRIES) { attempt ->
            try {
                val puzzle = tryGeneratePuzzle(effectiveRequest, chatId, attempt)
                if (puzzle != null) return puzzle
            } catch (e: ResponseException) {
                lastError = PuzzleGenerationException("Failed to generate puzzle", e)
                logger.warn(e) { "puzzle_generate_failed_response attempt=${attempt + 1}" }
            } catch (e: SerializationException) {
                lastError = PuzzleGenerationException("Failed to generate puzzle", e)
                logger.warn(e) { "puzzle_generate_failed_serialization attempt=${attempt + 1}" }
            } catch (e: IOException) {
                lastError = PuzzleGenerationException("Failed to generate puzzle", e)
                logger.warn(e) { "puzzle_generate_failed_io attempt=${attempt + 1}" }
            } catch (e: RedisException) {
                lastError = PuzzleGenerationException("Failed to generate puzzle", e)
                logger.warn(e) { "puzzle_generate_failed_redis attempt=${attempt + 1}" }
            } catch (e: IllegalStateException) {
                lastError = PuzzleGenerationException("Failed to generate puzzle", e)
                logger.warn(e) { "puzzle_generate_failed_state attempt=${attempt + 1}" }
            }
        }

        val fallback = selectPresetPuzzle(effectiveRequest.difficulty)
        val signature = computeSignature(fallback)
        dedupStore.markUsed(signature, chatId)

        if (lastError != null) {
            logger.info { "puzzle_fallback_preset reason=generate_failed chat_id=$chatId" }
        } else {
            logger.info { "puzzle_fallback_preset reason=duplicate_exhausted chat_id=$chatId" }
        }

        return fallback
    }

    private suspend fun tryGeneratePuzzle(
        request: PuzzleGenerationRequest,
        chatId: String,
        attempt: Int,
    ): Puzzle? {
        val response = restClient.generatePuzzle(request)
        val puzzle = response.toPuzzle()

        if (!puzzle.hasRequiredContent()) {
            logger.warn { "puzzle_invalid_empty_fields attempt=${attempt + 1} chat_id=$chatId" }
            return null
        }

        val signature = computeSignature(puzzle)

        if (dedupStore.isDuplicate(signature, chatId)) {
            logger.warn { "puzzle_duplicate_detected attempt=${attempt + 1} chat_id=$chatId" }
            return null
        }

        dedupStore.markUsed(signature, chatId)
        logger.info { "puzzle_generated chat_id=$chatId difficulty=${puzzle.difficulty}" }
        return puzzle
    }

    suspend fun getRandomPresetPuzzle(): Puzzle {
        val presetResult = restClient.getRandomPuzzle()
        val basePuzzle = presetResult.toPuzzle()
        return applyRewriteIfEnabled(basePuzzle)
    }

    suspend fun getPresetPuzzleByDifficulty(difficulty: Int): Puzzle {
        val presetResult = restClient.getRandomPuzzle(difficulty)
        val basePuzzle = presetResult.toPuzzle()
        return applyRewriteIfEnabled(basePuzzle)
    }

    @Suppress("TooGenericExceptionCaught")
    private suspend fun applyRewriteIfEnabled(puzzle: Puzzle): Puzzle {
        if (!puzzleConfig.rewriteEnabled) {
            logger.info { "preset_puzzle_selected title=${puzzle.title} rewrite=false" }
            return puzzle
        }

        return try {
            logger.info { "rewriting_puzzle title=${puzzle.title}" }
            val rewriteResult =
                restClient.rewriteScenario(
                    title = puzzle.title,
                    scenario = puzzle.scenario,
                    solution = puzzle.solution,
                    difficulty = puzzle.difficulty,
                )
            val rewrittenPuzzle =
                puzzle.copy(
                    scenario = rewriteResult.scenario,
                    solution = rewriteResult.solution,
                )
            logger.info { "puzzle_rewritten title=${puzzle.title}" }
            rewrittenPuzzle
        } catch (
            e: io.ktor.client.plugins.ResponseException,
        ) {
            logger.warn(e) { "rewrite_failed title=${puzzle.title} using_original" }
            puzzle
        } catch (e: kotlinx.serialization.SerializationException) {
            logger.warn(e) { "rewrite_failed_serialization title=${puzzle.title} using_original" }
            puzzle
        } catch (e: java.io.IOException) {
            logger.warn(e) { "rewrite_failed_io title=${puzzle.title} using_original" }
            puzzle
        } catch (e: Exception) {
            logger.warn(e) { "rewrite_failed_unexpected title=${puzzle.title} using_original" }
            puzzle
        }
    }

    private suspend fun selectPresetPuzzle(difficulty: Int?): Puzzle =
        if (difficulty != null) {
            getPresetPuzzleByDifficulty(difficulty)
        } else {
            getRandomPresetPuzzle()
        }

    private fun computeSignature(puzzle: Puzzle): String {
        val normalized =
            listOf(
                puzzle.title,
                puzzle.scenario,
                puzzle.solution,
                puzzle.hints.joinToString("|"),
            ).joinToString("|") { it.trim().lowercase(Locale.ROOT) }

        val digest = MessageDigest.getInstance("SHA-256").digest(normalized.toByteArray())
        return digest.joinToString("") { "%02x".format(it) }
    }

    private fun Puzzle.hasRequiredContent(): Boolean =
        title.isNotBlank() && scenario.isNotBlank() && solution.isNotBlank()

    companion object {
        private val logger = KotlinLogging.logger {}
    }
}
