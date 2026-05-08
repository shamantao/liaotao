/*
 * FeatureFlags.kt - shared feature flag contract.
 * Responsibilities: define toggles used for gradual rollout and safe fallback.
 */

package io.liaotao.shared.feature

interface FeatureFlags {
    val usePersistedProviderSelector: Boolean
}
