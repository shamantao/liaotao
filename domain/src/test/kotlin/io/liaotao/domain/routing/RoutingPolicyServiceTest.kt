/*
 * RoutingPolicyServiceTest.kt - tests for fallback and retry policy.
 * Responsibilities: verify primary/fallback ordering, bounded retries,
 * and attempt history production for diagnostics.
 */

package io.liaotao.domain.routing

import java.time.Instant
import kotlin.test.Test
import kotlin.test.assertEquals
import kotlin.test.assertFalse
import kotlin.test.assertTrue

class RoutingPolicyServiceTest {
    @Test
    fun `uses fallback provider when primary fails`() {
        val repository = InMemoryExecutionHistoryRepository()
        val service = RoutingPolicyService(
            historyRepository = repository,
            nowProvider = { Instant.parse("2026-05-07T12:00:00Z") },
            sleep = { _: Long -> },
        )

        val result = service.executeWithFallback(
            primaryProvider = "OLLAMA",
            fallbackProviders = listOf("LITELLM"),
            maxRetries = 0,
            backoffMs = 0,
        ) { provider ->
            if (provider == "OLLAMA") {
                ProviderExecutionResult(false, "offline")
            } else {
                ProviderExecutionResult(true)
            }
        }

        assertTrue(result.success)
        assertEquals("LITELLM", result.finalProvider)
        assertEquals(2, result.attempts.size)
        assertEquals(2, repository.listRecent().size)
    }

    @Test
    fun `retries are bounded by maxRetries`() {
        val repository = InMemoryExecutionHistoryRepository()
        val service = RoutingPolicyService(
            historyRepository = repository,
            nowProvider = { Instant.parse("2026-05-07T12:00:00Z") },
            sleep = { _: Long -> },
        )

        val result = service.executeWithFallback(
            primaryProvider = "AITAO",
            fallbackProviders = emptyList<String>(),
            maxRetries = 2,
            backoffMs = 0,
        ) {
            ProviderExecutionResult(false, "timeout")
        }

        assertFalse(result.success)
        assertEquals(3, result.attempts.size)
        assertEquals(listOf(0, 1, 2), result.attempts.map { it.retryIndex })
    }

    @Test
    fun `removes duplicate primary from fallback and applies linear backoff`() {
        val repository = InMemoryExecutionHistoryRepository()
        val delays = mutableListOf<Long>()
        val service = RoutingPolicyService(
            historyRepository = repository,
            nowProvider = { Instant.parse("2026-05-07T12:00:00Z") },
            sleep = { delay -> delays.add(delay) },
        )

        val result = service.executeWithFallback(
            primaryProvider = "OLLAMA",
            fallbackProviders = listOf("OLLAMA", "LITELLM"),
            maxRetries = 1,
            backoffMs = 50,
        ) { provider ->
            if (provider == "LITELLM") {
                ProviderExecutionResult(true)
            } else {
                ProviderExecutionResult(false, "temporary")
            }
        }

        assertTrue(result.success)
        assertEquals(listOf("OLLAMA", "OLLAMA", "LITELLM"), result.attempts.map { it.providerId })
        assertEquals(listOf(50L), delays)
    }
}