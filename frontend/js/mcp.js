/*
  mcp.js -- MCP server management UI (MCP-07).
  Responsibilities: load/render MCP server list in Settings, form CRUD,
  transport-aware field toggling (http/sse vs stdio).
*/

import { bridge } from "./bridge.js";
import { parseWailsError, applyFieldError, clearFieldError } from "./errors.js";

// ── DOM refs (scoped to MCP section) ──────────────────────────────────────
const $ = (id) => document.getElementById(id);

function mcpEls() {
  return {
    list:          $("mcp-servers-list"),
    form:          $("mcp-server-form"),
    placeholder:   $("mcp-form-placeholder"),
    id:            $("msf-id"),
    name:          $("msf-name"),
    transport:     $("msf-transport"),
    urlGroup:      $("msf-url-group"),
    url:           $("msf-url"),
    cmdGroup:      $("msf-cmd-group"),
    command:       $("msf-command"),
    argsGroup:     $("msf-args-group"),
    args:          $("msf-args"),
    active:        $("msf-active"),
    status:        $("msf-status"),
    pingBtn:       $("msf-ping-btn"),
    pingStatus:    $("msf-ping-status"),
    copyBtn:       $("msf-copy-btn"),
    toolsList:     $("msf-tools-list"),
    deleteBtn:     $("msf-delete-btn"),
    newBtn:        $("new-mcp-btn"),
  };
}

// inlineConfirm: 1st click → "Confirmer ?", 2nd click → runs onConfirm.
// Avoids window.confirm() which is silently blocked by Wails/WKWebView.
function inlineConfirm(btn, onConfirm) {
  if (btn.dataset.confirming === "1") {
    delete btn.dataset.confirming;
    clearTimeout(Number(btn.dataset.confirmTimer));
    btn.innerHTML = btn.dataset.origLabel || "🗑";
    btn.title = btn.dataset.origTitle || "Delete MCP server";
    onConfirm();
    return;
  }
  btn.dataset.confirming = "1";
  btn.dataset.origLabel = btn.innerHTML;
  btn.dataset.origTitle = btn.title || "Delete MCP server";
  btn.innerHTML = "✓";
  btn.title = "Confirm deletion";
  btn.dataset.confirmTimer = String(setTimeout(() => {
    delete btn.dataset.confirming;
    btn.innerHTML = btn.dataset.origLabel || "🗑";
    btn.title = btn.dataset.origTitle || "Delete MCP server";
  }, 3000));
}

// ── Transport field toggle ─────────────────────────────────────────────────
function applyTransportToggle(transport) {
  const e = mcpEls();
  const isStdio = transport === "stdio";
  e.urlGroup.classList.toggle("hidden", isStdio);
  e.cmdGroup.classList.toggle("hidden", !isStdio);
  e.argsGroup.classList.toggle("hidden", !isStdio);
}

// ── List rendering ─────────────────────────────────────────────────────────
export async function loadMCPServers() {
  const servers = await bridge.callService("ListMCPServers");
  const e = mcpEls();
  if (!e.list) return;

  if (!Array.isArray(servers) || servers.length === 0) {
    e.list.innerHTML = "<p class=\"empty-hint\">No MCP servers configured.</p>";
    return;
  }

  e.list.innerHTML = servers.map((s) => `
    <div class="provider-item" data-id="${s.id}">
      <span class="provider-name">${s.name}</span>
      <span class="provider-type">${s.transport}</span>
      <span class="provider-status ${s.active ? "active" : "inactive"}">${s.active ? "●" : "○"}</span>
    </div>
  `).join("");

  e.list.querySelectorAll(".provider-item").forEach((item) => {
    item.addEventListener("click", async () => {
      const id = Number(item.dataset.id);
      const srv = servers.find((s) => s.id === id);
      if (srv) showMCPForm(srv);
    });
  });
}

// ── Form helpers ───────────────────────────────────────────────────────────
function showMCPForm(srv) {
  const e = mcpEls();
  e.placeholder.classList.add("hidden");
  e.form.classList.remove("hidden");

  e.id.value        = srv ? srv.id    : "";
  e.name.value      = srv ? srv.name  : "";
  e.active.checked  = srv ? srv.active : true;
  if (e.status) clearFieldError(e.name, e.status);

  const transport = srv ? srv.transport : "http";
  e.transport.value = transport;
  e.url.value       = srv ? srv.url     : "";
  e.command.value   = srv ? srv.command : "";
  e.args.value      = srv ? srv.args    : "[]";

  applyTransportToggle(transport);

  if (e.deleteBtn) e.deleteBtn.classList.toggle("hidden", !srv || !srv.id);

  // Reset ping area whenever a different server is selected.
  if (e.pingStatus) e.pingStatus.textContent = "";
  if (e.toolsList)  { e.toolsList.innerHTML = ""; e.toolsList.classList.add("hidden"); }
}

export function showNewMCPServerForm() {
  showMCPForm(null);
}

