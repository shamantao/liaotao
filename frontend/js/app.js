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
  saveProvider, deleteCurrentProvider, showNewProviderForm,
  testProviderConnection, applyPreset,
} from "./providers.js";
import {
  renderMessages, sendPrompt, cancelGeneration,
  appendAssistantChunk, stopStreaming, activeConversation,
  copyMessage, editMessage, regenerateMessage, deleteMessage,
  attachResponseMeta,
} from "./chat.js";
import { newConversation, loadPersistedConversations } from "./conversations.js";

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
  els.refreshModels.addEventListener("click", loadModels);

  els.chatProvider.addEventListener("change", async () => {
    // Parse the selected value. "0" is the Automat (router) option — must stay 0, not coerced to null.
    const raw = els.chatProvider.value;
    const newId = raw === "" ? null : Number(raw);
    appState.activeProviderId = newId;
    persistSettingsToStorage();
    await loadModels();
  });

  els.chatModel.addEventListener("change", () => {
    const conv = appState.conversations.find((c) => c.id === appState.activeConversationId);
    if (conv) conv.model = els.chatModel.value;
  });

  els.prompt.addEventListener("keydown", (e) => {
    if (e.key === "Enter" && !e.shiftKey) { e.preventDefault(); sendPrompt(); }
  });

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
  await loadProviders();
  await loadProviderProfiles();
  await loadModels();
  await loadPersistedConversations();
}

init();

