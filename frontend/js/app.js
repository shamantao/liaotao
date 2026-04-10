/*
  app.js -- Liaotao MVP entry point (ES module).
  Responsibilities: tab switching, sidebar collapse/resize, settings navigation,
  event bindings, and app initialization. All domain logic is in sub-modules.
*/

import { appState, els, loadSettingsFromStorage, persistSettingsToStorage, applySettingsToUI } from "./state.js";
import { bridge } from "./bridge.js";
import {
  loadProviders, loadProviderProfiles, loadModels,
  updateChatProviderSelector,
  renderLastUsedModels,
  rememberLastUsedModel,
  saveProvider, deleteCurrentProvider, showNewProviderForm,
  testProviderConnection, applyPreset, updateProviderURLPlaceholder, syncChatModelSelector,
} from "./providers.js";
import {
  renderMessages, sendPrompt, cancelGeneration,
  appendAssistantChunk, stopStreaming, activeConversation,
  copyMessage, editMessage, regenerateMessage, deleteMessage,
  attachResponseMeta, appendToolCall, updateToolResult,
} from "./chat.js";
import { newConversation, loadPersistedConversations, saveActiveConversationSettings, searchConversations } from "./conversations.js";
import { loadMCPServers, initMCPFormListeners } from "./mcp.js";

// ── Settings navigation ────────────────────────────────────────────────────
function switchSettingsSection(sectionId) {
  appState.settingsSection = sectionId;
  els.settingsNavBtns.forEach((btn) =>
    btn.classList.toggle("active", btn.dataset.section === sectionId));
  els.settingsSections.forEach((sec) =>
    sec.classList.toggle("active", sec.id === `section-${sectionId}`));
  if (sectionId === "providers") {
    loadProviders();
  }
  if (sectionId === "mcp") {
    loadMCPServers();
  }
}

// ── Tab switching ──────────────────────────────────────────────────────────
function switchTab(tab) {
  els.tabs.forEach((btn) => btn.classList.toggle("active", btn.dataset.tab === tab));
  els.panels.forEach((panel) => panel.classList.toggle("active", panel.id === tab));
}

// ── Sidebar collapse / resize ──────────────────────────────────────────────
function applySidebarState() {
  const w = appState.sidebarCollapsed ? 72 : appState.expandedSidebarWidth;
  els.appShell.style.setProperty("--sidebar-width", `${w}px`);
  els.appShell.classList.toggle("sidebar-collapsed", appState.sidebarCollapsed);
}

function toggleSidebar() {
  if (!appState.sidebarCollapsed) {
    appState.expandedSidebarWidth = Math.max(180, appState.sidebarWidth);
    appState.sidebarCollapsed = true;
  } else {
    appState.sidebarCollapsed = false;
    appState.sidebarWidth = appState.expandedSidebarWidth;
  }
  applySidebarState();
}

function initSidebarResizer() {
  let drag = false;
  els.sidebarResizer.addEventListener("mousedown", () => {
    drag = true;
    if (appState.sidebarCollapsed) {
      appState.sidebarCollapsed = false;
      appState.sidebarWidth = appState.expandedSidebarWidth;
    }
    applySidebarState();
  });
  window.addEventListener("mousemove", (e) => {
    if (!drag) return;
    const next = Math.max(72, Math.min(460, e.clientX));
    appState.sidebarWidth = next;
    appState.expandedSidebarWidth = Math.max(180, next);
    applySidebarState();
  });
  window.addEventListener("mouseup", () => { drag = false; });
}

