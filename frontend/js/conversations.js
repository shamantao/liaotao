/*
  conversations.js -- Conversation sidebar management and persistence.
  Responsibilities: render grouped conversation list, search/rename/delete,
  load persisted conversations from DB, and create new conversation.
*/

import { appState, els, persistSettingsToStorage } from "./state.js";
import { bridge }                  from "./bridge.js";
import { getActiveProvider, syncChatModelSelector } from "./providers.js";
import { renderMessages, loadConversationMessages } from "./chat.js";

const SIDEBAR_DATE_TIME_FORMAT = new Intl.DateTimeFormat(undefined, {
  month: "short",
  day: "2-digit",
  hour: "2-digit",
  minute: "2-digit",
});

function escapeHTML(value) {
  return String(value)
    .replace(/&/g, "&amp;")
    .replace(/</g, "&lt;")
    .replace(/>/g, "&gt;")
    .replace(/\"/g, "&quot;")
    .replace(/'/g, "&#39;");
}

function parseUpdatedAt(value) {
  if (!value || typeof value !== "string") return null;
  const parsed = new Date(value.replace(" ", "T"));
  return Number.isNaN(parsed.getTime()) ? null : parsed;
}

function formatConversationDateTime(updatedAt) {
  const date = parseUpdatedAt(updatedAt);
  if (!date) return "";
  return SIDEBAR_DATE_TIME_FORMAT.format(date);
}

function dateOnly(date) {
  return new Date(date.getFullYear(), date.getMonth(), date.getDate());
}

function getConversationGroup(updatedAt) {
  const date = parseUpdatedAt(updatedAt);
  if (!date) return "Older";

  const now = new Date();
  const today = dateOnly(now);
  const target = dateOnly(date);
  const diffDays = Math.floor((today.getTime() - target.getTime()) / 86400000);

  if (diffDays === 0) return "Today";
  if (diffDays === 1) return "Yesterday";
  if (diffDays >= 2 && diffDays <= 6) return "This Week";
  return "Older";
}

function groupConversations(items) {
  const groups = new Map([
    ["Today", []],
    ["Yesterday", []],
    ["This Week", []],
    ["Older", []],
  ]);
  items.forEach((conv) => {
    groups.get(getConversationGroup(conv.updatedAt)).push(conv);
  });
  return groups;
}

function renderEmptyConversationState() {
  const suffix = appState.conversationSearchQuery
    ? " for this search."
    : ".";
  els.conversationList.innerHTML = `<p class="conversation-empty">No conversations${suffix}</p>`;
}

function inlineConfirm(btn, onConfirm) {
  if (btn.dataset.confirming === "1") {
    delete btn.dataset.confirming;
    clearTimeout(Number(btn.dataset.confirmTimer));
    btn.innerHTML = btn.dataset.origLabel || "🗑";
    onConfirm();
    return;
  }
  btn.dataset.confirming = "1";
  btn.dataset.origLabel = btn.innerHTML;
  btn.innerHTML = "✓";
  btn.title = "Confirm deletion";
  btn.dataset.confirmTimer = String(setTimeout(() => {
    delete btn.dataset.confirming;
    btn.innerHTML = btn.dataset.origLabel || "🗑";
    btn.title = "Delete conversation";
  }, 3000));
}

async function fetchConversationSummaries(limit = 100) {
  if (appState.conversationSearchQuery) {
    return bridge.callService("SearchConversations", {
      query: appState.conversationSearchQuery,
      limit,
    });
  }
  return bridge.callService("ListConversations", { limit });
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
  syncChatModelSelector(conv.model || "");
  renderConversationList();
  await loadConversationMessages(conv.id);
}

function mapConversationSummary(item) {
  return {
    id: item.id,
    title: item.title || `Conversation ${item.id}`,
    providerName: item.provider || "",
    providerId: item.provider_id || 0,
    model: item.model || els.chatModel.value,
    updatedAt: item.updated_at || "",
    messages: [],
  };
}

async function reloadConversationList(preferredConversationID = 0) {
  const result = await fetchConversationSummaries(100);
  if (!Array.isArray(result) || result.length === 0) {
    appState.conversations = [];
    appState.activeConversationId = null;
    renderConversationList();
    renderMessages();
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
  els.status.textContent = "conversation renamed";
}

function startRenameConversation(row, conv) {
  if (!row || row.dataset.renaming === "1") return;
  row.dataset.renaming = "1";

  const main = row.querySelector(".conversation-main");
  const actions = row.querySelector(".conversation-row-actions");
  if (!main || !actions) return;

  const originalTitle = conv.title || "";
  main.innerHTML = `<input class="conversation-rename-input" type="text" aria-label="Rename conversation" value="${escapeHTML(originalTitle)}">`;
  actions.innerHTML = `
    <button class="conversation-rename-btn icon-only-btn" type="button" title="Save" aria-label="Save">✓</button>
    <button class="conversation-delete-btn icon-only-btn" type="button" title="Cancel" aria-label="Cancel">✕</button>
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
      els.status.textContent = "title required";
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

  els.status.textContent = "conversation deleted";
}

// ── Conversation sidebar ───────────────────────────────────────────────────
export function renderConversationList() {
  els.conversationList.innerHTML = "";
  if (appState.conversations.length === 0) {
    renderEmptyConversationState();
    return;
  }

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
        <span class="dot">${(conv.title || "?").slice(0, 1).toUpperCase()}</span>
        <div class="conversation-main">
          <span class="label">${escapeHTML(conv.title)}</span>
          <span class="conversation-meta">${formatConversationDateTime(conv.updatedAt)}</span>
        </div>
        <div class="conversation-row-actions">
          <button class="conversation-rename-btn icon-only-btn" type="button" title="Rename conversation" aria-label="Rename conversation">✎</button>
          <button class="conversation-delete-btn icon-only-btn" type="button" title="Delete conversation" aria-label="Delete conversation">🗑</button>
        </div>
      `;

      row.onclick = async () => {
        if (row.dataset.renaming === "1") return;
        await activateConversation(conv);
      };

      const renameBtn = row.querySelector(".conversation-rename-btn");
      const deleteBtn = row.querySelector(".conversation-delete-btn");

      renameBtn.addEventListener("click", (event) => {
        event.stopPropagation();
        startRenameConversation(row, conv);
      });

      deleteBtn.addEventListener("click", (event) => {
        event.stopPropagation();
        inlineConfirm(deleteBtn, async () => {
          await deleteConversation(conv.id);
        });
      });

      section.appendChild(row);
    });

    els.conversationList.appendChild(section);
  });
}

export async function loadPersistedConversations() {
  if (els.conversationSearch) {
    els.conversationSearch.value = appState.conversationSearchQuery;
  }
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
  const prov    = getActiveProvider();
  const created = await bridge.callService("CreateConversation", {
    title:       "New chat",
    provider_id: prov ? prov.id : 0,
    model:       els.chatModel.value,
  });
  if (!created || typeof created.id !== "number") {
    els.status.textContent = "create conversation failed";
    return;
  }
  const conv = {
    id:           created.id,
    title:        created.title || `Conversation ${appState.conversations.length + 1}`,
    providerName: prov ? prov.name : "",
    providerId:   prov ? prov.id : 0,
    model:        created.model || els.chatModel.value,
    updatedAt:    created.updated_at || new Date().toISOString(),
    messages:     [],
  };
  appState.conversations.unshift(conv);
  appState.activeConversationId = conv.id;
  syncChatModelSelector(conv.model || "");
  renderConversationList();
  renderMessages();
  els.prompt.focus();
}
