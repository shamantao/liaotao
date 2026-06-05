/*
 * DesktopUserPreferences.kt - local desktop preferences persistence.
 * Responsibilities: store and load user-level preferences such as selected
 * application language.
 */

package io.liaotao.appdesktop.settings

import io.liaotao.appdesktop.i18n.AppLanguage
import java.nio.file.Files
import java.util.Properties

internal object DesktopUserPreferences {
    private const val FILE_NAME = "user-preferences.properties"
    private const val KEY_LANGUAGE = "language"

    fun loadLanguage(): AppLanguage {
        val file = DesktopPathManager.appDataDirectory().resolve(FILE_NAME)
        if (!Files.exists(file)) {
            return AppLanguage.ENGLISH
        }
        return runCatching {
            val props = Properties()
            Files.newInputStream(file).use { props.load(it) }
            val raw = props.getProperty(KEY_LANGUAGE)?.trim().orEmpty()
            AppLanguage.valueOf(raw)
        }.getOrDefault(AppLanguage.ENGLISH)
    }

    fun saveLanguage(language: AppLanguage) {
        val file = DesktopPathManager.appDataDirectory().resolve(FILE_NAME)
        val props = Properties()
        props.setProperty(KEY_LANGUAGE, language.name)
        Files.newOutputStream(file).use { props.store(it, "Liaotao desktop user preferences") }
    }
}
