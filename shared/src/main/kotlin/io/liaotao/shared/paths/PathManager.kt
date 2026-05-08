/*
 * PathManager.kt - shared path resolution contract.
 * Responsibilities: centralize app path derivation to avoid hardcoded path
 * usage scattered across modules.
 */

package io.liaotao.shared.paths

import java.nio.file.Path

interface PathManager {
    fun appDataDirectory(): Path
    fun appDatabaseFile(fileName: String): Path
    fun exportDirectory(): Path
}
