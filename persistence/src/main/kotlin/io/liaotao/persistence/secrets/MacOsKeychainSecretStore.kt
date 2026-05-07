/*
 * MacOsKeychainSecretStore.kt - macOS keychain secret store.
 * Responsibilities: persist and retrieve secrets through macOS Keychain using
 * the `security` CLI to avoid plain-text storage in local SQLite tables.
 */

package io.liaotao.persistence.secrets

class MacOsKeychainSecretStore(
    private val serviceName: String = "io.liaotao.desktop",
) : SecretStore {
    override fun putSecret(ref: String, value: String): Boolean {
        val result = runCommand(
            "security",
            "add-generic-password",
            "-a",
            ref,
            "-s",
            serviceName,
            "-w",
            value,
            "-U",
        )
        return result.exitCode == 0
    }

    override fun getSecret(ref: String): String? {
        val result = runCommand(
            "security",
            "find-generic-password",
            "-a",
            ref,
            "-s",
            serviceName,
            "-w",
        )
        if (result.exitCode != 0) {
            return null
        }
        return result.stdout.trim().takeIf { it.isNotEmpty() }
    }

    override fun deleteSecret(ref: String): Boolean {
        val result = runCommand(
            "security",
            "delete-generic-password",
            "-a",
            ref,
            "-s",
            serviceName,
        )
        return result.exitCode == 0
    }

    private fun runCommand(vararg args: String): CommandResult {
        return try {
            val process = ProcessBuilder(*args)
                .redirectErrorStream(true)
                .start()
            val output = process.inputStream.bufferedReader().use { it.readText() }
            val code = process.waitFor()
            CommandResult(exitCode = code, stdout = output)
        } catch (exception: Exception) {
            CommandResult(exitCode = -1, stdout = exception.message ?: "")
        }
    }
}

private data class CommandResult(
    val exitCode: Int,
    val stdout: String,
)