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
  lastUserPrompt:      "",
  sidebarCollapsed:    false,
  sidebarWidth:        290,
  expandedSidebarWidth: 290,
  providers:           [],     // ProviderRecord[] cached from DB
  activeProviderId:    null,   // number | null — currently selected provider
  settingsSection:     "general",
  settings: { language: "fr", theme: "dark" },
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
  chatProvider:   document.getElementById("chat-provider"),
  chatModel:      document.getElementById("chat-model"),
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
  } catch {
    // ignore corrupt storage
  }
}

export function persistSettingsToStorage() {
  localStorage.setItem(STORAGE_KEY, JSON.stringify({
    settings:         appState.settings,
    activeProviderId: appState.activeProviderId,
  }));
}

export function applySettingsToUI() {
  if (els.language) els.language.value = appState.settings.language || "fr";
  if (els.theme)    els.theme.value    = appState.settings.theme    || "dark";
}
