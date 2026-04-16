/*
  conversations.js -- Conversation sidebar management and persistence.
  Responsibilities: render grouped conversation list, search/rename/delete,
  load persisted conversations from DB, and create new conversation.
*/

import { appState, els, persistSettingsToStorage } from "./state.js";
import { bridge }                  from "./bridge.js";
import { getActiveProvider, syncChatModelSelector } from "./providers.js";
import { renderMessages, loadConversationMessages } from "./chat.js";
import { t }                       from "./i18n.js";
import { emitHook }                from "./plugins.js";
import { loadActiveConversationAttachments } from "./attachments.js";

const defaultProjectName = "Unsorted";

function escapeHTML(value) {
  return String(value)
    .replace(/&/g, "&amp;")
    .replace(/</g, "&lt;")
    .replace(/>/g, "&gt;")
    .replace(/\"/g, "&quot;")
    .replace(/'/g, "&#39;");
}

function formatTokenCount(n) {
  if (n >= 1_000_000) return `${(n / 1_000_000).toFixed(1)}M`;
  if (n >= 1000) return `${(n / 1000).toFixed(1)}k`;
  return String(n);
}

function parseUpdatedAt(value) {
  if (!value || typeof value !== "string") return null;
  const parsed = new Date(value.replace(" ", "T"));
  return Number.isNaN(parsed.getTime()) ? null : parsed;
}

function formatConversationDateTime(updatedAt) {
  const date = parseUpdatedAt(updatedAt);
  if (!date) return "";
  return formatLocalizedDateTime(date);
}

function pad2(value) {
  return String(value).padStart(2, "0");
}

function formatLocalizedDateTime(date) {
  const year = date.getFullYear();
  const month = pad2(date.getMonth() + 1);
  const day = pad2(date.getDate());
  const minutes = pad2(date.getMinutes());
  const lang = appState.settings.language || "en";

  if (lang === "fr") {
    return `${day}/${month}/${year} ${pad2(date.getHours())}:${minutes}`;
  }
  if (lang === "zh-TW") {
    return `${year}-${month}-${day} ${pad2(date.getHours())}:${minutes}`;
  }

  const hours24 = date.getHours();
  const suffix = hours24 >= 12 ? "pm" : "am";
  const hours12 = hours24 % 12 || 12;
  return `${month}/${day}/${year} ${pad2(hours12)}:${minutes}${suffix}`;
}

function localizeConversationTitle(title, fallbackID = 0) {
  const rawTitle = String(title || "").trim();
  if (!rawTitle) return t("sidebar.conversation_fallback", { id: fallbackID || "" }).trim();
  if (rawTitle === "New chat") return t("sidebar.new_chat");
  const match = rawTitle.match(/^Conversation\s+(\d+)$/i);
  if (match) return t("sidebar.conversation_fallback", { id: match[1] });
  return rawTitle;
}

function dateOnly(date) {
  return new Date(date.getFullYear(), date.getMonth(), date.getDate());
}

function getConversationGroup(updatedAt) {
  const date = parseUpdatedAt(updatedAt);
  if (!date) return t("sidebar.older");

  const now = new Date();
  const today = dateOnly(now);
  const target = dateOnly(date);
  const diffDays = Math.floor((today.getTime() - target.getTime()) / 86400000);

  if (diffDays === 0) return t("sidebar.today");
  if (diffDays === 1) return t("sidebar.yesterday");
  if (diffDays >= 2 && diffDays <= 6) return t("sidebar.this_week");
  return t("sidebar.older");
}

function groupConversations(items) {
  const groups = new Map([
    [t("sidebar.today"),     []],
    [t("sidebar.yesterday"), []],
    [t("sidebar.this_week"), []],
    [t("sidebar.older"),     []],
  ]);
  items.forEach((conv) => {
    groups.get(getConversationGroup(conv.updatedAt)).push(conv);
  });
  return groups;
}

function renderEmptyConversationState() {
  const key = appState.conversationSearchQuery
    ? "sidebar.no_conversations_search"
    : "sidebar.no_conversations";
  els.conversationList.innerHTML = `<p class="conversation-empty">${t(key)}</p>`;
}

function inlineConfirm(btn, onConfirm) {
  if (btn.dataset.confirming === "1") {
    delete btn.dataset.confirming;
    clearTimeout(Number(btn.dataset.confirmTimer));
    btn.innerHTML = btn.dataset.origLabel || "🗑";
    btn.title = btn.dataset.origTitle || t("sidebar.delete_title");
    onConfirm();
    return;
  }
  btn.dataset.confirming = "1";
  btn.dataset.origLabel = btn.innerHTML;
  btn.dataset.origTitle = btn.title || t("sidebar.delete_title");
  btn.innerHTML = "✓";
  btn.title = t("sidebar.confirm_deletion");
  btn.dataset.confirmTimer = String(setTimeout(() => {
    delete btn.dataset.confirming;
    btn.innerHTML = btn.dataset.origLabel || "🗑";
    btn.title = btn.dataset.origTitle || t("sidebar.delete_title");
  }, 3000));
}

function activeProject() {
  if (appState.activeProjectId <= 0) return null;
  return appState.projects.find((p) => p.id === appState.activeProjectId) || null;
}

function renderProjectFilter() {
  if (!els.projectFilter) return;
  const options = [`<option value="0">${escapeHTML(t("sidebar.all_projects"))}</option>`]
    .concat(appState.projects.map((project) => `<option value="${project.id}">${escapeHTML(project.name)}</option>`));
  els.projectFilter.innerHTML = options.join("");
  els.projectFilter.value = String(appState.activeProjectId || 0);
}

function renderProjectDashboard() {
  if (!els.projectDashboard || !els.projectDashboardStats || !els.projectRetrievalBackend) return;
  if (appState.activeProjectId <= 0 || !appState.activeProjectDashboard) {
    els.projectDashboardStats.innerHTML = `<span>${escapeHTML(t("sidebar.project_dashboard_empty"))}</span>`;
    els.projectRetrievalBackend.value = "local";
    els.projectRetrievalBackend.disabled = true;
    return;
  }

  const dashboard = appState.activeProjectDashboard;
  els.projectRetrievalBackend.disabled = false;
  els.projectRetrievalBackend.value = dashboard.retrievalBackend || "local";
  const statusClass = dashboard.retrievalStatus === "indexing"
    ? "retrieval-status-indexing"
    : "retrieval-status-ready";
  const statusLabel = dashboard.retrievalStatus === "indexing"
    ? t("sidebar.retrieval_status_indexing")
    : t("sidebar.retrieval_status_ready");

  els.projectDashboardStats.innerHTML = [
    `${escapeHTML(t("sidebar.dashboard_conversations", { count: String(dashboard.conversationCount || 0) }))}`,
    `${escapeHTML(t("sidebar.dashboard_tokens", { count: String(dashboard.totalTokens || 0) }))}`,
    `${escapeHTML(t("sidebar.dashboard_files", { count: String(dashboard.fileCount || 0) }))}`,
    `${escapeHTML(t("sidebar.dashboard_project_knowledge", { count: String(dashboard.projectKnowledgeCount || 0) }))}`,
    `<span class="${statusClass}">${escapeHTML(t("sidebar.dashboard_retrieval_status", { status: statusLabel }))}</span>`,
  ].join("<br>");
}

export async function refreshProjectDashboard() {
  if (appState.activeProjectId <= 0) {
    appState.activeProjectDashboard = null;
    renderProjectDashboard();
    return;
  }
  const dashboard = await bridge.callService("GetProjectDashboard", {
    project_id: Number(appState.activeProjectId),
  });
  appState.activeProjectDashboard = {
    projectID: Number(dashboard.project_id) || 0,
    conversationCount: Number(dashboard.conversation_count) || 0,
    totalTokens: Number(dashboard.total_tokens) || 0,
    fileCount: Number(dashboard.file_count) || 0,
    projectKnowledgeCount: Number(dashboard.project_knowledge_count) || 0,
    retrievalBackend: String(dashboard.retrieval_backend || "local"),
    retrievalStatus: String(dashboard.retrieval_status || "ready"),
  };
  renderProjectDashboard();
}

export async function loadProjects() {
  const result = await bridge.callService("ListProjects", { include_archived: false });
  appState.projects = Array.isArray(result)
    ? result.map((item) => ({
      id: Number(item.id) || 0,
      name: String(item.name || ""),
      description: String(item.description || ""),
      archived: Boolean(item.archived),
    })).filter((item) => item.id > 0)
    : [];

  if (appState.activeProjectId > 0 && !appState.projects.find((p) => p.id === appState.activeProjectId)) {
    appState.activeProjectId = 0;
  }
  renderProjectFilter();
  await refreshProjectDashboard();
  await loadTags();
}

async function loadTags() {
  const result = await bridge.callService("ListTags");
  appState.tags = Array.isArray(result)
    ? result.map((item) => ({
      id: Number(item.id) || 0,
      name: String(item.name || ""),
      color: String(item.color || "#6c757d"),
    })).filter((item) => item.id > 0)
    : [];
  renderTagFilter();
}

function renderTagFilter() {
  if (!els.conversationTagFilter) return;
  const options = [`<option value="0">${escapeHTML(t("sidebar.tag_filter_all"))}</option>`]
    .concat(appState.tags.map((tag) => `<option value="${tag.id}">${escapeHTML(tag.name)}</option>`));
  els.conversationTagFilter.innerHTML = options.join("");
  els.conversationTagFilter.value = String(Number(appState.activeTagId) || 0);
}

async function fetchConversationSummaries(limit = 100) {
  const projectID = Number(appState.activeProjectId) || 0;
  const tagID = Number(appState.activeTagId) || 0;
  if (tagID > 0) {
    return bridge.callService("ListConversationsByTag", {
      tag_id: tagID,
      limit,
      project_id: projectID,
    });
  }
  if (appState.conversationSearchQuery) {
    return bridge.callService("SearchConversations", {
      query: appState.conversationSearchQuery,
      limit,
      project_id: projectID,
    });
  }
  return bridge.callService("ListConversations", { limit, project_id: projectID });
}

async function activateConversation(conv) {
  appState.activeConversationId = conv.id;
  if (conv.providerId > 0) {
    const prov = appState.providers.find((p) => p.id === conv.providerId && p.active);
    if (prov) {
      appState.activeProviderId = prov.id;
      els.chatProvider.value = String(prov.id);
      persistSettingsToStorage();
    }
  } else {
    appState.activeProviderId = 0;
    els.chatProvider.value = "0";
    persistSettingsToStorage();
  }
  if (els.chatTemperature) els.chatTemperature.value = String(Number(conv.temperature) || 0.7);
  if (els.chatMaxTokens) els.chatMaxTokens.value = String(Number(conv.maxTokens) || 0);
  if (els.chatSystemPrompt) els.chatSystemPrompt.value = conv.systemPrompt || "";
  syncChatModelSelector(conv.model || "");
  renderConversationList();
  await loadConversationMessages(conv.id);
  await loadActiveConversationAttachments();
}

function mapConversationSummary(item) {
  return {
    id: item.id,
    title: localizeConversationTitle(item.title, item.id),
    projectId: Number(item.project_id) || 1,
    projectName: String(item.project || defaultProjectName),
    providerName: item.provider || "",
    providerId: item.provider_id || 0,
    model: item.model || els.chatModel.value,
    temperature: Number(item.temperature) > 0 ? Number(item.temperature) : 0.7,
    maxTokens: Number(item.max_tokens) > 0 ? Number(item.max_tokens) : 0,
    systemPrompt: item.system_prompt || "",
    updatedAt: item.updated_at || "",
    tokenCount: Number(item.token_count) || 0,
    tags: Array.isArray(item.tags) ? item.tags : [],
    messages: [],
  };
}

export async function saveActiveConversationSettings() {
  const conv = appState.conversations.find((c) => c.id === appState.activeConversationId);
  if (!conv) return;
  const providerID = Number(conv.providerId) || 0;
  const model = String(conv.model || "").trim();
  if (!model) return;

  const payload = {
    conversation_id: conv.id,
    provider_id: providerID,
    model,
    temperature: Number(conv.temperature) > 0 ? Number(conv.temperature) : 0.7,
    max_tokens: Math.max(0, Number(conv.maxTokens) || 0),
    system_prompt: String(conv.systemPrompt || ""),
  };

  const updated = await bridge.callService("UpdateConversationSettings", payload);
  if (updated && updated.id === conv.id) {
    conv.providerId = Number(updated.provider_id) || 0;
    conv.providerName = updated.provider || conv.providerName;
    conv.model = updated.model || conv.model;
    conv.temperature = Number(updated.temperature) > 0 ? Number(updated.temperature) : conv.temperature;
    conv.maxTokens = Number(updated.max_tokens) > 0 ? Number(updated.max_tokens) : 0;
    conv.systemPrompt = updated.system_prompt || "";
    conv.updatedAt = updated.updated_at || conv.updatedAt;
    await emitHook("onSaveConv", {
      conversationId: conv.id,
      providerId: conv.providerId,
      model: conv.model,
    });
    renderConversationList();
  }
}

async function reloadConversationList(preferredConversationID = 0) {
  const result = await fetchConversationSummaries(100);
  if (!Array.isArray(result) || result.length === 0) {
    appState.conversations = [];
    appState.activeConversationId = null;
    renderConversationList();
    renderMessages();
    await loadActiveConversationAttachments();
    return false;
  }

  appState.conversations = result.map(mapConversationSummary);
  const preferred = appState.conversations.find((conv) => conv.id === preferredConversationID) || appState.conversations[0];
  await activateConversation(preferred);
  return true;
}

async function renameConversation(conversationID, title) {
  const updated = await bridge.callService("RenameConversation", {
    conversation_id: conversationID,
    title,
  });
  await reloadConversationList(updated && updated.id ? updated.id : conversationID);
  els.status.textContent = t("sidebar.renamed");
}

function startRenameConversation(row, conv) {
  if (!row || row.dataset.renaming === "1") return;
  row.dataset.renaming = "1";

  const main = row.querySelector(".conversation-main");
  const actions = row.querySelector(".conversation-row-actions");
  if (!main || !actions) return;

  const originalTitle = conv.title || "";
  main.innerHTML = `<input class="conversation-rename-input" type="text" aria-label="${t("sidebar.rename_save")}" value="${escapeHTML(originalTitle)}">`;
  actions.innerHTML = `
    <button class="conversation-rename-btn icon-only-btn" type="button" title="${t("sidebar.rename_save")}" aria-label="${t("sidebar.rename_save")}">✓</button>
    <button class="conversation-delete-btn icon-only-btn" type="button" title="${t("sidebar.rename_cancel")}" aria-label="${t("sidebar.rename_cancel")}">✕</button>
  `;

  const input = main.querySelector(".conversation-rename-input");
  const saveBtn = actions.querySelector(".conversation-rename-btn");
  const cancelBtn = actions.querySelector(".conversation-delete-btn");
  if (!input || !saveBtn || !cancelBtn) return;

  const cancel = () => {
    delete row.dataset.renaming;
    renderConversationList();
  };

  const save = async () => {
    const nextTitle = input.value.trim();
    if (!nextTitle) {
      els.status.textContent = t("sidebar.title_required");
      input.focus();
      return;
    }
    if (nextTitle === originalTitle) {
      cancel();
      return;
    }
    try {
      await renameConversation(conv.id, nextTitle);
    } catch (err) {
      els.status.textContent = `rename failed: ${String(err && err.message ? err.message : err)}`;
      cancel();
    }
  };

  input.addEventListener("click", (event) => event.stopPropagation());
  input.addEventListener("keydown", (event) => {
    if (event.key === "Enter") {
      event.preventDefault();
      save();
      return;
    }
    if (event.key === "Escape") {
      event.preventDefault();
      cancel();
    }
  });
  saveBtn.addEventListener("click", (event) => {
    event.stopPropagation();
    save();
  });
  cancelBtn.addEventListener("click", (event) => {
    event.stopPropagation();
    cancel();
  });

  input.focus();
  input.select();
}

async function deleteConversation(conversationID) {
  const deletedIndex = appState.conversations.findIndex((conv) => conv.id === conversationID);
  const fallbackConversation = deletedIndex >= 0
    ? appState.conversations[Math.min(deletedIndex + 1, appState.conversations.length - 1)] ||
      appState.conversations[Math.max(0, deletedIndex - 1)]
    : null;

  await bridge.callService("DeleteConversation", conversationID);
  const stillHasConversations = await reloadConversationList(fallbackConversation ? fallbackConversation.id : 0);
  if (!stillHasConversations) {
    await newConversation();
  }

  els.status.textContent = t("sidebar.deleted");
}

// ── Conversation sidebar ───────────────────────────────────────────────────
export function renderConversationList() {
  els.conversationList.innerHTML = "";
  if (appState.conversations.length === 0) {
    renderEmptyConversationState();
    return;
  }

  // Close any open context menu when re-rendering.
  closeAllConvMenus();

  const grouped = groupConversations(appState.conversations);
  grouped.forEach((groupItems, groupName) => {
    if (groupItems.length === 0) return;

    const section = document.createElement("section");
    section.className = "conversation-group";
    section.innerHTML = `<h4 class="conversation-group-title">${groupName}</h4>`;

    groupItems.forEach((conv) => {
      const row = document.createElement("div");
      row.className = `conversation-item${conv.id === appState.activeConversationId ? " active" : ""}`;
      row.innerHTML = `
        <span class="dot">${localizeConversationTitle(conv.title, conv.id).slice(0, 1).toUpperCase()}</span>
        <div class="conversation-main">
          <span class="label">${escapeHTML(localizeConversationTitle(conv.title, conv.id))}</span>
          <span class="conversation-meta">${formatConversationDateTime(conv.updatedAt)}</span>
        </div>
        <div class="conversation-row-actions">
          <button class="conv-menu-btn icon-only-btn" type="button" title="${t("sidebar.more_actions")}" aria-label="${t("sidebar.more_actions")}">⋯</button>
          <div class="conv-menu" role="menu" hidden>
            <button class="conv-menu-item conv-menu-rename" data-i18n="sidebar.rename_action">${t("sidebar.rename_action")}</button>
            <button class="conv-menu-item conv-menu-export-md">${t("sidebar.export_md")}</button>
            <button class="conv-menu-item conv-menu-export-json">${t("sidebar.export_json")}</button>
            <button class="conv-menu-item conv-menu-delete danger">${t("sidebar.delete_action")}</button>
          </div>
        </div>
      `;

      row.onclick = async () => {
        if (row.dataset.renaming === "1") return;
        await activateConversation(conv);
      };

      const menuBtn = row.querySelector(".conv-menu-btn");
      const menu = row.querySelector(".conv-menu");

      menuBtn.addEventListener("click", (event) => {
        event.stopPropagation();
        const isOpen = !menu.hidden;
        closeAllConvMenus();
        if (!isOpen) {
          menu.hidden = false;
          row.classList.add("menu-open");
        }
      });

      row.querySelector(".conv-menu-rename").addEventListener("click", (event) => {
        event.stopPropagation();
        closeAllConvMenus();
        startRenameConversation(row, conv);
      });

      row.querySelector(".conv-menu-export-md").addEventListener("click", async (event) => {
        event.stopPropagation();
        closeAllConvMenus();
        try {
          const result = await bridge.callService("ExportConversation", {
            conversation_id: conv.id,
            format: "markdown",
          });
          if (result && result.file_path) {
            els.status.textContent = t("sidebar.export_done", { path: result.file_path });
          }
        } catch (err) {
          els.status.textContent = `export failed: ${String(err && err.message ? err.message : err)}`;
        }
      });

      row.querySelector(".conv-menu-export-json").addEventListener("click", async (event) => {
        event.stopPropagation();
        closeAllConvMenus();
        try {
          const result = await bridge.callService("ExportConversation", {
            conversation_id: conv.id,
            format: "json",
          });
          if (result && result.file_path) {
            els.status.textContent = t("sidebar.export_done", { path: result.file_path });
          }
        } catch (err) {
          els.status.textContent = `export failed: ${String(err && err.message ? err.message : err)}`;
        }
      });

      row.querySelector(".conv-menu-delete").addEventListener("click", (event) => {
        event.stopPropagation();
        closeAllConvMenus();
        if (!window.confirm(t("sidebar.confirm_deletion"))) return;
        deleteConversation(conv.id);
      });

      section.appendChild(row);
    });

    els.conversationList.appendChild(section);
  });

  // Close open menus when clicking elsewhere.
  document.addEventListener("click", closeAllConvMenus, { once: true });
}

function closeAllConvMenus() {
  document.querySelectorAll(".conv-menu").forEach((m) => {
    m.hidden = true;
  });
  document.querySelectorAll(".conversation-item.menu-open").forEach((r) => {
    r.classList.remove("menu-open");
  });
}

export async function loadPersistedConversations() {
  if (els.conversationSearch) {
    els.conversationSearch.value = appState.conversationSearchQuery;
  }
  await loadTags();
  const loaded = await reloadConversationList(appState.activeConversationId || 0);
  if (!loaded && !appState.conversationSearchQuery) {
    await newConversation();
  }
}

export async function searchConversations(query) {
  const normalizedQuery = (query || "").trim();
  appState.conversationSearchQuery = normalizedQuery;
  if (els.conversationSearch && els.conversationSearch.value !== normalizedQuery) {
    els.conversationSearch.value = normalizedQuery;
  }
  const loaded = await reloadConversationList(appState.activeConversationId || 0);
  if (!loaded && !appState.conversationSearchQuery) {
    await newConversation();
  }
}

export async function newConversation() {
  const selectedModelRaw = String(els.chatModel.value || "");
  const unifiedSelection = selectedModelRaw.includes("::") ? selectedModelRaw.split("::") : null;
  const selectedProviderFromModel = unifiedSelection ? Number(unifiedSelection[0]) || 0 : 0;
  const selectedModel = unifiedSelection ? (unifiedSelection[1] || "") : selectedModelRaw;
  const prov    = selectedProviderFromModel > 0
    ? (appState.providers.find((p) => p.id === selectedProviderFromModel) || null)
    : getActiveProvider();
  const created = await bridge.callService("CreateConversation", {
    title:       t("sidebar.new_chat"),
    project_id:  Number(appState.activeProjectId) || 0,
    provider_id: prov ? prov.id : 0,
    model:       selectedModel,
  });
  if (!created || typeof created.id !== "number") {
    els.status.textContent = "create conversation failed";
    return;
  }
  const conv = {
    id:           created.id,
    title:        localizeConversationTitle(created.title, appState.conversations.length + 1),
    projectId:    Number(created.project_id) || 1,
    projectName:  String(created.project || defaultProjectName),
    providerName: prov ? prov.name : "",
    providerId:   prov ? prov.id : 0,
    model:        created.model || selectedModel,
    temperature:  Number(created.temperature) > 0 ? Number(created.temperature) : 0.7,
    maxTokens:    Number(created.max_tokens) > 0 ? Number(created.max_tokens) : 0,
    systemPrompt: created.system_prompt || "",
    updatedAt:    created.updated_at || new Date().toISOString(),
    messages:     [],
  };
  appState.conversations.unshift(conv);
  appState.activeConversationId = conv.id;
  syncChatModelSelector(conv.model || "");
  renderConversationList();
  renderMessages();
  await loadActiveConversationAttachments();
  els.prompt.focus();
}

export function bindTagControls() {
  if (els.conversationTagFilter) {
    els.conversationTagFilter.addEventListener("change", async () => {
      appState.activeTagId = Number(els.conversationTagFilter.value) || 0;
      await loadPersistedConversations();
    });
  }

  if (els.newTagBtn) {
    els.newTagBtn.addEventListener("click", async () => {
      const name = window.prompt(t("sidebar.new_tag_name") || "Tag name");
      if (!name || !name.trim()) return;
      try {
        const created = await bridge.callService("CreateTag", {
          name: name.trim(),
          color: "#4b6cb7",
        });
        if (appState.activeConversationId) {
          await bridge.callService("AddTagToConversation", {
            conversation_id: Number(appState.activeConversationId),
            tag_id: Number(created.id),
          });
          els.status.textContent = t("sidebar.tag_added");
        } else {
          els.status.textContent = t("sidebar.tag_created");
        }
        await loadTags();
        await loadPersistedConversations();
      } catch (err) {
        els.status.textContent = `tag failed: ${String(err && err.message ? err.message : err)}`;
      }
    });
  }
}

export function bindProjectControls() {
  if (els.projectFilter) {
    els.projectFilter.addEventListener("change", async () => {
      appState.activeProjectId = Number(els.projectFilter.value) || 0;
      await refreshProjectDashboard();
      await loadPersistedConversations();
    });
  }

  if (els.projectRetrievalBackend) {
    els.projectRetrievalBackend.addEventListener("change", async () => {
      if (appState.activeProjectId <= 0) return;
      await bridge.callService("SetProjectRetrievalBackend", {
        project_id: Number(appState.activeProjectId),
        backend: els.projectRetrievalBackend.value || "local",
      });
      await refreshProjectDashboard();
      els.status.textContent = t("sidebar.retrieval_backend_updated");
    });
  }

  window.addEventListener("liaotao:project-dashboard-refresh", () => {
    void refreshProjectDashboard();
  });

  if (els.newProjectBtn) {
    els.newProjectBtn.addEventListener("click", async () => {
      const name = window.prompt(t("sidebar.new_project"));
      if (!name || !name.trim()) {
        els.status.textContent = t("sidebar.project_name_required");
        return;
      }
      await bridge.callService("CreateProject", { name: name.trim(), description: "" });
      await loadProjects();
      els.status.textContent = t("sidebar.project_created");
    });
  }

  if (els.renameProjectBtn) {
    els.renameProjectBtn.addEventListener("click", async () => {
      const project = activeProject();
      if (!project) return;
      const name = window.prompt(t("sidebar.rename_project"), project.name);
      if (!name || !name.trim()) {
        els.status.textContent = t("sidebar.project_name_required");
        return;
      }
      await bridge.callService("RenameProject", { project_id: project.id, name: name.trim() });
      await loadProjects();
      els.status.textContent = t("sidebar.project_renamed");
    });
  }

  if (els.archiveProjectBtn) {
    els.archiveProjectBtn.addEventListener("click", async () => {
      const project = activeProject();
      if (!project) return;
      const ok = window.confirm(t("sidebar.archive_project_confirm", { name: project.name }));
      if (!ok) return;
      await bridge.callService("ArchiveProject", { project_id: project.id, archived: true });
      appState.activeProjectId = 0;
      await loadProjects();
      await loadPersistedConversations();
      els.status.textContent = t("sidebar.project_archived");
    });
  }
}
