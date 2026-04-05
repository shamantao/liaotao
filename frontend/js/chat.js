/*
  chat.js -- Chat streaming, message rendering, and message actions.
  Responsibilities: appendAssistantChunk, stopStreaming, cancelGeneration,
  sendPrompt (PROV-04: uses provider_id not api_key), message CRUD actions,
  and renderMessages with Markdown/LaTeX/Prism post-processing.
*/

import { appState, els } from "./state.js";
import { bridge }        from "./bridge.js";
import { renderMarkdown, applyEnhancers } from "./markdown.js";
import { getActiveProvider } from "./providers.js";

// ── Utilities (chat-scoped) ────────────────────────────────────────────────
export function activeConversation() {
  return appState.conversations.find((c) => c.id === appState.activeConversationId);
}

// ── Message rendering ──────────────────────────────────────────────────────
function messageActions(index) {
  return `
    <div class="actions">
      <button class="action-btn" onclick="window.liaotao.copyMessage(${index})">copy</button>
      <button class="action-btn" onclick="window.liaotao.editMessage(${index})">edit</button>
      <button class="action-btn" onclick="window.liaotao.regenerateMessage(${index})">regen</button>
      <button class="action-btn" onclick="window.liaotao.deleteMessage(${index})">delete</button>
    </div>
  `;
}

export function renderMessages() {
  const conv = activeConversation();
  if (!conv) { els.messages.innerHTML = ""; return; }
  els.messages.innerHTML = conv.messages.map((m, idx) => `
    <article class="bubble ${m.role}">
      <div class="markdown">${renderMarkdown(m.content)}</div>
      ${messageActions(idx)}
    </article>
  `).join("");
  applyEnhancers(els.messages);
  els.messages.scrollTop = els.messages.scrollHeight;
}

// ── Conversation message loading ───────────────────────────────────────────
export async function loadConversationMessages(conversationId) {
  const result = await bridge.callService("ListMessages", {
    conversation_id: conversationId,
    limit: 500,
  });
  const conv = appState.conversations.find((c) => c.id === conversationId);
  if (!conv) return;
  conv.messages = Array.isArray(result)
    ? result.filter((m) => m && typeof m.role === "string")
             .map((m) => ({ role: m.role, content: m.content || "" }))
    : [];
  renderMessages();
}

// ── Streaming helpers ──────────────────────────────────────────────────────
export function appendAssistantChunk(content) {
  const conv = activeConversation();
  if (!conv) return;
  const last = conv.messages[conv.messages.length - 1];
  if (!last || last.role !== "assistant") {
    conv.messages.push({ role: "assistant", content: "" });
  }
  conv.messages[conv.messages.length - 1].content += content;
  renderMessages();
}

export function stopStreaming(reason) {
  if (appState.streamingTimer) {
    clearInterval(appState.streamingTimer);
    appState.streamingTimer = null;
  }
  els.stop.style.display = "none";
  els.status.textContent = reason === "cancel" ? "stopped" : "ready";
}

export async function cancelGeneration() {
  stopStreaming("cancel");
  await bridge.callService("CancelGeneration", {
    conversation_id: String(appState.activeConversationId || ""),
  });
  bridge.eventsEmit("chat:stop", { conversation_id: String(appState.activeConversationId || "") });
}

function startFallbackStream() {
  const generated = [
    `### ${els.chatModel.value}`,
    "",
    `You asked: **${appState.lastUserPrompt}**.`,
    "",
    "Fallback mode active — no Wails binding available.",
  ].join(" ");

  const chunks = generated.split(" ");
  let i = 0;
  appState.streamingTimer = setInterval(() => {
    if (i >= chunks.length) { stopStreaming("done"); return; }
    bridge.eventsEmit("chat:chunk", { content: `${chunks[i]} `, done: false });
    i++;
  }, 60);
}

// ── Send prompt (PROV-04: uses provider_id, not api_key) ──────────────────
export async function sendPrompt() {
  const conv = activeConversation();
  const text = els.prompt.value.trim();
  if (!conv || !text || appState.streamingTimer) return;

  const prov = getActiveProvider();
  appState.lastUserPrompt = text;
  conv.model        = els.chatModel.value;
  conv.providerName = prov ? prov.name : conv.providerName;
  conv.messages.push({ role: "user",      content: text });
  conv.messages.push({ role: "assistant", content: "" });
  els.prompt.value = "";
  renderMessages();

  els.stop.style.display = "inline-block";
  els.status.textContent = "streaming";

  await bridge.callService("SaveMessage", {
    conversation_id: conv.id,
    role:    "user",
    content: text,
  });

  const sendResult = await bridge.callService("SendMessage", {
    conversation_id: String(conv.id),
    provider_id:     prov ? prov.id : 0,
    model:           conv.model,
    prompt:          text,
    stream:          true,
    temperature:     prov ? prov.temperature : 0.7,
    num_ctx:         prov ? prov.num_ctx     : 1024,
  });

  if (!sendResult || sendResult.ok === false) {
    startFallbackStream();
  }
}

// ── Message actions ────────────────────────────────────────────────────────
export function copyMessage(index) {
  const conv = activeConversation();
  const msg  = conv && conv.messages[index];
  if (!msg) return;
  navigator.clipboard.writeText(msg.content);
  els.status.textContent = "copied";
  setTimeout(() => { els.status.textContent = "ready"; }, 800);
}

export function editMessage(index) {
  const conv = activeConversation();
  const msg  = conv && conv.messages[index];
  if (!msg || msg.role !== "user") return;
  els.prompt.value = msg.content;
  conv.messages.splice(index, 1);
  renderMessages();
  els.prompt.focus();
}

export function regenerateMessage(index) {
  const conv = activeConversation();
  const msg  = conv && conv.messages[index];
  if (!msg || msg.role !== "assistant" || appState.streamingTimer) return;
  conv.messages.splice(index, 1);
  renderMessages();
  sendPrompt();
}

export function deleteMessage(index) {
  const conv = activeConversation();
  if (!conv) return;
  conv.messages.splice(index, 1);
  renderMessages();
}
