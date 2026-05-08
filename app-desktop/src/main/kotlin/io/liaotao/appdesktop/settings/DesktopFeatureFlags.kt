/*
 * DesktopFeatureFlags.kt - desktop feature flags provider.
 * Responsibilities: read rollout toggles from system properties.
 */

package io.liaotao.appdesktop.settings

import io.liaotao.shared.feature.FeatureFlags

internal object DesktopFeatureFlags : FeatureFlags {
    private const val PROVIDER_SELECTOR_FLAG = "liaotao.feature.persistedProviderSelector"

    override val usePersistedProviderSelector: Boolean
        get() = System.getProperty(PROVIDER_SELECTOR_FLAG, "true").toBoolean()
}
