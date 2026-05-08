/*
 * DesktopPathManager.kt - desktop path manager implementation.
 * Responsibilities: provide resolved app data, db, and export directories
 * without spreading hardcoded paths in feature code.
 */

package io.liaotao.appdesktop.settings

import io.liaotao.shared.paths.PathManager
import java.nio.file.Files
import java.nio.file.Path
import java.nio.file.Paths

internal object DesktopPathManager : PathManager {
    private const val APP_DATA_OVERRIDE = "liaotao.app.data.dir"
    private const val APP_DIR_NAME = ".liaotao"
    private const val EXPORT_DIR_NAME = "exports"

    override fun appDataDirectory(): Path {
        val override = System.getProperty(APP_DATA_OVERRIDE)?.takeIf { it.isNotBlank() }
        val dir = if (override != null) {
            Paths.get(override)
        } else {
            val home = System.getProperty("user.home", ".")
            Paths.get(home).resolve(APP_DIR_NAME)
        }
        Files.createDirectories(dir)
        return dir
    }

    override fun appDatabaseFile(fileName: String): Path {
        return appDataDirectory().resolve(fileName)
    }

    override fun exportDirectory(): Path {
        val dir = appDataDirectory().resolve(EXPORT_DIR_NAME)
        Files.createDirectories(dir)
        return dir
    }
}
