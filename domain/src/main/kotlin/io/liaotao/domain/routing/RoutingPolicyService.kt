/*
 * RoutingPolicyService.kt - fallback and retry routing policy.
 * Responsibilities: choose primary provider, apply ordered fallback sequence,
 * enforce bounded retries with backoff, and expose attempt-level statuses.
 */

package io.liaotao.domain.routing

import java.time.Instant
import java.util.UUID

enum class AttemptStatus {
    SUCCESS,
    FAILED,
}

data class ExecutionAttempt(
    val id: String,
    val providerId: String,
    val status: AttemptStatus,
    val startedAt: Instant,
    val finishedAt: Instant,
    val retryIndex: Int,
    val errorMessage: String? = null,
)

data class ProviderExecutionResult(
    val isSuccess: Boolean,
    val errorMessage: String? = null,
)

data class RoutingExecutionResult(
    val finalProvider: String?,
    val success: Boolean,
    val attempts: List<ExecutionAttempt>,
)

interface ExecutionHistoryRepository {
    fun recordAttempt(attempt: ExecutionAttempt)

    fun listRecent(limit: Int = 100): List<ExecutionAttempt>
}

class RoutingPolicyService(
    private val historyRepository: ExecutionHistoryRepository,
    private val nowProvider: () -> Instant = { Instant.now() },
    private val idProvider: () -> String = { UUID.randomUUID().toString() },
    private val sleep: (Long) -> Unit = { millis -> Thread.sleep(millis) },
) {
    fun executeWithFallback(
        primaryProvider: String,
        fallbackProviders: List<String>,
        maxRetries: Int,
        backoffMs: Long,
        executeProvider: (providerId: String) -> ProviderExecutionResult,
    ): RoutingExecutionResult {
        require(primaryProvider.isNotBlank()) { "Primary provider is required" }
        require(maxRetries >= 0) { "maxRetries must be >= 0" }
        require(backoffMs >= 0) { "backoffMs must be >= 0" }

        val orderedProviders = buildList {
            add(primaryProvider)
            fallbackProviders
                .map { it.trim() }
                .filter { it.isNotEmpty() }
                .filter { it != primaryProvider }
                .forEach { add(it) }
        }

        val attempts = mutableListOf<ExecutionAttempt>()

        for ((providerIndex, providerId) in orderedProviders.withIndex()) {
            for (retry in 0..maxRetries) {
                val started = nowProvider()
                val result = executeProvider(providerId)
                val finished = nowProvider()

                val attempt = ExecutionAttempt(
                    id = idProvider(),
                    providerId = providerId,
                    status = if (result.isSuccess) AttemptStatus.SUCCESS else AttemptStatus.FAILED,
                    startedAt = started,
                    finishedAt = finished,
                    retryIndex = retry,
                    errorMessage = result.errorMessage,
                )

                attempts.add(attempt)
                historyRepository.recordAttempt(attempt)

                if (result.isSuccess) {
                    return RoutingExecutionResult(
                        finalProvider = providerId,
                        success = true,
                        attempts = attempts,
                    )
                }

                val shouldRetry = retry < maxRetries
                if (shouldRetry) {
                    val delay = backoffMs * (retry + 1)
                    if (delay > 0) {
                        sleep(delay)
                    }
                }
            }

            val hasNextProvider = providerIndex < orderedProviders.lastIndex
            if (!hasNextProvider) {
                break
            }
        }

        return RoutingExecutionResult(
            finalProvider = null,
            success = false,
            attempts = attempts,
        )
    }
}