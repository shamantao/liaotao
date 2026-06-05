package io.liaotao.appcli

object Version {
    private const val UNKNOWN = "dev"

    val current: String by lazy {
        try {
            javaClass.`package`.implementationVersion
                ?: readFromProperties()
        } catch (_: Exception) {
            UNKNOWN
        }
    }

    private fun readFromProperties(): String {
        val stream = Version::class.java.classLoader.getResourceAsStream("version.properties")
            ?: return UNKNOWN
        return stream.use { input ->
            val props = java.util.Properties()
            props.load(input)
            props.getProperty("version", UNKNOWN)
        }
    }
}
