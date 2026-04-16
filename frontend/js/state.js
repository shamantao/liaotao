/*
  state.js -- Shared application state and DOM references.
  Responsibilities: appState singleton, els (DOM refs), settings persistence helpers.
  Exported as live bindings — never re-assign appState or els; mutate their properties.
*/

// ── App state ──────────────────────────────────────────────────────────────
export const appState = {
  conversations:       [],
  activeAttachments:   [],
  activeConversationId: null,
  streamingTimer:      null,
  isStreaming:         false,
  lastUserPrompt:      "",
  sidebarCollapsed:    false,
  sidebarWidth:        290,
  expandedSidebarWidth: 290,
  conversationSearchQuery: "",
  tags:                [],     // TagRecord[] cached from DB
  activeTagId:         0,      // 0 = all tags filter
  providers:           [],     // ProviderRecord[] cached from DB
  projects:            [],     // ProjectRecord[] cached from DB
  activeProjectId:     0,      // 0 = all projects filter
  activeProjectDashboard: null,
  activeProviderId:    null,   // number | null — currently selected provider
  providerStatus:      {},     // providerId -> "connected" | "disconnected" | "unknown"
  lastUsedModels:      [],     // [{ providerId, providerName, model, usedAt }]
  modelFilterQuery:    "",
  settingsSection:     "general",
  settings: {
    language: "en",
    theme: "dark",
    showMetaFooter: true,
    defaultSystemPrompt: "",
    expertMode: false,
    responseStyle: "balanced",
    chatFontSize: "L",
    autoCheckUpdates: true,    // UPD-02: auto-check for updates on startup
  },
};

// ── DOM refs ───────────────────────────────────────────────────────────────
export const els = {
  appShell:       document.getElementById("app-shell"),
  tabs:           document.querySelectorAll(".tab-btn"),
  panels:         document.querySelectorAll(".tab-panel"),
  status:         document.getElementById("status"),
  // chat
  conversationList: document.getElementById("conversation-list"),
  projectFilter: document.getElementById("project-filter"),
  projectDashboard: document.getElementById("project-dashboard"),
  projectDashboardStats: document.getElementById("project-dashboard-stats"),
  projectRetrievalBackend: document.getElementById("project-retrieval-backend"),
  newProjectBtn: document.getElementById("new-project-btn"),
  renameProjectBtn: document.getElementById("rename-project-btn"),
  archiveProjectBtn: document.getElementById("archive-project-btn"),
  messages:       document.getElementById("messages"),
  prompt:         document.getElementById("prompt"),
  send:           document.getElementById("send-btn"),
  stop:           document.getElementById("stop-btn"),
  newChat:        document.getElementById("new-chat-btn"),
  conversationSearch: document.getElementById("conversation-search"),
  conversationTagFilter: document.getElementById("conversation-tag-filter"),
  newTagBtn: document.getElementById("new-tag-btn"),
  attachmentList: document.getElementById("attachment-list"),
  chatProvider:   document.getElementById("chat-provider"),
  chatModelFilter: document.getElementById("chat-model-filter"),
  chatModel:      document.getElementById("chat-model"),
  chatResponseStyle: document.getElementById("chat-response-style"),
  chatAdvancedFields: document.querySelectorAll(".chat-advanced-field"),
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
  defaultSystemPrompt: document.getElementById("default-system-prompt"),
  expertMode: document.getElementById("expert-mode"),
  autoCheckUpdates: document.getElementById("auto-check-updates"),
  exportConfigBtn: document.getElementById("export-config-btn"),
  importConfigInput: document.getElementById("import-config-input"),
  aboutContent: document.getElementById("about-content"),
  // settings - plugins
  pluginsReloadBtn: document.getElementById("plugins-reload-btn"),
  pluginsList: document.getElementById("plugins-list"),
  pluginsPromptName: document.getElementById("plugins-prompt-name"),
  pluginsPromptContent: document.getElementById("plugins-prompt-content"),
  pluginsPromptList: document.getElementById("plugins-prompt-list"),
  pluginsPromptInsertBtn: document.getElementById("plugins-prompt-insert-btn"),
  pluginsPromptSaveBtn: document.getElementById("plugins-prompt-save-btn"),
  pluginsPromptDeleteBtn: document.getElementById("plugins-prompt-delete-btn"),
  pluginsExportMdBtn: document.getElementById("plugins-export-md-btn"),
  pluginsExportPdfBtn: document.getElementById("plugins-export-pdf-btn"),
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
  // CORCT-01: chat font size selector
  chatFontSize: document.getElementById("chat-font-size"),
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

// Font size map for CORCT-01 (label → [font-size, icon-size])
const FONT_SIZE_MAP = {
  xs: ["0.78rem", "20px"],
  s:  ["0.88rem", "22px"],
  m:  ["0.94rem", "24px"],
  L:  ["1rem",    "26px"],
  XL: ["1.12rem", "30px"],
};

export function applySettingsToUI() {
  if (els.language) els.language.value = appState.settings.language || "en";
  if (els.theme)    els.theme.value    = appState.settings.theme    || "dark";
  if (els.showMetaFooter) els.showMetaFooter.checked = appState.settings.showMetaFooter !== false;
  if (els.defaultSystemPrompt) els.defaultSystemPrompt.value = appState.settings.defaultSystemPrompt || "";
  if (els.expertMode) els.expertMode.checked = Boolean(appState.settings.expertMode);
  if (els.autoCheckUpdates) els.autoCheckUpdates.checked = appState.settings.autoCheckUpdates !== false;
  if (els.chatResponseStyle) els.chatResponseStyle.value = appState.settings.responseStyle || "balanced";
  if (els.chatFontSize) els.chatFontSize.value = appState.settings.chatFontSize || "L";
  const [fontSize, iconSize] = FONT_SIZE_MAP[appState.settings.chatFontSize] ?? FONT_SIZE_MAP["L"];
  document.documentElement.style.setProperty("--chat-font-size", fontSize);
  document.documentElement.style.setProperty("--chat-icon-size", iconSize);
  applyChatModeToUI();
}

export function applyChatModeToUI() {
  const isExpert = Boolean(appState.settings.expertMode);
  if (els.chatAdvancedFields && els.chatAdvancedFields.forEach) {
    els.chatAdvancedFields.forEach((field) => {
      field.style.display = isExpert ? "" : "none";
    });
  }
}
