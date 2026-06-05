package io.liaotao.appcli

import kotlin.test.Test
import kotlin.test.assertNotNull
import kotlin.test.assertTrue

class VersionTest {
    @Test
    fun `version is not unknown`() {
        assertTrue(Version.current.isNotBlank())
    }

    @Test
    fun `version has valid format`() {
        assertTrue(Version.current.matches(Regex("""^\d+\.\d+\.\d+.*$""")) || Version.current == "dev")
    }
}
