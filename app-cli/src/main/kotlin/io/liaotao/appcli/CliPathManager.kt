package io.liaotao.appcli

import io.liaotao.shared.paths.PathManager
import java.nio.file.Files
import java.nio.file.Path
import java.nio.file.Paths

object CliPathManager : PathManager {
    private const val APP_DATA_PROPERTY = "liaotao.app.data.dir"
    private const val APP_DIR_NAME = ".liaotao"
    private const val EXPORT_DIR_NAME = "exports"

    override fun appDataDirectory(): Path {
        val override = System.getProperty(APP_DATA_PROPERTY)?.takeIf { it.isNotBlank() }
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
        return appDataDirectory().resolve("db").resolve(fileName)
    }

    override fun exportDirectory(): Path {
        val dir = appDataDirectory().resolve(EXPORT_DIR_NAME)
        Files.createDirectories(dir)
        return dir
    }
}
