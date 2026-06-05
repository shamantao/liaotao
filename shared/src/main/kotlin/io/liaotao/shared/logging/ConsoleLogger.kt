package io.liaotao.shared.logging

import java.io.PrintWriter
import java.io.StringWriter
import java.time.LocalDateTime
import java.time.format.DateTimeFormatter

class ConsoleLogger(
    private val name: String,
    private val minLevel: LogLevel = LogLevel.INFO,
) : Logger {

    private val timestampFormat = DateTimeFormatter.ofPattern("HH:mm:ss.SSS")

    override fun trace(message: String) = log(LogLevel.TRACE, message)
    override fun debug(message: String) = log(LogLevel.DEBUG, message)
    override fun info(message: String) = log(LogLevel.INFO, message)
    override fun warn(message: String) = log(LogLevel.WARN, message)
    override fun error(message: String) = log(LogLevel.ERROR, message)

    override fun error(message: String, throwable: Throwable) {
        if (minLevel <= LogLevel.ERROR) {
            val sw = StringWriter()
            throwable.printStackTrace(PrintWriter(sw))
            log(LogLevel.ERROR, "$message\n${sw}")
        }
    }

    override fun log(level: LogLevel, message: String) {
        if (level < minLevel) return
        val timestamp = LocalDateTime.now().format(timestampFormat)
        val tag = level.name.padEnd(5)
        println("$timestamp [$tag] [$name] $message")
    }
}
