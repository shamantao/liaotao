/*
  conversations.js -- Conversation sidebar management and persistence.
  Responsibilities: render conversation list, load persisted conversations from DB,
  create new conversation.
*/

import { appState, els, persistSettingsToStorage } from "./state.js";
import { bridge }                  from "./bridge.js";
import { getActiveProvider, syncChatModelSelector } from "./providers.js";
import { renderMessages, loadConversationMessages } from "./chat.js";

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
    messages: [],
  };
}

async function reloadConversationList(preferredConversationID = 0) {
  const result = await bridge.callService("ListConversations", { limit: 100 });
  if (!Array.isArray(result) || result.length === 0) {
    appState.conversations = [];
    appState.activeConversationId = null;
    renderConversationList();
    renderMessages();
    return false;
  }

  appState.conversations = result.map(mapConversationSummary);
  const preferred = appState.conversations.find((conv) => conv.id === preferredConversationID);
  const nextActive = preferred || appState.conversations[0];
  await activateConversation(nextActive);
  return true;
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
  appState.conversations.forEach((conv) => {
    const row = document.createElement("div");
    row.className = `conversation-item${conv.id === appState.activeConversationId ? " active" : ""}`;
    row.innerHTML = `
      <span class="dot">${conv.title.slice(0, 1).toUpperCase()}</span>
      <span class="label">${conv.title}</span>
      <button class="conversation-delete-btn icon-only-btn" type="button" title="Delete conversation" aria-label="Delete conversation">🗑</button>
    `;
    row.onclick = async () => activateConversation(conv);
    const deleteBtn = row.querySelector(".conversation-delete-btn");
    deleteBtn.addEventListener("click", (event) => {
      event.stopPropagation();
      inlineConfirm(deleteBtn, async () => {
        await deleteConversation(conv.id);
      });
    });
    els.conversationList.appendChild(row);
  });
}

export async function loadPersistedConversations() {
  const loaded = await reloadConversationList(appState.activeConversationId || 0);
  if (!loaded) {
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
    messages:     [],
  };
  appState.conversations.unshift(conv);
  appState.activeConversationId = conv.id;
  syncChatModelSelector(conv.model || "");
  renderConversationList();
  renderMessages();
  els.prompt.focus();
}
