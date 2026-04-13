/*
  plugins.js -- Frontend plugin hook architecture (PLUG-01).
  Responsibilities: register plugins, manage hook handlers, and run hook pipelines
  in a safe way (errors isolated per plugin).
*/

const SUPPORTED_HOOKS = [
  "beforeSend",
  "afterReceive",
  "onFileUpload",
  "renderTool",
  "onSaveConv",
];

const hooks = new Map(SUPPORTED_HOOKS.map((name) => [name, []]));
const pluginRegistry = new Map();

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
    enabled: plugin.enabled !== false,
    hooks: {},
  };

  for (const [hookName, handler] of Object.entries(plugin.hooks)) {
    if (!isSupportedHook(hookName)) continue;
    if (typeof handler !== "function") continue;

    normalized.hooks[hookName] = handler;
    hooks.get(hookName).push({ pluginID, handler });
  }

  pluginRegistry.set(pluginID, normalized);
  return pluginID;
}

export function unregisterPlugin(pluginID) {
  if (!pluginID || !pluginRegistry.has(pluginID)) return false;
  for (const hookName of SUPPORTED_HOOKS) {
    const entries = hooks.get(hookName);
    hooks.set(hookName, entries.filter((entry) => entry.pluginID !== pluginID));
  }
  pluginRegistry.delete(pluginID);
  return true;
}

export function setPluginEnabled(pluginID, enabled) {
  const plugin = pluginRegistry.get(pluginID);
  if (!plugin) return false;
  plugin.enabled = Boolean(enabled);
  return true;
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
  };
}