// ── Event bindings ─────────────────────────────────────────────────────────
function bindEvents() {
  els.tabs.forEach((btn) => btn.addEventListener("click", () => switchTab(btn.dataset.tab)));

  els.newChat.addEventListener("click", newConversation);
  els.send.addEventListener("click", sendPrompt);
  els.stop.addEventListener("click", cancelGeneration);
  els.sidebarToggle.addEventListener("click", toggleSidebar);
  els.refreshModels.addEventListener("click", () => loadModels({ force: true }));

  els.chatProvider.addEventListener("change", async () => {
    // Parse the selected value. "0" is the Automat (router) option — must stay 0, not coerced to null.
    const raw = els.chatProvider.value;
    const newId = raw === "" ? null : Number(raw);
    appState.activeProviderId = newId;
    const conv = appState.conversations.find((c) => c.id === appState.activeConversationId);
    if (conv) {
      conv.providerId = newId || 0;
      conv.providerName = newId > 0
        ? (appState.providers.find((p) => p.id === newId)?.name || "")
        : "";
      conv.model = "";
      await saveActiveConversationSettings();
    }
    persistSettingsToStorage();
    syncChatModelSelector("");
  });

  const ensureModelsLoaded = () => loadModels({ preferredModel: els.chatModel.value || "" });
  els.chatModel.addEventListener("focus", ensureModelsLoaded);
  els.chatModel.addEventListener("pointerdown", ensureModelsLoaded);

  els.chatModel.addEventListener("change", () => {
    const conv = appState.conversations.find((c) => c.id === appState.activeConversationId);
    if (!conv) return;
    if (appState.activeProviderId === 0 && typeof els.chatModel.value === "string" && els.chatModel.value.includes("::")) {
      const [providerIDRaw, model] = els.chatModel.value.split("::");
      const providerID = Number(providerIDRaw) || 0;
      if (providerID > 0) {
        appState.activeProviderId = providerID;
        els.chatProvider.value = String(providerID);
        conv.providerId = providerID;
        conv.providerName = appState.providers.find((p) => p.id === providerID)?.name || "";
      }
      conv.model = model || "";
    } else {
      conv.model = els.chatModel.value;
    }
    rememberLastUsedModel(conv.providerId, conv.providerName, conv.model);
    saveActiveConversationSettings();
  });

  if (els.chatModelFilter) {
    els.chatModelFilter.addEventListener("input", () => {
      appState.modelFilterQuery = els.chatModelFilter.value || "";
      const conv = appState.conversations.find((c) => c.id === appState.activeConversationId);
      syncChatModelSelector(conv ? conv.model : "");
    });
  }

  if (els.chatTemperature) {
    els.chatTemperature.addEventListener("change", () => {
      const conv = appState.conversations.find((c) => c.id === appState.activeConversationId);
      if (!conv) return;
      conv.temperature = Number(els.chatTemperature.value) > 0 ? Number(els.chatTemperature.value) : 0.7;
      saveActiveConversationSettings();
    });
  }

  if (els.chatMaxTokens) {
    els.chatMaxTokens.addEventListener("change", () => {
      const conv = appState.conversations.find((c) => c.id === appState.activeConversationId);
      if (!conv) return;
      conv.maxTokens = Math.max(0, Number(els.chatMaxTokens.value) || 0);
      saveActiveConversationSettings();
    });
  }

  if (els.chatSystemPrompt) {
    let systemPromptDebounce = null;
    els.chatSystemPrompt.addEventListener("input", () => {
      const conv = appState.conversations.find((c) => c.id === appState.activeConversationId);
      if (!conv) return;
      conv.systemPrompt = els.chatSystemPrompt.value || "";
      if (systemPromptDebounce) clearTimeout(systemPromptDebounce);
      systemPromptDebounce = setTimeout(() => {
        saveActiveConversationSettings();
      }, 240);
    });
  }

  els.prompt.addEventListener("keydown", (e) => {
    if (e.key === "Enter" && !e.shiftKey) { e.preventDefault(); sendPrompt(); }
  });

  if (els.conversationSearch) {
    let searchDebounce = null;
    els.conversationSearch.addEventListener("input", () => {
      if (searchDebounce) clearTimeout(searchDebounce);
      searchDebounce = setTimeout(() => {
        searchConversations(els.conversationSearch.value);
      }, 180);
    });
  }

  els.settingsNavBtns.forEach((btn) =>
    btn.addEventListener("click", () => switchSettingsSection(btn.dataset.section)));

  if (els.language) {
    els.language.addEventListener("change", () => {
      appState.settings.language = els.language.value;
      persistSettingsToStorage();
    });
  }
  if (els.theme) {
    els.theme.addEventListener("change", () => {
      appState.settings.theme = els.theme.value;
      persistSettingsToStorage();
    });
  }

  if (els.newProviderBtn)  els.newProviderBtn.addEventListener("click", showNewProviderForm);
  if (els.providerForm)    els.providerForm.addEventListener("submit", saveProvider);
  if (els.pfDeleteBtn)     els.pfDeleteBtn.addEventListener("click", deleteCurrentProvider);
  if (els.pfTestBtn)       els.pfTestBtn.addEventListener("click", testProviderConnection);
  if (els.pfPreset)        els.pfPreset.addEventListener("change", () => applyPreset(els.pfPreset.value));
  if (els.pfType)          els.pfType.addEventListener("change", updateProviderURLPlaceholder);

  if (els.showMetaFooter) {
    els.showMetaFooter.addEventListener("change", () => {
      appState.settings.showMetaFooter = els.showMetaFooter.checked;
      persistSettingsToStorage();
      renderMessages();
    });
  }

  // Streaming events from Go backend
  bridge.eventsOn("chat:chunk", (chunk) => {
    if (!chunk || typeof chunk.content !== "string") return;
    appendAssistantChunk(chunk.content);
    if (chunk.done) stopStreaming("done");
  });
  bridge.eventsOn("chat:done",  () => stopStreaming("done"));
  bridge.eventsOn("chat:meta",  (payload) => {
    if (payload && payload.provider_name) attachResponseMeta(payload);
  });
  bridge.eventsOn("chat:tool_call", (payload) => {
    if (payload && payload.tool_name) appendToolCall(payload.tool_name);
  });
  bridge.eventsOn("chat:tool_result", (payload) => {
    if (payload && payload.tool_call_id) updateToolResult(payload.tool_call_id, payload.content);
  });
  bridge.eventsOn("chat:error", (payload) => {
    const msg = (payload && payload.message) ? payload.message : "Generation failed";
    els.status.textContent = msg;
    // Show error text inside the assistant bubble so the user doesn't miss it.
    const conv = activeConversation();
    if (conv && conv.messages.length > 0) {
      const last = conv.messages[conv.messages.length - 1];
      if (last.role === "assistant" && last.content === "") {
        last.content = `_⚠ ${msg}_`;
        renderMessages();
      }
    }
    stopStreaming("cancel");
  });
}

// ── Init ───────────────────────────────────────────────────────────────────
async function init() {
  // Expose message actions to onclick attributes in rendered HTML.
  window.liaotao = { copyMessage, editMessage, regenerateMessage, deleteMessage };

  initSidebarResizer();
  applySidebarState();
  loadSettingsFromStorage();
  applySettingsToUI();
  bindEvents();
  initMCPFormListeners();
  await loadProviders();
  await loadProviderProfiles();
  await loadPersistedConversations();
  renderLastUsedModels();
}

init();

