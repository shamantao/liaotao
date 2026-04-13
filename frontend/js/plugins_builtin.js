/*
  plugins_builtin.js -- Built-in plugins for PLUG-06/07/08.
  Responsibilities: register bundled plugins (TTS, prompt library, export) and
  expose helper functions used by the plugin manager UI.
*/

import { appState, els } from "./state.js";
import { registerPlugin } from "./plugins.js";

const PROMPT_LIBRARY_KEY = "liaotao.plugins.promptLibrary.v1";

function loadPromptLibrary() {
  try {
    const raw = localStorage.getItem(PROMPT_LIBRARY_KEY);
    if (!raw) {
      return {
        summarize: "Summarize the latest answer in 5 concise bullet points.",
        translate_fr: "Translate the latest answer into French.",
      };
    }
    const parsed = JSON.parse(raw);
    return parsed && typeof parsed === "object" ? parsed : {};
  } catch {
    return {};
  }
}

function savePromptLibrary(lib) {
  localStorage.setItem(PROMPT_LIBRARY_KEY, JSON.stringify(lib));
}

export function listPromptTemplates() {
  return loadPromptLibrary();
}

export function upsertPromptTemplate(name, content) {
  const key = String(name || "").trim();
  const value = String(content || "").trim();
  if (!key || !value) return false;
  const lib = loadPromptLibrary();
  lib[key] = value;
  savePromptLibrary(lib);
  return true;
}

export function deletePromptTemplate(name) {
  const key = String(name || "").trim();
  if (!key) return false;
  const lib = loadPromptLibrary();
  if (!Object.prototype.hasOwnProperty.call(lib, key)) return false;
  delete lib[key];
  savePromptLibrary(lib);
  return true;
}

export function insertPromptTemplate(name) {
  const lib = loadPromptLibrary();
  const value = lib[String(name || "").trim()];
  if (!value || !els.prompt) return false;
  const spacer = els.prompt.value && !els.prompt.value.endsWith("\n") ? "\n" : "";
  els.prompt.value = `${els.prompt.value}${spacer}${value}`;
  els.prompt.focus();
  return true;
}

function getActiveConversation() {
  return appState.conversations.find((c) => c.id === appState.activeConversationId) || null;
}

function toMarkdownConversation(conv) {
  if (!conv) return "";
  const lines = [`# ${conv.title || "Conversation"}`, ""];
  for (const msg of conv.messages || []) {
    const role = msg.role === "assistant" ? "Assistant" : "User";
    lines.push(`## ${role}`);
    lines.push("");
    lines.push(String(msg.content || ""));
    lines.push("");
  }
  return lines.join("\n");
}

export function exportCurrentConversationMarkdown() {
  const conv = getActiveConversation();
  if (!conv) return false;
  const markdown = toMarkdownConversation(conv);
  const blob = new Blob([markdown], { type: "text/markdown;charset=utf-8" });
  const url = URL.createObjectURL(blob);
  const a = document.createElement("a");
  a.href = url;
  a.download = `liaotao-conversation-${conv.id}.md`;
  document.body.appendChild(a);
  a.click();
  a.remove();
  URL.revokeObjectURL(url);
  return true;
}

export function exportCurrentConversationPDF() {
  const conv = getActiveConversation();
  if (!conv) return false;
  const markdown = toMarkdownConversation(conv);
  const popup = window.open("", "_blank", "width=900,height=700");
  if (!popup) return false;

  popup.document.write(`<!doctype html><html><head><title>${conv.title || "Conversation"}</title><style>
    body { font-family: Georgia, 'Times New Roman', serif; padding: 24px; line-height: 1.5; }
    pre { white-space: pre-wrap; }
  </style></head><body><pre>${markdown.replace(/</g, "&lt;").replace(/>/g, "&gt;")}</pre></body></html>`);
  popup.document.close();
  popup.focus();
  popup.print();
  return true;
}

export function registerBuiltInPlugins() {
  registerPlugin({
    id: "builtin-tts",
    name: "TTS",
    description: "Text-to-speech for assistant replies",
    source: "builtin",
    enabled: false,
    hooks: {
      afterReceive(payload) {
        if (!payload || typeof payload.content !== "string") return payload;
        if (!window.speechSynthesis) return payload;
        try {
          const utter = new SpeechSynthesisUtterance(payload.content.slice(0, 800));
          window.speechSynthesis.cancel();
          window.speechSynthesis.speak(utter);
        } catch {
          // Ignore speech errors to avoid blocking chat flow.
        }
        return payload;
      },
    },
  });

  registerPlugin({
    id: "builtin-prompt-library",
    name: "Prompt Library",
    description: "Reusable prompt templates via /tpl <name>",
    source: "builtin",
    enabled: true,
    hooks: {
      beforeSend(payload) {
        if (!payload || typeof payload.prompt !== "string") return payload;
        const prompt = payload.prompt.trim();
        if (!prompt.startsWith("/tpl ")) return payload;
        const name = prompt.slice(5).trim();
        const lib = loadPromptLibrary();
        const tpl = lib[name];
        if (!tpl) {
          return { ...payload, cancel: true };
        }
        return { ...payload, prompt: tpl };
      },
    },
  });

  registerPlugin({
    id: "builtin-export",
    name: "Conversation Export",
    description: "Export current conversation to Markdown or PDF",
    source: "builtin",
    enabled: true,
    hooks: {
      onSaveConv(payload) {
        return payload;
      },
    },
  });
}
