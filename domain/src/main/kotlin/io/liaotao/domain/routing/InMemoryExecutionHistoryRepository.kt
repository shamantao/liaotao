/*
 * InMemoryExecutionHistoryRepository.kt - in-memory execution history.
 * Responsibilities: provide local storage for execution attempts in tests and
 * early UI integration before mandatory database wiring.
 */

package io.liaotao.domain.routing

class InMemoryExecutionHistoryRepository : ExecutionHistoryRepository {
    private val attempts = mutableListOf<ExecutionAttempt>()

    override fun recordAttempt(attempt: ExecutionAttempt) {
        attempts.add(attempt)
    }

    override fun listRecent(limit: Int): List<ExecutionAttempt> {
        return attempts
            .asReversed()
            .take(limit)
    }
}