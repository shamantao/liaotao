package io.liaotao.shared.logging

enum class LogLevel {
    TRACE, DEBUG, INFO, WARN, ERROR
}

interface Logger {
    fun trace(message: String)
    fun debug(message: String)
    fun info(message: String)
    fun warn(message: String)
    fun error(message: String)
    fun error(message: String, throwable: Throwable)

    fun log(level: LogLevel, message: String) {
        when (level) {
            LogLevel.TRACE -> trace(message)
            LogLevel.DEBUG -> debug(message)
            LogLevel.INFO -> info(message)
            LogLevel.WARN -> warn(message)
            LogLevel.ERROR -> error(message)
        }
    }
}

fun logger(name: String): Logger = ConsoleLogger(name)
fun logger(clazz: Class<*>): Logger = ConsoleLogger(clazz.simpleName)
