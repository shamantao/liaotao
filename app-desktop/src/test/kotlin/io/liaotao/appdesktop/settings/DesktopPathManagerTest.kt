/*
 * DesktopPathManagerTest.kt - tests for desktop path manager behavior.
 * Responsibilities: verify path override and directory creation semantics.
 */

package io.liaotao.appdesktop.settings

import java.nio.file.Files
import kotlin.test.Test
import kotlin.test.assertEquals
import kotlin.test.assertTrue

class DesktopPathManagerTest {
    @Test
    fun `path manager uses override directory and creates paths`() {
        val temp = Files.createTempDirectory("liaotao-path-manager-")
        val previous = System.getProperty("liaotao.app.data.dir")
        try {
            System.setProperty("liaotao.app.data.dir", temp.toString())

            val dataDir = DesktopPathManager.appDataDirectory()
            val dbPath = DesktopPathManager.appDatabaseFile("liaotao.db")
            val exportDir = DesktopPathManager.exportDirectory()

            assertEquals(temp.toAbsolutePath().normalize(), dataDir.toAbsolutePath().normalize())
            assertEquals(dataDir.resolve("liaotao.db"), dbPath)
            assertTrue(Files.exists(exportDir))
        } finally {
            if (previous == null) {
                System.clearProperty("liaotao.app.data.dir")
            } else {
                System.setProperty("liaotao.app.data.dir", previous)
            }
        }
    }
}
