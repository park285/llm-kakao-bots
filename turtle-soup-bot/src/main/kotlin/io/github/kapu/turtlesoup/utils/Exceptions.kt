package io.github.kapu.turtlesoup.utils

/** 기본 예외 (sealed class - 타입 안전성 보장) */
sealed class TurtleSoupException(
    message: String,
    cause: Throwable? = null,
) : RuntimeException(message, cause) {
    init {
        cause?.let { initCause(it) }
    }
}

// Game exceptions
class SessionNotFoundException(
    sessionId: String,
) : TurtleSoupException("Session not found: $sessionId")

class InvalidQuestionException(
    message: String,
) : TurtleSoupException(message)

class GameAlreadyStartedException(
    sessionId: String,
) : TurtleSoupException("Game already started: $sessionId")

class GameNotStartedException(
    sessionId: String,
) : TurtleSoupException("Game not started: $sessionId")

class GameAlreadySolvedException(
    sessionId: String,
) : TurtleSoupException("Game already solved: $sessionId")

class MaxHintsReachedException : TurtleSoupException("Maximum hints reached")

// Puzzle exceptions
class PuzzleGenerationException(
    message: String,
    cause: Throwable? = null,
) : TurtleSoupException(message, cause)

class TruncatedJsonException(
    message: String,
) : TurtleSoupException(message)

// Redis exceptions
class RedisException(
    message: String,
    cause: Throwable? = null,
) : TurtleSoupException(message, cause)

class LockException(
    message: String,
    val holderName: String? = null,
) : TurtleSoupException(message)

// Access control exceptions
class AccessDeniedException(
    reason: String,
) : TurtleSoupException("Access denied: $reason")

class UserBlockedException(
    userId: String,
) : TurtleSoupException("User blocked: $userId")

class ChatBlockedException(
    chatId: String,
) : TurtleSoupException("Chat blocked: $chatId")

// Vote exceptions
class VoteNotFoundException(
    sessionId: String,
) : TurtleSoupException("Vote not found: $sessionId")

class VoteAlreadyActiveException(
    sessionId: String,
) : TurtleSoupException("Vote already active: $sessionId")

class AlreadyVotedException(
    userId: String,
) : TurtleSoupException("Already voted: $userId")

// Security exceptions
class InputInjectionException(
    message: String,
    val detectedPattern: String? = null,
) : TurtleSoupException(message)

class MalformedInputException(
    message: String,
) : TurtleSoupException(message)

/** 예상 가능한 사용자 행동으로 인한 예외 여부 (WARN 레벨 로깅용) */
fun TurtleSoupException.isExpectedUserBehavior(): Boolean =
    when (this) {
        is SessionNotFoundException,
        is GameNotStartedException,
        is GameAlreadyStartedException,
        is GameAlreadySolvedException,
        is MaxHintsReachedException,
        is VoteNotFoundException,
        is VoteAlreadyActiveException,
        is AlreadyVotedException,
        is InvalidQuestionException,
        is MalformedInputException,
        -> true
        // sealed class이므로 모든 케이스 처리 필요
        is PuzzleGenerationException,
        is TruncatedJsonException,
        is RedisException,
        is LockException,
        is AccessDeniedException,
        is UserBlockedException,
        is ChatBlockedException,
        is InputInjectionException,
        -> false
    }
