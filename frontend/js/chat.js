/*
  chat.js -- Chat streaming, message rendering, and message actions.
  Responsibilities: appendAssistantChunk, stopStreaming, cancelGeneration,
  sendPrompt (PROV-04: uses provider_id not api_key), message CRUD actions,
  and renderMessages with Markdown/LaTeX/Prism post-processing.
*/

import { appState, els } from "./state.js";
import { bridge }        from "./bridge.js";
import { renderMarkdown, applyEnhancers } from "./markdown.js";
import { getActiveProvider, rememberLastUsedModel } from "./providers.js";
import { t }             from "./i18n.js";

// ── Utilities (chat-scoped) ────────────────────────────────────────────────
export function activeConversation() {
  return appState.conversations.find((c) => c.id === appState.activeConversationId);
}

// ── Message rendering ──────────────────────────────────────────────────────
function messageActions(index) {
  return `
    <div class="actions">
      <button class="action-btn icon-only-btn" onclick="window.liaotao.copyMessage(${index})" title="Copy message" aria-label="Copy message">⧉</button>
      <button class="action-btn icon-only-btn" onclick="window.liaotao.editMessage(${index})" title="Edit message" aria-label="Edit message">✎</button>
      <button class="action-btn icon-only-btn" onclick="window.liaotao.regenerateMessage(${index})" title="Regenerate response" aria-label="Regenerate response">↺</button>
      <button class="action-btn icon-only-btn" onclick="window.liaotao.deleteMessage(${index})" title="Delete message" aria-label="Delete message">🗑</button>
    </div>
  `;
}

function formatDuration(durationMS) {
  if (!Number.isFinite(durationMS) || durationMS <= 0) return "";
  const sec = durationMS / 1000;
  if (sec < 10) return `${sec.toFixed(1)}s`;
  return `${Math.round(sec)}s`;
}

function thinkingIndicator(m) {
  if (!m.thinking) return "";
  return `<div class="thinking-indicator" aria-live="polite">
    <span class="thinking-dot"></span>
    <span class="thinking-dot"></span>
    <span class="thinking-dot"></span>
    <span class="thinking-text">${t("chat.thinking")}</span>
  </div>`;
}

// toolCallsBlock renders inline tool call indicators and collapsible results (MCP-08/09).
function toolCallsBlock(m) {
  if (!m.toolCalls || m.toolCalls.length === 0) return "";
  return m.toolCalls.map((tc) => {
    const safeName = tc.name.replace(/</g, "&lt;").replace(/>/g, "&gt;");
    if (tc.status === "calling") {
      return `<div class="tool-call calling">⚙ <strong>${safeName}</strong>…</div>`;
    }
    const safeResult = (tc.result || "").replace(/</g, "&lt;").replace(/>/g, "&gt;");
    return `<div class="tool-call done">⚙ <strong>${safeName}</strong></div>
      <details class="tool-result">
        <summary>${t("chat.tool_result")}</summary>
        <pre class="tool-result-content">${safeResult}</pre>
      </details>`;
  }).join("");
}
// Returns empty string when meta is absent or the global toggle is OFF.
function metaFooter(m) {
  if (!m.meta || !appState.settings.showMetaFooter) return "";
  const { provider_name, model, tokens_used, tokens_remaining, duration_ms } = m.meta;
  let text = `${provider_name} · ${model} · ~${tokens_used} tokens`;
  const durationText = formatDuration(duration_ms);
  if (durationText) text += ` · ${durationText}`;
  if (tokens_remaining > 0) text += ` · ${tokens_remaining} remaining`;
  return `<footer class="msg-meta">${text}</footer>`;
}

function tokenStatsFooter(m) {
  if (!m.tokenStats) return "";
  const parts = [];
  if (m.tokenStats.tokens_in > 0) parts.push(`~${m.tokenStats.tokens_in} in`);
  if (m.tokenStats.tokens_out > 0) parts.push(`~${m.tokenStats.tokens_out} out`);
  if (m.tokenStats.duration_ms > 0) parts.push(formatDuration(m.tokenStats.duration_ms));
  if (m.tokenStats.tokens_per_second > 0) parts.push(`${m.tokenStats.tokens_per_second.toFixed(1)} tok/s`);
  if (parts.length === 0) return "";
  const estimatedTag = m.tokenStats.estimated ? " <span class=\"msg-stats-estimated\">estimated</span>" : "";
  return `<footer class="msg-stats">${parts.join(" · ")}${estimatedTag}</footer>`;
}

