package io.liaotao.appcli.ui

import com.github.ajalt.mordant.rendering.TextColors
import com.github.ajalt.mordant.rendering.TextStyles
import com.github.ajalt.mordant.terminal.Terminal

fun Terminal.success(msg: String) {
    println(" ${TextColors.brightGreen("✓")} $msg")
}

fun Terminal.error(msg: String) {
    println(" ${TextColors.brightRed("✗")} $msg")
}

fun Terminal.info(msg: String) {
    println(" ${TextColors.brightBlue("ℹ")} $msg")
}

fun Terminal.warn(msg: String) {
    println(" ${TextColors.brightYellow("!")} $msg")
}

fun Terminal.header(title: String, subtitle: String) {
    println()
    println(TextColors.brightCyan(TextStyles.bold(" $title ")).toString() + TextColors.gray("│ $subtitle").toString())
    val width = size?.width?.coerceAtMost(60) ?: 60
    println(TextColors.gray("─".repeat(width)))
}

fun Terminal.renderTable(
    columns: List<String>,
    rows: List<List<Any>>,
) {
    val widths = columns.indices.map { colIdx ->
        val headerLen = columns[colIdx].length
        val maxDataLen = rows.maxOfOrNull { row ->
            row.getOrNull(colIdx)?.toString()?.length ?: 0
        } ?: 0
        (maxOf(headerLen, maxDataLen) + 2).coerceIn(4, 40)
    }

    fun pad(text: String, idx: Int) = " ${text.padEnd(widths[idx] - 2)} "

    val sep = "├${widths.joinToString("┼") { "─".repeat(widths[it]) }}┤"
    val top = "┌${widths.joinToString("┬") { "─".repeat(widths[it]) }}┐"
    val bot = "└${widths.joinToString("┴") { "─".repeat(widths[it]) }}┘"

    fun colored(s: String) = TextColors.gray(s)

    println(colored(top))
    println(colored("│") + columns.mapIndexed { i, c -> TextStyles.bold(pad(c, i)) }.joinToString(colored("│")) + colored("│"))
    println(colored(sep))
    rows.forEach { row ->
        println(colored("│") + row.mapIndexed { i, c -> pad(c.toString(), i) }.joinToString(colored("│")) + colored("│"))
    }
    println(colored(bot))
}

fun Terminal.renderMarkdown(text: String) {
    text.lines().forEach { line ->
        val t = line.trimStart()
        when {
            t.startsWith("```") -> println(TextColors.gray(line))
            t.startsWith("# ") -> println(TextStyles.bold(TextColors.brightCyan(t.removePrefix("# "))))
            t.startsWith("## ") -> println(TextStyles.bold(TextColors.brightCyan(t.removePrefix("## "))))
            t.startsWith("### ") -> println(TextStyles.bold(TextColors.cyan(t.removePrefix("### "))))
            t.startsWith("- ") || t.startsWith("* ") -> println("  ${TextColors.gray("•")} ${t.removePrefix("- ").removePrefix("* ")}")
            t.startsWith("> ") -> println(TextColors.gray(" │ ${t.removePrefix("> ")}"))
            line.isBlank() -> println()
            else -> println(line)
        }
    }
}
