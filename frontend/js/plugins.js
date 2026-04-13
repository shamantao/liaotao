/*
  plugins.js -- Frontend plugin hook architecture (PLUG-01).
  Responsibilities: register plugins, manage hook handlers, and run hook pipelines
  in a safe way (errors isolated per plugin), and load plugin files from plugins/.
*/

import { bridge } from "./bridge.js";

const SUPPORTED_HOOKS = [
  "beforeSend",
  "afterReceive",
  "onFileUpload",
  "renderTool",
  "onSaveConv",
];

const hooks = new Map(SUPPORTED_HOOKS.map((name) => [name, []]));
const pluginRegistry = new Map();
const PLUGIN_STATE_KEY = "liaotao.plugins.state.v1";

function loadPluginState() {
  try {
    const raw = localStorage.getItem(PLUGIN_STATE_KEY);
    if (!raw) return {};
    const parsed = JSON.parse(raw);
    return parsed && typeof parsed === "object" ? parsed : {};
  } catch {
    return {};
  }
}

function savePluginState() {
  const state = {};
  for (const p of pluginRegistry.values()) {
    state[p.id] = { enabled: p.enabled !== false };
  }
  localStorage.setItem(PLUGIN_STATE_KEY, JSON.stringify(state));
}

function isSupportedHook(hookName) {
  return hooks.has(hookName);
}

function normalizePluginID(plugin) {
  const raw = plugin && typeof plugin.id === "string" ? plugin.id.trim() : "";
  return raw || `plugin-${Date.now()}`;
}

export function listSupportedHooks() {
  return [...SUPPORTED_HOOKS];
}

export function listPlugins() {
  return [...pluginRegistry.values()].map((p) => ({
    id: p.id,
    name: p.name || p.id,
    description: p.description || "",
    source: p.source || "runtime",
    enabled: p.enabled !== false,
    hooks: Object.keys(p.hooks || {}).filter((h) => isSupportedHook(h)),
  }));
}

export function registerPlugin(plugin) {
  if (!plugin || typeof plugin !== "object") {
    throw new Error("registerPlugin: plugin object required");
  }
  if (!plugin.hooks || typeof plugin.hooks !== "object") {
    throw new Error("registerPlugin: plugin.hooks object required");
  }

  const pluginID = normalizePluginID(plugin);
  unregisterPlugin(pluginID);

  const normalized = {
    id: pluginID,
    name: typeof plugin.name === "string" ? plugin.name : pluginID,
    description: typeof plugin.description === "string" ? plugin.description : "",
    source: typeof plugin.source === "string" ? plugin.source : "runtime",
    enabled: plugin.enabled !== false,
    hooks: {},
  };

  const persisted = loadPluginState();
  if (persisted[pluginID] && typeof persisted[pluginID].enabled === "boolean") {
    normalized.enabled = persisted[pluginID].enabled;
  }

  for (const [hookName, handler] of Object.entries(plugin.hooks)) {
    if (!isSupportedHook(hookName)) continue;
    if (typeof handler !== "function") continue;

    normalized.hooks[hookName] = handler;
    hooks.get(hookName).push({ pluginID, handler });
  }

  pluginRegistry.set(pluginID, normalized);
  savePluginState();
  return pluginID;
}

export function unregisterPlugin(pluginID) {
  if (!pluginID || !pluginRegistry.has(pluginID)) return false;
  for (const hookName of SUPPORTED_HOOKS) {
    const entries = hooks.get(hookName);
    hooks.set(hookName, entries.filter((entry) => entry.pluginID !== pluginID));
  }
  pluginRegistry.delete(pluginID);
  savePluginState();
  return true;
}

export function setPluginEnabled(pluginID, enabled) {
  const plugin = pluginRegistry.get(pluginID);
  if (!plugin) return false;
  plugin.enabled = Boolean(enabled);
  savePluginState();
  return true;
}

export async function loadPluginsFromDirectory() {
  let records = [];
  try {
    records = await bridge.callService("ListPluginScripts");
  } catch (err) {
    console.warn("[liaotao][plugins] cannot list plugin scripts", err);
    return [];
  }

  if (!Array.isArray(records)) return [];

  const loaded = [];
  for (const record of records) {
    if (!record || typeof record.content !== "string") continue;
    const sourceLabel = typeof record.name === "string" ? record.name : "plugin.js";

    // Append sourceURL for easier debugging in devtools.
    const body = `${record.content}\n//# sourceURL=plugins/${sourceLabel}`;
    const url = URL.createObjectURL(new Blob([body], { type: "text/javascript" }));
    try {
      const mod = await import(url);
      if (mod && mod.default && typeof mod.default === "object") {
        registerPlugin({ ...mod.default, source: sourceLabel });
        loaded.push(sourceLabel);
      }
    } catch (err) {
      console.warn(`[liaotao][plugins] failed to load ${sourceLabel}`, err);
    } finally {
      URL.revokeObjectURL(url);
    }
  }
  return loaded;
}

export async function runHookPipeline(hookName, payload) {
  if (!isSupportedHook(hookName)) return payload;

  let current = payload;
  for (const entry of hooks.get(hookName)) {
    const plugin = pluginRegistry.get(entry.pluginID);
    if (!plugin || plugin.enabled === false) continue;
    try {
      const next = await entry.handler(current);
      if (next !== undefined) current = next;
    } catch (err) {
      console.warn(`[liaotao][plugin:${entry.pluginID}] hook ${hookName} failed`, err);
    }
  }
  return current;
}

export async function emitHook(hookName, payload) {
  if (!isSupportedHook(hookName)) return;
  for (const entry of hooks.get(hookName)) {
    const plugin = pluginRegistry.get(entry.pluginID);
    if (!plugin || plugin.enabled === false) continue;
    try {
      await entry.handler(payload);
    } catch (err) {
      console.warn(`[liaotao][plugin:${entry.pluginID}] hook ${hookName} failed`, err);
    }
  }
}

export function initializePluginSystem() {
  window.liaotaoPlugins = {
    register: registerPlugin,
    unregister: unregisterPlugin,
    setEnabled: setPluginEnabled,
    list: listPlugins,
    hooks: listSupportedHooks,
    emit: emitHook,
    run: runHookPipeline,
  };
}
