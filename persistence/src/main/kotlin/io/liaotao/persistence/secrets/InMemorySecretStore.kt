/*
 * InMemorySecretStore.kt - in-memory secret store implementation.
 * Responsibilities: provide a deterministic fallback for local development
 * and tests when OS keychain integration is unavailable.
 */

package io.liaotao.persistence.secrets

class InMemorySecretStore : SecretStore {
    private val values = mutableMapOf<String, String>()

    override fun putSecret(ref: String, value: String): Boolean {
        values[ref] = value
        return true
    }

    override fun getSecret(ref: String): String? = values[ref]

    override fun deleteSecret(ref: String): Boolean = values.remove(ref) != null
}