export async function saveMCPServer(e) {
  e.preventDefault();
  const els = mcpEls();
  if (els.status) els.status.textContent = "";

  const payload = {
    id:        Number(els.id.value) || 0,
    name:      els.name.value.trim(),
    transport: els.transport.value,
    url:       els.url.value.trim(),
    command:   els.command.value.trim(),
    args:      els.args.value.trim() || "[]",
    active:    els.active.checked,
  };

  try {
    const result = await bridge.callService("SaveMCPServer", payload);
    if (result && result.ok) {
      els.form.classList.add("hidden");
      els.placeholder.classList.remove("hidden");
      await loadMCPServers();
    } else {
      applyFieldError(null, els.status, "save failed");
    }
  } catch (err) {
    const { message, field } = parseWailsError(err);
    const fieldEl = field ? document.getElementById(field) : null;
    applyFieldError(fieldEl, els.status, message);
  }
}

export async function deleteCurrentMCPServer() {
  const els = mcpEls();
  const id = Number(els.id.value);
  if (!id || !els.deleteBtn) return;
  inlineConfirm(els.deleteBtn, async () => {
    try {
      await bridge.callService("DeleteMCPServer", id);
      els.form.classList.add("hidden");
      els.placeholder.classList.remove("hidden");
      await loadMCPServers();
    } catch (err) {
      const msg = String(err && err.message ? err.message : err || "delete failed");
      if (els.status) els.status.textContent = msg;
    }
  });
}

// ── Ping / test connection ────────────────────────────────────────────────
async function pingMCPServer() {
  const e = mcpEls();
  const id = Number(e.id.value);
  if (!id) {
    if (e.pingStatus) { e.pingStatus.className = "test-result err"; e.pingStatus.textContent = "Save the server first."; }
    return;
  }
  if (e.pingBtn)    e.pingBtn.disabled = true;
  if (e.pingStatus) { e.pingStatus.className = "test-result"; e.pingStatus.textContent = "Testing…"; }
  if (e.toolsList)  { e.toolsList.innerHTML = ""; e.toolsList.classList.add("hidden"); }

  try {
    const res = await bridge.callService("PingMCPServer", id);
    if (res && res.ok) {
      const tools = Array.isArray(res.tools) ? res.tools : [];
      if (e.pingStatus) { e.pingStatus.className = "test-result ok"; e.pingStatus.textContent = `✓ Connected — ${tools.length} tool(s)`; }
      if (e.toolsList && tools.length > 0) {
        e.toolsList.innerHTML = tools.map((t) => `<span class="tool-badge">${t}</span>`).join("");
        e.toolsList.classList.remove("hidden");
      }
    } else {
      const errMsg = res && res.error ? res.error : "unreachable";
      if (e.pingStatus) { e.pingStatus.className = "test-result err"; e.pingStatus.textContent = `✗ ${errMsg}`; }
    }
  } catch (err) {
    const { message } = parseWailsError(err);
    if (e.pingStatus) { e.pingStatus.className = "test-result err"; e.pingStatus.textContent = `✗ ${message}`; }
  } finally {
    if (e.pingBtn) e.pingBtn.disabled = false;
  }
}

// ── Copy config to clipboard ───────────────────────────────────────────────
async function copyMCPConfig() {
  const e = mcpEls();
  const transport = e.transport ? e.transport.value : "?";
  const isStdio = transport === "stdio";

  const lines = [
    `Name:      ${e.name ? e.name.value.trim() || "(unsaved)" : "?"}`,
    `Transport: ${transport}`,
  ];
  if (isStdio) {
    lines.push(`Command:   ${e.command ? e.command.value.trim() || "(empty)" : "?"}`);
    lines.push(`Args:      ${e.args ? e.args.value.trim() || "[]" : "[]"}`);
  } else {
    lines.push(`URL:       ${e.url ? e.url.value.trim() || "(empty)" : "?"}`);
  }
  lines.push(`Active:    ${e.active && e.active.checked ? "yes" : "no"}`);
  lines.push(`ID:        ${e.id ? e.id.value || "0 (not saved yet)" : "?"}`);

  const text = lines.join("\n");
  try {
    await navigator.clipboard.writeText(text);
    if (e.copyBtn) {
      const orig = e.copyBtn.innerHTML;
      const origTitle = e.copyBtn.title || "Copy config";
      e.copyBtn.innerHTML = "✓";
      e.copyBtn.title = "Copied";
      setTimeout(() => {
        if (e.copyBtn) {
          e.copyBtn.innerHTML = orig;
          e.copyBtn.title = origTitle;
        }
      }, 1500);
    }
  } catch {
    if (e.pingStatus) e.pingStatus.textContent = "Clipboard unavailable";
  }
}

// ── Init transport toggle listener ────────────────────────────────────────
export function initMCPFormListeners() {
  const e = mcpEls();
  if (!e.transport) return;
  e.transport.addEventListener("change", () => applyTransportToggle(e.transport.value));
  if (e.newBtn)    e.newBtn.addEventListener("click", showNewMCPServerForm);
  if (e.form)      e.form.addEventListener("submit", saveMCPServer);
  if (e.deleteBtn) e.deleteBtn.addEventListener("click", deleteCurrentMCPServer);
  if (e.pingBtn)   e.pingBtn.addEventListener("click", pingMCPServer);
  if (e.copyBtn)   e.copyBtn.addEventListener("click", copyMCPConfig);
}
