package io.liaotao.appcli

import io.liaotao.shared.logging.ConsoleLogger
import io.liaotao.shared.logging.LogLevel
import java.io.ByteArrayOutputStream
import java.io.PrintStream
import kotlin.test.Test
import kotlin.test.assertContains
import kotlin.test.assertTrue

class ConsoleLoggerTest {
    @Test
    fun `logs messages at configured level`() {
        val output = ByteArrayOutputStream()
        val printStream = PrintStream(output)
        val original = System.out
        try {
            System.setOut(printStream)
            val log = ConsoleLogger("test-logger", LogLevel.INFO)
            log.info("hello world")
            log.debug("should not appear")
        } finally {
            System.setOut(original)
        }

        val text = output.toString()
        assertContains(text, "hello world")
        assertTrue(!text.contains("should not appear"))
    }

    @Test
    fun `trace messages are suppressed at info level`() {
        val output = ByteArrayOutputStream()
        val printStream = PrintStream(output)
        val original = System.out
        try {
            System.setOut(printStream)
            val log = ConsoleLogger("quiet", LogLevel.WARN)
            log.trace("silent")
            log.info("silent too")
            log.warn("visible")
        } finally {
            System.setOut(original)
        }

        val text = output.toString()
        assertContains(text, "visible")
        assertTrue(!text.contains("silent"))
    }
}
