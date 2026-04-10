/*
  state.js -- Shared application state and DOM references.
  Responsibilities: appState singleton, els (DOM refs), settings persistence helpers.
  Exported as live bindings — never re-assign appState or els; mutate their properties.
*/

// ── App state ──────────────────────────────────────────────────────────────
export const appState = {
  conversations:       [],
  activeConversationId: null,
  streamingTimer:      null,
  isStreaming:         false,
  lastUserPrompt:      "",
  sidebarCollapsed:    false,
  sidebarWidth:        290,
  expandedSidebarWidth: 290,
  conversationSearchQuery: "",
  providers:           [],     // ProviderRecord[] cached from DB
  activeProviderId:    null,   // number | null — currently selected provider
  providerStatus:      {},     // providerId -> "connected" | "disconnected" | "unknown"
  lastUsedModels:      [],     // [{ providerId, providerName, model, usedAt }]
  modelFilterQuery:    "",
  settingsSection:     "general",
  settings: { language: "fr", theme: "dark", showMetaFooter: true },
};

// ── DOM refs ───────────────────────────────────────────────────────────────
export const els = {
  appShell:       document.getElementById("app-shell"),
  tabs:           document.querySelectorAll(".tab-btn"),
  panels:         document.querySelectorAll(".tab-panel"),
  status:         document.getElementById("status"),
  // chat
  conversationList: document.getElementById("conversation-list"),
  messages:       document.getElementById("messages"),
  prompt:         document.getElementById("prompt"),
  send:           document.getElementById("send-btn"),
  stop:           document.getElementById("stop-btn"),
  newChat:        document.getElementById("new-chat-btn"),
  conversationSearch: document.getElementById("conversation-search"),
  chatProvider:   document.getElementById("chat-provider"),
  chatModelFilter: document.getElementById("chat-model-filter"),
  chatModel:      document.getElementById("chat-model"),
  chatTemperature: document.getElementById("chat-temperature"),
  chatMaxTokens:   document.getElementById("chat-max-tokens"),
  chatSystemPrompt: document.getElementById("chat-system-prompt"),
  lastUsedModels: document.getElementById("last-used-models"),
  refreshModels:  document.getElementById("refresh-models-btn"),
  sidebarToggle:  document.getElementById("sidebar-toggle"),
  sidebarResizer: document.getElementById("sidebar-resizer"),
  // settings nav
  settingsNavBtns:  document.querySelectorAll(".settings-nav-btn"),
  settingsSections: document.querySelectorAll(".settings-section"),
  // settings – general
  language: document.getElementById("language"),
  theme:    document.getElementById("theme"),
  // settings – providers CRUD
  providersList:           document.getElementById("providers-list"),
  newProviderBtn:          document.getElementById("new-provider-btn"),
  providerForm:            document.getElementById("provider-form"),
  providerFormPlaceholder: document.querySelector(".provider-form-placeholder"),
  pfId:          document.getElementById("pf-id"),
  pfName:        document.getElementById("pf-name"),
  pfType:        document.getElementById("pf-type"),
  pfUrl:         document.getElementById("pf-url"),
  pfKey:         document.getElementById("pf-key"),
  pfDescription: document.getElementById("pf-description"),
  pfRag:         document.getElementById("pf-rag"),
  pfActive:      document.getElementById("pf-active"),
  pfTemperature: document.getElementById("pf-temperature"),
  pfNumCtx:      document.getElementById("pf-num-ctx"),
  pfDeleteBtn:   document.getElementById("pf-delete-btn"),
  // PROV-08: preset selector
  pfPreset:   document.getElementById("pf-preset"),
  pfDocsLink: document.getElementById("pf-docs-link"),
  // PROV-05: test connection
  pfTestBtn:    document.getElementById("pf-test-btn"),
  pfTestResult: document.getElementById("pf-test-result"),
  // ROUTER-08: response metadata footer toggle
  showMetaFooter: document.getElementById("show-meta-footer"),
};

// ── Settings persistence (localStorage) ───────────────────────────────────
const STORAGE_KEY = "liaotao.settings.v2";

export function loadSettingsFromStorage() {
  try {
    const raw = localStorage.getItem(STORAGE_KEY);
    if (!raw) return;
    const parsed = JSON.parse(raw);
    appState.settings = { ...appState.settings, ...parsed.settings };
    if (parsed.activeProviderId != null) {
      appState.activeProviderId = parsed.activeProviderId;
    }
    if (Array.isArray(parsed.lastUsedModels)) {
      appState.lastUsedModels = parsed.lastUsedModels.slice(0, 6);
    }
  } catch {
    // ignore corrupt storage
  }
}

export function persistSettingsToStorage() {
  localStorage.setItem(STORAGE_KEY, JSON.stringify({
    settings:         appState.settings,
    activeProviderId: appState.activeProviderId,
    lastUsedModels:   appState.lastUsedModels,
  }));
}

export function applySettingsToUI() {
  if (els.language) els.language.value = appState.settings.language || "fr";
  if (els.theme)    els.theme.value    = appState.settings.theme    || "dark";
  if (els.showMetaFooter) els.showMetaFooter.checked = appState.settings.showMetaFooter !== false;
}
