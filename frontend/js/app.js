/*
  app.js -- Liaotao MVP frontend runtime.
  Responsibilities: tabs, sidebar behavior, chat rendering, streaming, markdown, Prism, KaTeX, Wails event hooks.
*/

(function bootstrap() {
  const appState = {
    conversations: [],
    activeConversationId: null,
    streamingTimer: null,
    lastUserPrompt: "",
    sidebarCollapsed: false,
    sidebarWidth: 290,
    expandedSidebarWidth: 290,
  };

  const els = {
    appShell: document.getElementById("app-shell"),
    tabs: document.querySelectorAll(".tab-btn"),
    panels: document.querySelectorAll(".tab-panel"),
    status: document.getElementById("status"),
    conversationList: document.getElementById("conversation-list"),
    messages: document.getElementById("messages"),
    prompt: document.getElementById("prompt"),
    send: document.getElementById("send-btn"),
    stop: document.getElementById("stop-btn"),
    newChat: document.getElementById("new-chat-btn"),
    chatModel: document.getElementById("chat-model"),
    sidebarToggle: document.getElementById("sidebar-toggle"),
    sidebarResizer: document.getElementById("sidebar-resizer"),
  };

  const bridge = {
    eventsOn(name, cb) {
      if (window.runtime && typeof window.runtime.EventsOn === "function") {
        window.runtime.EventsOn(name, cb);
        return true;
      }
      document.addEventListener(`liaotao:${name}`, (event) => cb(event.detail));
      return false;
    },

    eventsEmit(name, payload) {
      if (window.runtime && typeof window.runtime.EventsEmit === "function") {
        window.runtime.EventsEmit(name, payload);
        return;
      }
      document.dispatchEvent(new CustomEvent(`liaotao:${name}`, { detail: payload }));
    },

    async callService(method, payload) {
      const goRoot = window.go;
      if (!goRoot || typeof goRoot !== "object") {
        return { ok: false, reason: "no-wails-binding" };
      }

      for (const namespace of Object.values(goRoot)) {
        if (!namespace || typeof namespace !== "object") {
          continue;
        }
        for (const value of Object.values(namespace)) {
          if (!value || typeof value !== "object") {
            continue;
          }
          if (typeof value[method] === "function") {
            return value[method](payload);
          }
        }
      }

      return { ok: false, reason: "method-not-found" };
    },
  };

  function uid() {
    return `${Date.now()}_${Math.random().toString(36).slice(2, 8)}`;
  }

  function switchTab(tab) {
    els.tabs.forEach((btn) => btn.classList.toggle("active", btn.dataset.tab === tab));
    els.panels.forEach((panel) => panel.classList.toggle("active", panel.id === tab));
  }

  function applySidebarState() {
    const effectiveWidth = appState.sidebarCollapsed ? 72 : appState.expandedSidebarWidth;
    els.appShell.style.setProperty("--sidebar-width", `${effectiveWidth}px`);
    els.appShell.classList.toggle("sidebar-collapsed", appState.sidebarCollapsed);
  }

  function toggleSidebar() {
    if (!appState.sidebarCollapsed) {
      appState.expandedSidebarWidth = Math.max(180, appState.sidebarWidth);
      appState.sidebarCollapsed = true;
    } else {
      appState.sidebarCollapsed = false;
      appState.sidebarWidth = appState.expandedSidebarWidth;
    }
    applySidebarState();
  }

  function initSidebarResizer() {
    let drag = false;
    els.sidebarResizer.addEventListener("mousedown", () => {
      drag = true;
      if (appState.sidebarCollapsed) {
        appState.sidebarCollapsed = false;
        appState.sidebarWidth = appState.expandedSidebarWidth;
      }
      applySidebarState();
    });

    window.addEventListener("mousemove", (event) => {
      if (!drag) return;
      const next = Math.max(72, Math.min(460, event.clientX));
      appState.sidebarWidth = next;
      appState.expandedSidebarWidth = Math.max(180, next);
      applySidebarState();
    });

    window.addEventListener("mouseup", () => {
      drag = false;
    });
  }

  function activeConversation() {
    return appState.conversations.find((c) => c.id === appState.activeConversationId);
  }

  function newConversation() {
    const conv = {
      id: uid(),
      title: `Conversation ${appState.conversations.length + 1}`,
      model: els.chatModel.value,
      messages: [],
    };
    appState.conversations.unshift(conv);
    appState.activeConversationId = conv.id;
    renderConversationList();
    renderMessages();
    els.prompt.focus();
  }

  function renderConversationList() {
    els.conversationList.innerHTML = "";
    appState.conversations.forEach((conv) => {
      const row = document.createElement("div");
      row.className = `conversation-item${conv.id === appState.activeConversationId ? " active" : ""}`;
      row.innerHTML = `<span class="dot">${conv.title.slice(0, 1).toUpperCase()}</span><span class="label">${conv.title}</span>`;
      row.onclick = () => {
        appState.activeConversationId = conv.id;
        els.chatModel.value = conv.model || els.chatModel.value;
        renderConversationList();
        renderMessages();
      };
      els.conversationList.appendChild(row);
    });
  }

  function escapeHtml(text) {
    return text
      .replaceAll("&", "&amp;")
      .replaceAll("<", "&lt;")
      .replaceAll(">", "&gt;")
      .replaceAll('"', "&quot;")
      .replaceAll("'", "&#39;");
  }

  function renderInlineMarkdown(text) {
    let html = escapeHtml(text);
    html = html.replace(/\*\*(.*?)\*\*/g, "<strong>$1</strong>");
    html = html.replace(/\*(.*?)\*/g, "<em>$1</em>");
    html = html.replace(/`([^`]+)`/g, "<code>$1</code>");
    html = html.replace(/\[(.*?)\]\((https?:\/\/[^\s)]+)\)/g, '<a href="$2" target="_blank" rel="noreferrer">$1</a>');
    html = html.replace(/&lt;think&gt;([\s\S]*?)&lt;\/think&gt;/g, '<details class="think"><summary>Reasoning</summary><div>$1</div></details>');
    return html;
  }

  function parseTableBlock(lines) {
    if (lines.length < 2) return "";
    const header = lines[0].split("|").map((s) => s.trim()).filter(Boolean);
    const separator = lines[1].split("|").map((s) => s.trim()).filter(Boolean);
    if (!header.length || separator.length !== header.length || !separator.every((s) => /^-+:?$|^:-+:$|^:?-+$/.test(s))) {
      return "";
    }

    const body = lines.slice(2).map((row) => row.split("|").map((s) => s.trim()).filter(Boolean));
    const thead = `<thead><tr>${header.map((h) => `<th>${renderInlineMarkdown(h)}</th>`).join("")}</tr></thead>`;
    const tbody = `<tbody>${body.map((cols) => `<tr>${header.map((_, i) => `<td>${renderInlineMarkdown(cols[i] || "")}</td>`).join("")}</tr>`).join("")}</tbody>`;
    return `<table>${thead}${tbody}</table>`;
  }

  function renderMarkdown(raw) {
    let text = raw || "";
    const codeBlocks = [];

    text = text.replace(/```([\w-]*)\n([\s\S]*?)```/g, (_m, lang, code) => {
      const idx = codeBlocks.length;
      codeBlocks.push({ lang: lang || "text", code: escapeHtml(code) });
      return `__CODE_BLOCK_${idx}__`;
    });

    const lines = text.split("\n");
    const html = [];
    let i = 0;

    while (i < lines.length) {
      const line = lines[i];

      if (/^\s*$/.test(line)) {
        html.push("<br>");
        i += 1;
        continue;
      }

      if (line.includes("|") && i + 1 < lines.length && lines[i + 1].includes("|")) {
        const tableLines = [line, lines[i + 1]];
        let j = i + 2;
        while (j < lines.length && lines[j].includes("|")) {
          tableLines.push(lines[j]);
          j += 1;
        }
        const tableHTML = parseTableBlock(tableLines);
        if (tableHTML) {
          html.push(tableHTML);
          i = j;
          continue;
        }
      }

      const heading = line.match(/^(#{1,3})\s+(.*)$/);
      if (heading) {
        const lvl = heading[1].length;
        html.push(`<h${lvl}>${renderInlineMarkdown(heading[2])}</h${lvl}>`);
        i += 1;
        continue;
      }

      if (/^>\s+/.test(line)) {
        html.push(`<blockquote>${renderInlineMarkdown(line.replace(/^>\s+/, ""))}</blockquote>`);
        i += 1;
        continue;
      }

      if (/^\d+\.\s+/.test(line)) {
        const items = [];
        let j = i;
        while (j < lines.length && /^\d+\.\s+/.test(lines[j])) {
          items.push(lines[j].replace(/^\d+\.\s+/, ""));
          j += 1;
        }
        html.push(`<ol>${items.map((item) => `<li>${renderInlineMarkdown(item)}</li>`).join("")}</ol>`);
        i = j;
        continue;
      }

      if (/^[-*]\s+/.test(line)) {
        const items = [];
        let j = i;
        while (j < lines.length && /^[-*]\s+/.test(lines[j])) {
          items.push(lines[j].replace(/^[-*]\s+/, ""));
          j += 1;
        }
        html.push(`<ul>${items.map((item) => `<li>${renderInlineMarkdown(item)}</li>`).join("")}</ul>`);
        i = j;
        continue;
      }

      html.push(`<p>${renderInlineMarkdown(line)}</p>`);
      i += 1;
    }

    let merged = html.join("\n");
    merged = merged.replace(/__CODE_BLOCK_(\d+)__/g, (_m, idxStr) => {
      const idx = Number(idxStr);
      const block = codeBlocks[idx];
      return `<pre><code class="language-${block.lang}">${block.code}</code></pre>`;
    });

    return merged;
  }

  function applyEnhancers(container) {
    if (window.Prism && typeof window.Prism.highlightAllUnder === "function") {
      window.Prism.highlightAllUnder(container);
    }

    if (window.renderMathInElement) {
      window.renderMathInElement(container, {
        delimiters: [
          { left: "$$", right: "$$", display: true },
          { left: "$", right: "$", display: false },
        ],
        throwOnError: false,
      });
      return;
    }

    // Fallback renderer when KaTeX is not loaded yet.
    container.querySelectorAll("p,li,blockquote").forEach((node) => {
      node.innerHTML = node.innerHTML
        .replace(/\$\$([\s\S]+?)\$\$/g, '<span class="math-block">$1</span>')
        .replace(/\$(.+?)\$/g, '<span class="math-inline">$1</span>');
    });
  }

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

  function renderMessages() {
    const conv = activeConversation();
    if (!conv) {
      els.messages.innerHTML = "";
      return;
    }

    els.messages.innerHTML = conv.messages.map((m, idx) => `
      <article class="bubble ${m.role}">
        <div class="markdown">${renderMarkdown(m.content)}</div>
        ${messageActions(idx)}
      </article>
    `).join("");

    applyEnhancers(els.messages);
    els.messages.scrollTop = els.messages.scrollHeight;
  }

  function appendAssistantChunk(content) {
    const conv = activeConversation();
    if (!conv) return;

    const last = conv.messages[conv.messages.length - 1];
    if (!last || last.role !== "assistant") {
      conv.messages.push({ role: "assistant", content: "" });
    }
    conv.messages[conv.messages.length - 1].content += content;
    renderMessages();
  }

  function stopStreaming(reason) {
    if (appState.streamingTimer) {
      clearInterval(appState.streamingTimer);
      appState.streamingTimer = null;
    }
    els.stop.style.display = "none";
    els.status.textContent = reason === "cancel" ? "stopped" : "ready";
  }

  async function cancelGeneration() {
    stopStreaming("cancel");
    await bridge.callService("CancelGeneration", { conversation_id: appState.activeConversationId });
    bridge.eventsEmit("chat:stop", { conversation_id: appState.activeConversationId });
  }

  function startFallbackStream() {
    const generated = [
      `### ${els.chatModel.value}`,
      "",
      `You asked: **${appState.lastUserPrompt}**.`,
      "",
      "- Streaming through events",
      "- Markdown with list + table",
      "",
      "| Item | Status |",
      "| --- | --- |",
      "| UI-04 | done |",
      "| UI-08 | done |",
      "",
      "```go",
      "fmt.Println(\"liaotao streaming ready\")",
      "```",
      "",
      "$a^2 + b^2 = c^2$",
      "<think>Fallback mode is active until backend event stream is connected.</think>",
    ].join(" ");

    const chunks = generated.split(" ");
    let i = 0;

    appState.streamingTimer = setInterval(() => {
      if (i >= chunks.length) {
        stopStreaming("done");
        return;
      }
      bridge.eventsEmit("chat:chunk", { content: `${chunks[i]} `, done: false });
      i += 1;
    }, 60);
  }

  async function sendPrompt() {
    const conv = activeConversation();
    const text = els.prompt.value.trim();
    if (!conv || !text || appState.streamingTimer) return;

    appState.lastUserPrompt = text;
    conv.model = els.chatModel.value;
    conv.messages.push({ role: "user", content: text });
    conv.messages.push({ role: "assistant", content: "" });
    els.prompt.value = "";
    renderMessages();

    els.stop.style.display = "inline-block";
    els.status.textContent = "streaming";

    await bridge.callService("SaveMessage", {
      conversation_id: conv.id,
      role: "user",
      content: text,
    });

    const sendResult = await bridge.callService("SendMessage", {
      conversation_id: conv.id,
      model: conv.model,
      prompt: text,
      stream: true,
    });

    if (!sendResult || sendResult.ok === false) {
      startFallbackStream();
    }
  }

  function copyMessage(index) {
    const conv = activeConversation();
    const msg = conv && conv.messages[index];
    if (!msg) return;

    navigator.clipboard.writeText(msg.content);
    els.status.textContent = "copied";
    setTimeout(() => {
      els.status.textContent = "ready";
    }, 800);
  }

  function editMessage(index) {
    const conv = activeConversation();
    const msg = conv && conv.messages[index];
    if (!msg || msg.role !== "user") return;

    els.prompt.value = msg.content;
    conv.messages.splice(index, 1);
    renderMessages();
    els.prompt.focus();
  }

  function regenerateMessage(index) {
    const conv = activeConversation();
    const msg = conv && conv.messages[index];
    if (!msg || msg.role !== "assistant" || appState.streamingTimer) return;

    conv.messages.splice(index, 1);
    renderMessages();
    sendPrompt();
  }

  function deleteMessage(index) {
    const conv = activeConversation();
    if (!conv) return;

    conv.messages.splice(index, 1);
    renderMessages();
  }

  function bindEvents() {
    els.tabs.forEach((btn) => btn.addEventListener("click", () => switchTab(btn.dataset.tab)));
    els.newChat.addEventListener("click", newConversation);
    els.send.addEventListener("click", sendPrompt);
    els.stop.addEventListener("click", cancelGeneration);
    els.sidebarToggle.addEventListener("click", toggleSidebar);

    els.chatModel.addEventListener("change", () => {
      const conv = activeConversation();
      if (conv) {
        conv.model = els.chatModel.value;
      }
    });

    els.prompt.addEventListener("keydown", (event) => {
      if (event.key === "Enter" && !event.shiftKey) {
        event.preventDefault();
        sendPrompt();
      }
    });

    bridge.eventsOn("chat:chunk", (chunk) => {
      if (!chunk || typeof chunk.content !== "string") return;
      appendAssistantChunk(chunk.content);
      if (chunk.done) {
        stopStreaming("done");
      }
    });

    bridge.eventsOn("chat:done", () => stopStreaming("done"));
    bridge.eventsOn("chat:error", () => stopStreaming("cancel"));
  }

  function init() {
    window.liaotao = {
      copyMessage,
      editMessage,
      regenerateMessage,
      deleteMessage,
    };

    initSidebarResizer();
    applySidebarState();
    bindEvents();
    newConversation();
  }

  init();
})();
