/*
 * SecretStore.kt - secure secret storage contract.
 * Responsibilities: abstract secret persistence independently from settings
 * storage so sensitive values never live in plain local configuration tables.
 */

package io.liaotao.persistence.secrets

interface SecretStore {
    fun putSecret(ref: String, value: String): Boolean

    fun getSecret(ref: String): String?

    fun deleteSecret(ref: String): Boolean
}