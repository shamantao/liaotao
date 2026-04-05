/*
  conversations.js -- Conversation sidebar management and persistence.
  Responsibilities: render conversation list, load persisted conversations from DB,
  create new conversation.
*/

import { appState, els, persistSettingsToStorage } from "./state.js";
import { bridge }                  from "./bridge.js";
import { getActiveProvider, loadProviders } from "./providers.js";
import { renderMessages, loadConversationMessages } from "./chat.js";

// ── Conversation sidebar ───────────────────────────────────────────────────
export function renderConversationList() {
  els.conversationList.innerHTML = "";
  appState.conversations.forEach((conv) => {
    const row = document.createElement("div");
    row.className = `conversation-item${conv.id === appState.activeConversationId ? " active" : ""}`;
    row.innerHTML = `
      <span class="dot">${conv.title.slice(0, 1).toUpperCase()}</span>
      <span class="label">${conv.title}</span>
    `;
    row.onclick = async () => {
      appState.activeConversationId = conv.id;
      if (conv.providerName) {
        const prov = appState.providers.find((p) => p.name === conv.providerName && p.active);
        if (prov) {
          appState.activeProviderId = prov.id;
          els.chatProvider.value    = String(prov.id);
          persistSettingsToStorage();
        }
      }
      if (conv.model) els.chatModel.value = conv.model;
      renderConversationList();
      await loadConversationMessages(conv.id);
    };
    els.conversationList.appendChild(row);
  });
}

export async function loadPersistedConversations() {
  const result = await bridge.callService("ListConversations", { limit: 100 });
  if (!Array.isArray(result) || result.length === 0) {
    await newConversation();
    return;
  }
  appState.conversations = result.map((item) => ({
    id:           item.id,
    title:        item.title || `Conversation ${item.id}`,
    providerName: item.provider || "",
    model:        item.model   || els.chatModel.value,
    messages:     [],
  }));
  appState.activeConversationId = appState.conversations[0].id;
  renderConversationList();
  await loadConversationMessages(appState.activeConversationId);
}

export async function newConversation() {
  const prov    = getActiveProvider();
  const created = await bridge.callService("CreateConversation", {
    title:       "New chat",
    provider_id: prov ? prov.name : "default",
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
    model:        created.model || els.chatModel.value,
    messages:     [],
  };
  appState.conversations.unshift(conv);
  appState.activeConversationId = conv.id;
  renderConversationList();
  renderMessages();
  els.prompt.focus();
}
