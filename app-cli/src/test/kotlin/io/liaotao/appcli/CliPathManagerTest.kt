package io.liaotao.appcli

import java.nio.file.Files
import kotlin.test.Test
import kotlin.test.assertEquals
import kotlin.test.assertNotNull
import kotlin.test.assertTrue

class CliPathManagerTest {
    @Test
    fun `uses override directory and creates paths`() {
        val temp = Files.createTempDirectory("liaotao-cli-path-")
        val previous = System.getProperty("liaotao.app.data.dir")
        try {
            System.setProperty("liaotao.app.data.dir", temp.toString())

            val dataDir = CliPathManager.appDataDirectory()
            val dbPath = CliPathManager.appDatabaseFile("liaotao.db")
            val exportDir = CliPathManager.exportDirectory()

            assertEquals(temp.toAbsolutePath().normalize(), dataDir.toAbsolutePath().normalize())
            assertTrue(dbPath.startsWith(dataDir))
            assertTrue(dbPath.toString().endsWith("liaotao.db"))
            assertTrue(Files.exists(exportDir))
        } finally {
            if (previous == null) {
                System.clearProperty("liaotao.app.data.dir")
            } else {
                System.setProperty("liaotao.app.data.dir", previous)
            }
        }
    }

    @Test
    fun `default data dir is under user home`() {
        val home = System.getProperty("user.home")
        System.clearProperty("liaotao.app.data.dir")

        val dataDir = CliPathManager.appDataDirectory()

        assertTrue(dataDir.toString().startsWith(home))
        assertTrue(dataDir.toString().contains(".liaotao"))
        assertTrue(Files.exists(dataDir))
    }

    @Test
    fun `database file resolves to db subdirectory`() {
        val temp = Files.createTempDirectory("liaotao-cli-db-")
        val previous = System.getProperty("liaotao.app.data.dir")
        try {
            System.setProperty("liaotao.app.data.dir", temp.toString())

            val dbPath = CliPathManager.appDatabaseFile("test.db")

            assertTrue(dbPath.toString().contains("/db/"))
            assertEquals("test.db", dbPath.fileName.toString())
        } finally {
            if (previous == null) {
                System.clearProperty("liaotao.app.data.dir")
            } else {
                System.setProperty("liaotao.app.data.dir", previous)
            }
        }
    }
}
