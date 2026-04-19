/**
 * bridge.js -- Typed Wails v3 runtime bridge for Svelte frontend.
 * Responsibilities: wrap window.wails.Call.ByName with typed async functions,
 * expose event listener/emitter helpers compatible with Svelte stores.
 * Replicates the contract of frontend/js/bridge.js with explicit method signatures.
 */

// ── Constants ──────────────────────────────────────────────────────────────

const SERVICE_FQN = "liaotao/internal/bindings.Service.";

// ── Runtime readiness ──────────────────────────────────────────────────────

async function waitForWailsRuntime(timeoutMs = 4000) {
  const start = Date.now();
  while (Date.now() - start < timeoutMs) {
    if (window.wails?.Call?.ByName) return;
    await new Promise((r) => setTimeout(r, 60));
  }
  console.error("[liaotao] waitForWailsRuntime: timeout");
}

// ── Low-level call ─────────────────────────────────────────────────────────

async function callService(method, payload) {
  await waitForWailsRuntime();
  if (!window.wails?.Call?.ByName) {
    throw new Error("no-wails-runtime");
  }
  const fqn = SERVICE_FQN + method;
  return payload !== undefined && payload !== null
    ? window.wails.Call.ByName(fqn, payload)
    : window.wails.Call.ByName(fqn);
}

// ── Events ─────────────────────────────────────────────────────────────────

/**
 * Subscribe to a Wails event. Returns an unsubscribe function.
 * Falls back to DOM CustomEvent when Wails runtime is absent.
 */
export function eventsOn(name, cb) {
  if (window.wails?.Events?.On) {
    window.wails.Events.On(name, (e) =>
      cb(e && e.data !== undefined ? e.data : e),
    );
    // Wails v3 alpha does not return an unsub function — use Off.
    return () => window.wails?.Events?.Off?.(name);
  }
  const handler = (e) => cb(e.detail);
  document.addEventListener(`liaotao:${name}`, handler);
  return () => document.removeEventListener(`liaotao:${name}`, handler);
}

/**
 * Emit a Wails event (frontend → Go or frontend → frontend).
 */
export function eventsEmit(name, payload) {
  if (window.wails?.Events?.Emit) {
    window.wails.Events.Emit(name, payload);
    return;
  }
  document.dispatchEvent(
    new CustomEvent(`liaotao:${name}`, { detail: payload }),
  );
}

// ── Service bindings (typed wrappers) ──────────────────────────────────────

// -- Core / Health --

export const health = () => callService("Health");
export const getAboutInfo = () => callService("GetAboutInfo");

// -- Conversations --

export const createConversation = (payload) =>
  callService("CreateConversation", payload);

export const listConversations = (payload) =>
  callService("ListConversations", payload);

export const searchConversations = (payload) =>
  callService("SearchConversations", payload);

export const renameConversation = (payload) =>
  callService("RenameConversation", payload);

export const assignConversationGroup = (payload) =>
  callService("AssignConversationGroup", payload);

export const updateConversationSettings = (payload) =>
  callService("UpdateConversationSettings", payload);

export const deleteConversation = (id) =>
  callService("DeleteConversation", id);

// -- Messages --

export const listMessages = (payload) =>
  callService("ListMessages", payload);

export const saveMessage = (payload) =>
  callService("SaveMessage", payload);

export const deleteMessage = (payload) =>
  callService("DeleteMessage", payload);

// -- Chat / Streaming --

export const sendMessage = (payload) =>
  callService("SendMessage", payload);

export const cancelGeneration = (payload) =>
  callService("CancelGeneration", payload);

export const listModels = (payload) =>
  callService("ListModels", payload);

// -- Providers --

export const listProviders = (payload) =>
  callService("ListProviders", payload);

export const createProvider = (payload) =>
  callService("CreateProvider", payload);

export const updateProvider = (payload) =>
  callService("UpdateProvider", payload);

export const deleteProvider = (payload) =>
  callService("DeleteProvider", payload);

export const testConnection = (payload) =>
  callService("TestConnection", payload);

export const listProviderProfiles = () =>
  callService("ListProviderProfiles");

// -- Projects --

export const listProjects = (payload) =>
  callService("ListProjects", payload);

export const createProject = (payload) =>
  callService("CreateProject", payload);

export const renameProject = (payload) =>
  callService("RenameProject", payload);

export const archiveProject = (payload) =>
  callService("ArchiveProject", payload);

export const setProjectRetrievalBackend = (payload) =>
  callService("SetProjectRetrievalBackend", payload);

export const getProjectDashboard = (payload) =>
  callService("GetProjectDashboard", payload);

// -- Tags --

export const createTag = (payload) =>
  callService("CreateTag", payload);

export const listTags = () => callService("ListTags");

export const updateTag = (payload) =>
  callService("UpdateTag", payload);

export const deleteTag = (id) => callService("DeleteTag", id);

export const addTagToConversation = (payload) =>
  callService("AddTagToConversation", payload);

export const removeTagFromConversation = (payload) =>
  callService("RemoveTagFromConversation", payload);

export const listConversationsByTag = (payload) =>
  callService("ListConversationsByTag", payload);

// -- Attachments --

export const uploadAttachment = (payload) =>
  callService("UploadAttachment", payload);

export const listAttachments = (payload) =>
  callService("ListAttachments", payload);

export const setAttachmentProjectScope = (payload) =>
  callService("SetAttachmentProjectScope", payload);

// -- Quotas / Router --

export const getQuotaStatus = () => callService("GetQuotaStatus");

export const setProviderQuota = (payload) =>
  callService("SetProviderQuota", payload);

export const reorderProviders = (payload) =>
  callService("ReorderProviders", payload);

// -- Settings --

export const getGeneralSettings = () =>
  callService("GetGeneralSettings");

export const updateGeneralSettings = (payload) =>
  callService("UpdateGeneralSettings", payload);

export const exportConfiguration = () =>
  callService("ExportConfiguration");

export const exportConfigurationToFile = () =>
  callService("ExportConfigurationToFile");

export const importConfiguration = (payload) =>
  callService("ImportConfiguration", payload);

// -- MCP Servers --

export const listMCPServers = () => callService("ListMCPServers");

export const saveMCPServer = (payload) =>
  callService("SaveMCPServer", payload);

export const deleteMCPServer = (id) =>
  callService("DeleteMCPServer", id);

export const toggleMCPServer = (payload) =>
  callService("ToggleMCPServer", payload);

export const pingMCPServer = (id) =>
  callService("PingMCPServer", id);

export const allAvailableTools = () =>
  callService("AllAvailableTools");

export const dispatchToolCalls = (payload) =>
  callService("DispatchToolCalls", payload);

// -- Plugins --

export const listPluginScripts = () =>
  callService("ListPluginScripts");

// -- Export --

export const exportConversation = (payload) =>
  callService("ExportConversation", payload);

export const exportProject = (payload) =>
  callService("ExportProject", payload);

// -- Updates --

export const checkForUpdate = () => callService("CheckForUpdate");

export const downloadAndInstallUpdate = () =>
  callService("DownloadAndInstallUpdate");
