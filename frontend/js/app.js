/*
  app.js -- Liaotao MVP frontend runtime.
  Responsibilities: tabs, sidebar, chat rendering, streaming, markdown,
  provider CRUD (Settings), model selection, preset profiles, and DB persistence.
*/

(function bootstrap() {

  // ── App state ──────────────────────────────────────────────────────
  const appState = {
    conversations: [],
    activeConversationId: null,
    streamingTimer: null,
    lastUserPrompt: "",
    sidebarCollapsed: false,
    sidebarWidth: 290,
    expandedSidebarWidth: 290,
    providers: [],          // ProviderRecord[] cached from DB
    activeProviderId: null, // number | null — currently selected provider
    settingsSection: "general",
    settings: { language: "fr", theme: "dark" },
  };

  // ── Provider profiles cache (PROV-08) ──────────────────────────────
  let providerProfiles = []; // ProviderProfile[]

  // ── DOM refs ───────────────────────────────────────────────────────
  const els = {
    appShell:      document.getElementById("app-shell"),
    tabs:          document.querySelectorAll(".tab-btn"),
    panels:        document.querySelectorAll(".tab-panel"),
    status:        document.getElementById("status"),
    // chat
    conversationList: document.getElementById("conversation-list"),
    messages:      document.getElementById("messages"),
    prompt:        document.getElementById("prompt"),
    send:          document.getElementById("send-btn"),
    stop:          document.getElementById("stop-btn"),
    newChat:       document.getElementById("new-chat-btn"),
    chatProvider:  document.getElementById("chat-provider"),
    chatModel:     document.getElementById("chat-model"),
    refreshModels: document.getElementById("refresh-models-btn"),
    sidebarToggle: document.getElementById("sidebar-toggle"),
    sidebarResizer: document.getElementById("sidebar-resizer"),
    // settings nav
    settingsNavBtns:   document.querySelectorAll(".settings-nav-btn"),
    settingsSections:  document.querySelectorAll(".settings-section"),
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
    pfPreset:      document.getElementById("pf-preset"),
    pfDocsLink:    document.getElementById("pf-docs-link"),
    // PROV-05: test connection
    pfTestBtn:     document.getElementById("pf-test-btn"),
    pfTestResult:  document.getElementById("pf-test-result"),
  };

  // ── Wails v3 bridge ─────────────────────────────────────────────────
  // Wails v3 alpha exposes:
  //   window.wails.Call.ByName(fqn, ...args)  → call a Go service method
  //   window.wails.Events.On(name, cb)         → receive Go-emitted events (cb(e) where e.data is payload)
  //   window.wails.Events.Emit(name, data)     → send events to Go
  // Method FQN format: "{go-package-path}.{TypeName}.{MethodName}"
  const SERVICE_FQN = "liaotao/internal/bindings.Service.";

  const bridge = {
    eventsOn(name, cb) {
      if (window.wails?.Events?.On) {
        // Wails v3: event callback receives a WailsEvent object — unwrap .data for callers.
        window.wails.Events.On(name, (e) => cb(e && e.data !== undefined ? e.data : e));
        return true;
      }
      // DOM event fallback (offline / test mode without Wails runtime)
      document.addEventListener(`liaotao:${name}`, (e) => cb(e.detail));
      return false;
    },

    eventsEmit(name, payload) {
      if (window.wails?.Events?.Emit) {
        window.wails.Events.Emit(name, payload);
        return;
      }
      document.dispatchEvent(new CustomEvent(`liaotao:${name}`, { detail: payload }));
    },

    async callService(method, payload) {
      await waitForWailsRuntime();
      if (!window.wails?.Call?.ByName) {
        console.error("[liaotao] callService: no-wails-runtime — window.wails =", window.wails);
        throw new Error("no-wails-runtime");
      }
      const fqn = SERVICE_FQN + method;
      console.debug("[liaotao] →", fqn, payload);
      try {
        // Only pass payload as arg when it is provided — Go-side argument count must match.
        const result = payload !== undefined && payload !== null
          ? await window.wails.Call.ByName(fqn, payload)
          : await window.wails.Call.ByName(fqn);
        console.debug("[liaotao] ←", fqn, result);
        return result;
      } catch (err) {
        console.error("[liaotao] call failed:", fqn, err);
        throw err;
      }
    },
  };

  // waitForWailsRuntime polls until window.wails.Call.ByName is ready (injected by /wails/runtime.js).
  async function waitForWailsRuntime(timeoutMs = 4000) {
    const start = Date.now();
    while (Date.now() - start < timeoutMs) {
      if (window.wails?.Call?.ByName) {
        return;
      }
      await new Promise((resolve) => setTimeout(resolve, 60));
    }
    // Timeout: runtime.js not loaded — likely a Wails integration issue.
    console.error("[liaotao] waitForWailsRuntime: timeout. window.wails =", window.wails, "window._wails =", window._wails);
  }

  // ── Settings persistence (localStorage) ───────────────────────────
  const STORAGE_KEY = "liaotao.settings.v2";

  function loadSettingsFromStorage() {
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

  function persistSettingsToStorage() {
    localStorage.setItem(STORAGE_KEY, JSON.stringify({
      settings: appState.settings,
      activeProviderId: appState.activeProviderId,
    }));
  }

  function applySettingsToUI() {
    if (els.language) els.language.value = appState.settings.language || "fr";
    if (els.theme)    els.theme.value    = appState.settings.theme    || "dark";
  }

  // ── Settings navigation ────────────────────────────────────────────
  function switchSettingsSection(sectionId) {
    appState.settingsSection = sectionId;
    els.settingsNavBtns.forEach((btn) =>
      btn.classList.toggle("active", btn.dataset.section === sectionId));
    els.settingsSections.forEach((sec) =>
      sec.classList.toggle("active", sec.id === `section-${sectionId}`));
    if (sectionId === "providers") {
      loadProviders();
    }
  }

  // ── Tab switching ──────────────────────────────────────────────────
  function switchTab(tab) {
    els.tabs.forEach((btn) => btn.classList.toggle("active", btn.dataset.tab === tab));
    els.panels.forEach((panel) => panel.classList.toggle("active", panel.id === tab));
  }

  // ── Sidebar collapse / resize ──────────────────────────────────────
  function applySidebarState() {
    const w = appState.sidebarCollapsed ? 72 : appState.expandedSidebarWidth;
    els.appShell.style.setProperty("--sidebar-width", `${w}px`);
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
    window.addEventListener("mousemove", (e) => {
      if (!drag) return;
      const next = Math.max(72, Math.min(460, e.clientX));
      appState.sidebarWidth = next;
      appState.expandedSidebarWidth = Math.max(180, next);
      applySidebarState();
    });
    window.addEventListener("mouseup", () => { drag = false; });
  }

  // ── Provider helpers ───────────────────────────────────────────────
  function getActiveProvider() {
    if (appState.activeProviderId == null) return null;
    return appState.providers.find((p) => p.id === appState.activeProviderId) || null;
  }

  function updateChatProviderSelector() {
    const active = appState.providers.filter((p) => p.active);
    if (active.length === 0) {
      els.chatProvider.innerHTML = `<option value="">No providers – add one in Settings</option>`;
      appState.activeProviderId = null;
      persistSettingsToStorage();
      return;
    }
    els.chatProvider.innerHTML = active
      .map((p) => `<option value="${p.id}">${escapeHtml(p.name)}</option>`)
      .join("");

    const prevId = appState.activeProviderId;
    if (prevId != null && active.some((p) => p.id === prevId)) {
      els.chatProvider.value = String(prevId);
    } else {
      appState.activeProviderId = active[0].id;
      els.chatProvider.value   = String(active[0].id);
    }
    persistSettingsToStorage();
  }

  // ── PROV-08: Provider profiles (presets) ───────────────────────────
  async function loadProviderProfiles() {
    const result = await bridge.callService("ListProviderProfiles");
    providerProfiles = Array.isArray(result) ? result : [];
    populatePresetSelector();
  }

  function populatePresetSelector() {
    if (!els.pfPreset) return;
    els.pfPreset.innerHTML = `<option value="">— Custom / Local —</option>`;
    providerProfiles.forEach((prof) => {
      const opt = document.createElement("option");
      opt.value = prof.key;
      opt.textContent = prof.name;
      els.pfPreset.appendChild(opt);
    });
  }

  function applyPreset(key) {
    const prof = providerProfiles.find((p) => p.key === key);
    if (!prof) {
      if (els.pfDocsLink) els.pfDocsLink.classList.add("hidden");
      return;
    }
    els.pfUrl.value  = prof.base_url;
    els.pfType.value = prof.type;
    if (!els.pfName.value.trim()) {
      els.pfName.value = prof.name;
    }
    if (prof.docs_url && els.pfDocsLink) {
      els.pfDocsLink.href = prof.docs_url;
      els.pfDocsLink.textContent = `Get ${prof.name} API key ↗`;
      els.pfDocsLink.classList.remove("hidden");
    }
  }

  // ── PROV-05: Test connection ───────────────────────────────────────
  async function testProviderConnection() {
    const id = Number(els.pfId.value) || 0;
    if (!id) {
      if (els.pfTestResult) {
        els.pfTestResult.textContent = "Save the provider first";
        els.pfTestResult.style.color = "";
      }
      return;
    }
    if (els.pfTestResult) {
      els.pfTestResult.textContent = "Testing…";
      els.pfTestResult.style.color = "";
    }
    const result = await bridge.callService("TestConnection", { provider_id: id });
    if (!els.pfTestResult) return;
    if (result && result.ok) {
      els.pfTestResult.textContent = `✓ ${result.latency_ms} ms — ${result.model_count} model(s)`;
      els.pfTestResult.style.color = "var(--c-ok, #4caf50)";
    } else {
      const errMsg = (result && result.error) ? result.error : "connection failed";
      els.pfTestResult.textContent = `✗ ${errMsg}`;
      els.pfTestResult.style.color = "var(--c-err, #f44336)";
    }
  }

  // ── Provider CRUD ─────────────────────────────────────────────────
  async function loadProviders() {
    const result = await bridge.callService("ListProviders", { active_only: false });
    appState.providers = Array.isArray(result)
      ? result.map((item) => ({
          ...item,
          id:          Number(item.id),
          active:      Boolean(item.active),
          use_in_rag:  Boolean(item.use_in_rag),
          temperature: Number(item.temperature),
          num_ctx:     Number(item.num_ctx),
        }))
      : [];
    renderProvidersList();
    updateChatProviderSelector();
  }

  function renderProvidersList() {
    if (!els.providersList) return;
    els.providersList.innerHTML = "";
    if (appState.providers.length === 0) {
      els.providersList.innerHTML = `<p class="empty-hint">No providers yet.</p>`;
      return;
    }
    appState.providers.forEach((p) => {
      const item = document.createElement("div");
      item.className = `provider-item${p.active ? "" : " inactive"}`;
      item.dataset.id = String(p.id);
      item.innerHTML = `
        <span class="provider-item-dot">${escapeHtml(p.name.slice(0, 1).toUpperCase())}</span>
        <div class="provider-item-info">
          <span class="provider-item-name">${escapeHtml(p.name)}</span>
          <span class="provider-item-type">${escapeHtml(p.type)}</span>
        </div>
        ${p.active ? "" : '<span class="inactive-badge">off</span>'}
      `;
      item.addEventListener("click", () => openProviderForm(p));
      els.providersList.appendChild(item);
    });
  }

  function openProviderForm(p) {
    document.querySelectorAll(".provider-item").forEach((el) => el.classList.remove("selected"));
    const item = els.providersList.querySelector(`[data-id="${p.id}"]`);
    if (item) item.classList.add("selected");

    els.providerFormPlaceholder.classList.add("hidden");
    els.providerForm.classList.remove("hidden");
    els.pfDeleteBtn.classList.remove("hidden");

    els.pfId.value          = String(p.id);
    els.pfName.value        = p.name;
    els.pfType.value        = p.type;
    els.pfUrl.value         = p.url;
    // PROV-04: API key is not returned from backend; show placeholder if a key is stored.
    els.pfKey.value         = "";
    els.pfKey.placeholder   = p.api_key_set ? "•••••••• (unchanged)" : "sk-...";
    els.pfDescription.value = p.description;
    els.pfRag.checked       = p.use_in_rag;
    els.pfActive.checked    = p.active;
    els.pfTemperature.value = String(p.temperature);
    els.pfNumCtx.value      = String(p.num_ctx);
    // Reset preset selector and docs link for existing providers.
    if (els.pfPreset)   els.pfPreset.value = "";
    if (els.pfDocsLink) els.pfDocsLink.classList.add("hidden");
    // Clear previous test result.
    if (els.pfTestResult) { els.pfTestResult.textContent = ""; els.pfTestResult.style.color = ""; }
  }

  function hideProviderForm() {
    document.querySelectorAll(".provider-item").forEach((el) => el.classList.remove("selected"));
    els.providerFormPlaceholder.classList.remove("hidden");
    els.providerForm.classList.add("hidden");
  }

  function showNewProviderForm() {
    document.querySelectorAll(".provider-item").forEach((el) => el.classList.remove("selected"));
    els.providerFormPlaceholder.classList.add("hidden");
    els.providerForm.classList.remove("hidden");
    els.pfDeleteBtn.classList.add("hidden");

    els.pfId.value          = "";
    els.pfName.value        = "";
    els.pfType.value        = "openai-compatible";
    els.pfUrl.value         = "";
    els.pfKey.value         = "";
    els.pfKey.placeholder   = "sk-...";
    els.pfDescription.value = "";
    els.pfRag.checked       = false;
    els.pfActive.checked    = true;
    els.pfTemperature.value = "0.7";
    els.pfNumCtx.value      = "1024";
    if (els.pfPreset)     els.pfPreset.value = "";
    if (els.pfDocsLink)   els.pfDocsLink.classList.add("hidden");
    if (els.pfTestResult) { els.pfTestResult.textContent = ""; els.pfTestResult.style.color = ""; }
    els.pfName.focus();
  }

  async function saveProvider(event) {
    event.preventDefault();
    const id   = Number(els.pfId.value) || 0;
    const temp = Number(els.pfTemperature.value);
    const ctx  = Number(els.pfNumCtx.value);
    const payload = {
      name:        els.pfName.value.trim(),
      type:        els.pfType.value,
      url:         els.pfUrl.value.trim(),
      api_key:     els.pfKey.value.trim(), // empty = keep existing (Go-side handles this)
      description: els.pfDescription.value.trim(),
      use_in_rag:  els.pfRag.checked,
      active:      els.pfActive.checked,
      temperature: isNaN(temp) ? 0.7 : temp,
      num_ctx:     isNaN(ctx)  ? 1024 : ctx,
    };

    if (!payload.name) {
      els.status.textContent = "name required";
      return;
    }

    try {
      let result;
      if (id > 0) {
        result = await bridge.callService("UpdateProvider", { id, ...payload });
      } else {
        result = await bridge.callService("CreateProvider", payload);
      }

      if (result && result.ok === false && result.reason) {
        els.status.textContent = `save failed: ${result.reason}`;
        return;
      }

      const savedID = Number(result && result.id);
      if (!savedID || Number.isNaN(savedID)) {
        els.status.textContent = "save failed";
        return;
      }

      els.status.textContent = "saved";
      await loadProviders();
      const saved = appState.providers.find((p) => p.id === savedID);
      if (saved) openProviderForm(saved);
    } catch (err) {
      const msg = String(err && err.message ? err.message : err || "save failed");
      if (msg.toLowerCase().includes("unique") || msg.toLowerCase().includes("providers.name")) {
        els.status.textContent = "duplicate name";
      } else {
        els.status.textContent = `save failed: ${msg}`;
      }
    }
  }

  async function deleteCurrentProvider() {
    const id = Number(els.pfId.value) || 0;
    if (!id) return;
    if (!window.confirm("Delete this provider?")) return;

    const result = await bridge.callService("DeleteProvider", { id });
    if (!result || !result.ok) {
      els.status.textContent = "delete failed";
      return;
    }
    if (appState.activeProviderId === id) {
      appState.activeProviderId = null;
    }
    hideProviderForm();
    await loadProviders();
    els.status.textContent = "deleted";
  }

  // ── Model listing (PROV-04: uses provider_id, not api_key) ─────────
  async function loadModels() {
    const prov = getActiveProvider();
    if (!prov) {
      els.status.textContent = "select a provider";
      return;
    }
    // provider_id resolves credentials Go-side — no api_key sent over the wire.
    const result = await bridge.callService("ListModels", { provider_id: prov.id });
    if (!Array.isArray(result) || result.length === 0) {
      els.status.textContent = "no models found";
      return;
    }
    const previous = els.chatModel.value;
    els.chatModel.innerHTML = result
      .filter((item) => item && typeof item.id === "string" && item.id.trim() !== "")
      .map((item) => `<option value="${escapeHtml(item.id)}">${escapeHtml(item.id)}</option>`)
      .join("");

    if (previous && Array.from(els.chatModel.options).some((opt) => opt.value === previous)) {
      els.chatModel.value = previous;
    }
    els.status.textContent = "models loaded";
  }

  // ── Utilities ──────────────────────────────────────────────────────
  function activeConversation() {
    return appState.conversations.find((c) => c.id === appState.activeConversationId);
  }

  function escapeHtml(text) {
    return String(text)
      .replaceAll("&", "&amp;")
      .replaceAll("<", "&lt;")
      .replaceAll(">", "&gt;")
      .replaceAll('"', "&quot;")
      .replaceAll("'", "&#39;");
  }

  // ── Markdown rendering ─────────────────────────────────────────────
  function renderInlineMarkdown(text) {
    let html = escapeHtml(text);
    html = html.replace(/\*\*(.*?)\*\*/g, "<strong>$1</strong>");
    html = html.replace(/\*(.*?)\*/g, "<em>$1</em>");
    html = html.replace(/`([^`]+)`/g, "<code>$1</code>");
    html = html.replace(/\[(.*?)\]\((https?:\/\/[^\s)]+)\)/g,
      '<a href="$2" target="_blank" rel="noreferrer">$1</a>');
    html = html.replace(/&lt;think&gt;([\s\S]*?)&lt;\/think&gt;/g,
      '<details class="think"><summary>Reasoning</summary><div>$1</div></details>');
    return html;
  }

  function parseTableBlock(lines) {
    if (lines.length < 2) return "";
    const header    = lines[0].split("|").map((s) => s.trim()).filter(Boolean);
    const separator = lines[1].split("|").map((s) => s.trim()).filter(Boolean);
    if (!header.length || separator.length !== header.length ||
        !separator.every((s) => /^-+:?$|^:-+:$|^:?-+$/.test(s))) {
      return "";
    }
    const body  = lines.slice(2).map((row) => row.split("|").map((s) => s.trim()).filter(Boolean));
    const thead = `<thead><tr>${header.map((h) => `<th>${renderInlineMarkdown(h)}</th>`).join("")}</tr></thead>`;
    const tbody = `<tbody>${body.map((cols) =>
      `<tr>${header.map((_, i) => `<td>${renderInlineMarkdown(cols[i] || "")}</td>`).join("")}</tr>`
    ).join("")}</tbody>`;
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
    const html  = [];
    let i = 0;

    while (i < lines.length) {
      const line = lines[i];

      if (/^\s*$/.test(line)) { html.push("<br>"); i++; continue; }

      if (line.includes("|") && i + 1 < lines.length && lines[i + 1].includes("|")) {
        const tableLines = [line, lines[i + 1]];
        let j = i + 2;
        while (j < lines.length && lines[j].includes("|")) { tableLines.push(lines[j]); j++; }
        const tbl = parseTableBlock(tableLines);
        if (tbl) { html.push(tbl); i = j; continue; }
      }

      const heading = line.match(/^(#{1,3})\s+(.*)$/);
      if (heading) {
        html.push(`<h${heading[1].length}>${renderInlineMarkdown(heading[2])}</h${heading[1].length}>`);
        i++; continue;
      }

      if (/^>\s+/.test(line)) {
        html.push(`<blockquote>${renderInlineMarkdown(line.replace(/^>\s+/, ""))}</blockquote>`);
        i++; continue;
      }

      if (/^\d+\.\s+/.test(line)) {
        const items = [];
        let j = i;
        while (j < lines.length && /^\d+\.\s+/.test(lines[j])) {
          items.push(lines[j].replace(/^\d+\.\s+/, ""));
          j++;
        }
        html.push(`<ol>${items.map((it) => `<li>${renderInlineMarkdown(it)}</li>`).join("")}</ol>`);
        i = j; continue;
      }

      if (/^[-*]\s+/.test(line)) {
        const items = [];
        let j = i;
        while (j < lines.length && /^[-*]\s+/.test(lines[j])) {
          items.push(lines[j].replace(/^[-*]\s+/, ""));
          j++;
        }
        html.push(`<ul>${items.map((it) => `<li>${renderInlineMarkdown(it)}</li>`).join("")}</ul>`);
        i = j; continue;
      }

      html.push(`<p>${renderInlineMarkdown(line)}</p>`);
      i++;
    }

    let merged = html.join("\n");
    merged = merged.replace(/__CODE_BLOCK_(\d+)__/g, (_m, idxStr) => {
      const block = codeBlocks[Number(idxStr)];
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
          { left: "$",  right: "$",  display: false },
        ],
        throwOnError: false,
      });
      return;
    }
    container.querySelectorAll("p,li,blockquote").forEach((node) => {
      node.innerHTML = node.innerHTML
        .replace(/\$\$([\s\S]+?)\$\$/g, '<span class="math-block">$1</span>')
        .replace(/\$(.+?)\$/g,          '<span class="math-inline">$1</span>');
    });
  }

  // ── Message rendering ──────────────────────────────────────────────
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

  // ── Conversation sidebar ───────────────────────────────────────────
  function renderConversationList() {
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

  async function loadConversationMessages(conversationId) {
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

  async function loadPersistedConversations() {
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

  async function newConversation() {
    const prov = getActiveProvider();
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

  // ── Streaming helpers ──────────────────────────────────────────────
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

  // ── Send prompt (PROV-04: uses provider_id, not api_key) ───────────
  async function sendPrompt() {
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

    // PROV-04: provider_id resolves credentials Go-side — no api_key in payload.
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

  // ── Message actions ────────────────────────────────────────────────
  function copyMessage(index) {
    const conv = activeConversation();
    const msg  = conv && conv.messages[index];
    if (!msg) return;
    navigator.clipboard.writeText(msg.content);
    els.status.textContent = "copied";
    setTimeout(() => { els.status.textContent = "ready"; }, 800);
  }

  function editMessage(index) {
    const conv = activeConversation();
    const msg  = conv && conv.messages[index];
    if (!msg || msg.role !== "user") return;
    els.prompt.value = msg.content;
    conv.messages.splice(index, 1);
    renderMessages();
    els.prompt.focus();
  }

  function regenerateMessage(index) {
    const conv = activeConversation();
    const msg  = conv && conv.messages[index];
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

  // ── Event bindings ─────────────────────────────────────────────────
  function bindEvents() {
    els.tabs.forEach((btn) => btn.addEventListener("click", () => switchTab(btn.dataset.tab)));

    els.newChat.addEventListener("click", newConversation);
    els.send.addEventListener("click", sendPrompt);
    els.stop.addEventListener("click", cancelGeneration);
    els.sidebarToggle.addEventListener("click", toggleSidebar);
    els.refreshModels.addEventListener("click", loadModels);

    els.chatProvider.addEventListener("change", async () => {
      const newId = Number(els.chatProvider.value) || null;
      appState.activeProviderId = newId;
      persistSettingsToStorage();
      await loadModels();
    });

    els.chatModel.addEventListener("change", () => {
      const conv = activeConversation();
      if (conv) conv.model = els.chatModel.value;
    });

    els.prompt.addEventListener("keydown", (e) => {
      if (e.key === "Enter" && !e.shiftKey) { e.preventDefault(); sendPrompt(); }
    });

    els.settingsNavBtns.forEach((btn) =>
      btn.addEventListener("click", () => switchSettingsSection(btn.dataset.section)));

    if (els.language) {
      els.language.addEventListener("change", () => {
        appState.settings.language = els.language.value;
        persistSettingsToStorage();
      });
    }
    if (els.theme) {
      els.theme.addEventListener("change", () => {
        appState.settings.theme = els.theme.value;
        persistSettingsToStorage();
      });
    }

    if (els.newProviderBtn)  els.newProviderBtn.addEventListener("click", showNewProviderForm);
    if (els.providerForm)    els.providerForm.addEventListener("submit", saveProvider);
    if (els.pfDeleteBtn)     els.pfDeleteBtn.addEventListener("click", deleteCurrentProvider);

    // PROV-05: Test connection button
    if (els.pfTestBtn) els.pfTestBtn.addEventListener("click", testProviderConnection);

    // PROV-08: Preset selector
    if (els.pfPreset) els.pfPreset.addEventListener("change", () => applyPreset(els.pfPreset.value));

    // PROV-06: show user-friendly error message from structured chat:error event
    bridge.eventsOn("chat:chunk", (chunk) => {
      if (!chunk || typeof chunk.content !== "string") return;
      appendAssistantChunk(chunk.content);
      if (chunk.done) stopStreaming("done");
    });
    bridge.eventsOn("chat:done",  () => stopStreaming("done"));
    bridge.eventsOn("chat:error", (payload) => {
      if (payload && payload.message) {
        els.status.textContent = payload.message;
      }
      stopStreaming("cancel");
    });
  }

  // ── Init ───────────────────────────────────────────────────────────
  async function init() {
    window.liaotao = { copyMessage, editMessage, regenerateMessage, deleteMessage };

    initSidebarResizer();
    applySidebarState();
    loadSettingsFromStorage();
    applySettingsToUI();
    bindEvents();
    await loadProviders();           // populates chat provider selector from DB
    await loadProviderProfiles();    // PROV-08: populates preset dropdown
    await loadModels();              // loads models for the active provider
    await loadPersistedConversations();
  }

  init();

})();
