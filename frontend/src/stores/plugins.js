/**
 * plugins.js -- Plugin runtime store.
 * Responsibilities: load plugin scripts from backend, execute them,
 * manage lifecycle (onInit/onDestroy), expose topbar action indicators.
 * Plugins use `export default { ... }` format with hooks.
 */

import { writable, derived } from "svelte/store";
import * as bridge from "../lib/bridge.js";

// ── Stores ─────────────────────────────────────────────────────────────────

/** All loaded plugin instances. */
export const loadedPlugins = writable([]);

/** Topbar action indicators registered by plugins. */
export const topbarActions = writable([]);

// Internal registry for cleanup
const _intervals = [];

// ── Context factory ────────────────────────────────────────────────────────

function createPluginContext(pluginId) {
  return {
    bridge,
    pluginId,
    /**
     * Register a topbar status indicator.
     * @param {{ id: string, label: string, color: string, tooltip?: string }} action
     */
    registerTopbarAction(action) {
      topbarActions.update((list) => {
        const existing = list.findIndex((a) => a.id === action.id);
        if (existing >= 0) {
          const copy = [...list];
          copy[existing] = { ...action };
          return copy;
        }
        return [...list, { ...action }];
      });
    },
    /**
     * Update a previously registered topbar action.
     * @param {string} id
     * @param {Partial<{ color: string, tooltip: string, label: string }>} updates
     */
    updateTopbarAction(id, updates) {
      topbarActions.update((list) =>
        list.map((a) => (a.id === id ? { ...a, ...updates } : a)),
      );
    },
    /**
     * Register an interval that will be cleaned up on plugin unload.
     * @param {Function} fn
     * @param {number} ms
     * @returns {number} interval id
     */
    setInterval(fn, ms) {
      const id = setInterval(fn, ms);
      _intervals.push(id);
      return id;
    },
  };
}

// ── Plugin loader ──────────────────────────────────────────────────────────

/**
 * Load a single plugin from its source content string.
 * Returns the plugin default export or null on failure.
 */
async function loadPluginFromSource(name, content) {
  try {
    const blob = new Blob([content], { type: "text/javascript" });
    const url = URL.createObjectURL(blob);
    const mod = await import(/* @vite-ignore */ url);
    URL.revokeObjectURL(url);
    return mod.default || null;
  } catch (e) {
    console.warn(`[plugins] Failed to load ${name}:`, e);
    return null;
  }
}

/**
 * Initialize all plugins from backend.
 * Called once on app startup.
 */
export async function initPlugins() {
  let scripts;
  try {
    scripts = await bridge.listPluginScripts();
  } catch (e) {
    console.warn("[plugins] Cannot load plugin scripts:", e);
    return;
  }
  if (!scripts || scripts.length === 0) return;

  const instances = [];
  for (const script of scripts) {
    // Skip example files
    if (script.name.endsWith(".example.js")) continue;

    const plugin = await loadPluginFromSource(script.name, script.content);
    if (!plugin || !plugin.id) continue;
    if (plugin.enabled === false) continue;

    instances.push(plugin);

    // Run onInit hook
    if (typeof plugin.hooks?.onInit === "function") {
      try {
        const ctx = createPluginContext(plugin.id);
        await plugin.hooks.onInit(ctx);
      } catch (e) {
        console.warn(`[plugins] onInit failed for ${plugin.id}:`, e);
      }
    }
  }

  loadedPlugins.set(instances);
}

/**
 * Cleanup all plugin intervals and reset state.
 */
export function destroyPlugins() {
  for (const id of _intervals) clearInterval(id);
  _intervals.length = 0;
  topbarActions.set([]);
  loadedPlugins.set([]);
}