export function renderMessages() {
  const conv = activeConversation();
  if (!conv) { els.messages.innerHTML = ""; return; }
  els.messages.innerHTML = conv.messages.map((m, idx) => `
    <article class="bubble ${m.role}">
      <div class="markdown">${renderMarkdown(m.content)}</div>
      ${m.role === "assistant" ? thinkingIndicator(m) : ""}
      ${m.role === "assistant" ? toolCallsBlock(m) : ""}
      ${tokenStatsFooter(m)}
      ${m.role === "assistant" ? metaFooter(m) : ""}
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
             .map((m) => ({ id: Number(m.id) || 0, role: m.role, content: m.content || "", tokenStats: m.token_stats || null }))
    : [];
  renderMessages();
}

// ── Streaming helpers ──────────────────────────────────────────────────────
export function appendAssistantChunk(content) {
  const conv = activeConversation();
  if (!conv) return;
  const last = conv.messages[conv.messages.length - 1];
  if (!last || last.role !== "assistant") {
    conv.messages.push({ role: "assistant", content: "", thinking: true, startedAt: Date.now() });
  }
  const msg = conv.messages[conv.messages.length - 1];
  msg.thinking = false;
  msg.content += content;
  renderMessages();
}

export function stopStreaming(reason) {
  appState.isStreaming = false;
  if (appState.streamingTimer) {
    clearInterval(appState.streamingTimer);
    appState.streamingTimer = null;
  }
  const conv = activeConversation();
  if (conv && conv.messages.length > 0) {
    const last = conv.messages[conv.messages.length - 1];
    if (last.role === "assistant") {
      last.thinking = false;
    }
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
  if (!conv || !text || appState.isStreaming) return;

  const activeProv = getActiveProvider();
  const prov = activeProv || appState.providers.find((p) => p.id === conv.providerId) || null;
  appState.isStreaming = true;
  appState.lastUserPrompt = text;
  conv.model        = appState.activeProviderId === 0 && typeof els.chatModel.value === "string" && els.chatModel.value.includes("::")
    ? (els.chatModel.value.split("::")[1] || "")
    : els.chatModel.value;
  conv.providerName = prov ? prov.name : conv.providerName;
  rememberLastUsedModel(conv.providerId, conv.providerName, conv.model);
  conv.messages.push({
    role: "user",
    content: text,
    tokenStats: {
      tokens_in: Math.max(1, Math.floor(text.trim().length / 4)),
      estimated: true,
    },
  });
  conv.messages.push({ role: "assistant", content: "", thinking: true, startedAt: Date.now() });
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
    provider_id:     Number(conv.providerId) > 0 ? Number(conv.providerId) : (prov ? prov.id : 0),
    model:           conv.model,
    prompt:          text,
    stream:          true,
    temperature:     Number(conv.temperature) > 0 ? Number(conv.temperature) : (prov ? prov.temperature : 0.7),
    max_tokens:      Math.max(0, Number(conv.maxTokens) || 0),
    system_prompt:   String(conv.systemPrompt || appState.settings.defaultSystemPrompt || ""),
    num_ctx:         prov ? prov.num_ctx     : 1024,
  });

  if (!sendResult || sendResult.ok === false) {
    startFallbackStream();
  }
}

// attachResponseMeta is called from app.js on chat:meta events (ROUTER-08).
// Attaches provider/model/token metadata to the last assistant message and re-renders.
export function attachResponseMeta(meta) {
  const conv = activeConversation();
  if (!conv || conv.messages.length === 0) return;
  const last = conv.messages[conv.messages.length - 1];
  if (last.role === "assistant") {
    const enrichedMeta = { ...meta };
    if ((!enrichedMeta.duration_ms || enrichedMeta.duration_ms <= 0) && Number.isFinite(last.startedAt)) {
      enrichedMeta.duration_ms = Math.max(1, Date.now() - last.startedAt);
    }
    last.meta = enrichedMeta;
    last.tokenStats = {
      tokens_out: enrichedMeta.tokens_out || Math.max(1, Math.floor((last.content || "").trim().length / 4)),
      duration_ms: enrichedMeta.duration_ms || 0,
      tokens_per_second: enrichedMeta.duration_ms > 0
        ? ((enrichedMeta.tokens_out || Math.max(1, Math.floor((last.content || "").trim().length / 4))) / (enrichedMeta.duration_ms / 1000))
        : 0,
      estimated: true,
    };
    last.thinking = false;
    renderMessages();
  }
}

// appendToolCall inserts an inline "calling: tool_name…" indicator in the last assistant bubble (MCP-08).
export function appendToolCall(toolName) {
  const conv = activeConversation();
  if (!conv) return;
  const last = conv.messages[conv.messages.length - 1];
  if (!last || last.role !== "assistant") {
    conv.messages.push({ role: "assistant", content: "", toolCalls: [] });
  }
  const msg = conv.messages[conv.messages.length - 1];
  if (!msg.toolCalls) msg.toolCalls = [];
  msg.toolCalls.push({ name: toolName, status: "calling", result: null });
  renderMessages();
}

// updateToolResult replaces the "calling" indicator with a collapsible result block (MCP-09).
export function updateToolResult(toolCallId, content) {
  const conv = activeConversation();
  if (!conv || conv.messages.length === 0) return;
  const last = conv.messages[conv.messages.length - 1];
  if (!last || !last.toolCalls) return;
  const tc = last.toolCalls.find((c) => c.id === toolCallId || c.status === "calling");
  if (tc) {
    tc.id = toolCallId;
    tc.status = "done";
    tc.result = content;
    renderMessages();
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

export async function deleteMessage(index) {
  const conv = activeConversation();
  if (!conv) return;
  const msg = conv.messages[index];
  if (!msg) return;

  if (Number(msg.id) > 0) {
    try {
      const res = await bridge.callService("DeleteMessage", {
        conversation_id: conv.id,
        message_id: Number(msg.id),
      });
      if (!res || res.ok !== true) {
        els.status.textContent = "delete failed";
        return;
      }
    } catch (err) {
      els.status.textContent = `delete failed: ${String(err && err.message ? err.message : err)}`;
      return;
    }
  }

  conv.messages.splice(index, 1);
  renderMessages();
}
