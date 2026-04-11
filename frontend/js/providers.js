/*
  providers.js -- Provider CRUD, preset profiles, connection test, and model listing.
  Responsibilities: load/create/update/delete providers from SQLite via Go bindings,
  populate Settings UI, manage preset profiles (PROV-08), test connection (PROV-05),
  list models for the active provider.
*/

import { appState, els, persistSettingsToStorage } from "./state.js";
import { bridge }     from "./bridge.js";
import { escapeHtml } from "./markdown.js";
import { parseWailsError, applyFieldError, clearFieldError } from "./errors.js";
import { t }          from "./i18n.js";

// Module-level cache for PROV-08 preset profiles.
let providerProfiles = [];
const modelCache = new Map();

function providerStatusSymbol(providerID) {
  const status = appState.providerStatus[providerID] || "unknown";
  if (status === "connected") return "✓";
  if (status === "disconnected") return "✗";
  return "…";
}

function providerStatusLabel(providerID) {
  const status = appState.providerStatus[providerID] || "unknown";
  if (status === "connected") return t("providers.status_connected");
  if (status === "disconnected") return t("providers.status_disconnected");
  return t("providers.status_unknown");
}

export function setProviderStatus(providerID, status) {
  if (!providerID || providerID <= 0) return;
  appState.providerStatus[providerID] = status;
}

// CORCT-02: provider-specific URL placeholders used in Settings form.
const providerURLPlaceholders = {
  "openai-compatible": "https://api.openai.com/v1",
  ollama: "http://localhost:11434/v1",
  gemini: "https://generativelanguage.googleapis.com/v1beta/openai",
  anthropic: "https://api.anthropic.com/v1",
  mistral: "https://api.mistral.ai/v1",
};

// ── Provider helpers ───────────────────────────────────────────────────────
export function getActiveProvider() {
  if (appState.activeProviderId == null) return null;
  return appState.providers.find((p) => p.id === appState.activeProviderId) || null;
}

export function updateChatProviderSelector() {
  const active = appState.providers.filter((p) => p.active);

  // Always prepend the "Automat" virtual entry (id=0 → Smart Router).
  const automatOption = `<option value="0">✦ Automat</option>`;

  if (active.length === 0) {
    els.chatProvider.innerHTML = automatOption +
      `<option value="" disabled>${t("chat.no_providers")}</option>`;
    appState.activeProviderId = 0;
    persistSettingsToStorage();
    return;
  }

  els.chatProvider.innerHTML = automatOption +
    active.map((p) => `<option value="${p.id}">${escapeHtml(p.name)} ${providerStatusSymbol(p.id)}</option>`).join("");

  const prevId = appState.activeProviderId;
  if (prevId === 0) {
    // Automat was previously selected — restore it.
    els.chatProvider.value = "0";
  } else if (prevId != null && active.some((p) => p.id === prevId)) {
    els.chatProvider.value = String(prevId);
  } else {
    // Default: select the first real provider.
    appState.activeProviderId = active[0].id;
    els.chatProvider.value   = String(active[0].id);
  }
  persistSettingsToStorage();
}

// ── PROV-08: Provider profiles (presets) ──────────────────────────────────
export async function loadProviderProfiles() {
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

export function applyPreset(key) {
  const prof = providerProfiles.find((p) => p.key === key);
  if (!prof) {
    if (els.pfDocsLink) els.pfDocsLink.classList.add("hidden");
    updateProviderURLPlaceholder();
    return;
  }
  els.pfUrl.value  = prof.base_url;
  els.pfType.value = prof.type;
  updateProviderURLPlaceholder();
  if (!els.pfName.value.trim()) {
    els.pfName.value = prof.name;
  }
  if (prof.docs_url && els.pfDocsLink) {
    els.pfDocsLink.href        = prof.docs_url;
    els.pfDocsLink.textContent = t("providers.get_api_key", { name: prof.name });
    els.pfDocsLink.classList.remove("hidden");
  }
}

export function updateProviderURLPlaceholder() {
  if (!els.pfUrl || !els.pfType) return;
  const selectedType = els.pfType.value || "openai-compatible";
  const placeholder = providerURLPlaceholders[selectedType] || providerURLPlaceholders["openai-compatible"];
  els.pfUrl.placeholder = placeholder;
}

function setModelSelectorOptions(models, selectedModel = "") {
  const filter = (appState.modelFilterQuery || "").trim().toLowerCase();
  const safeModels = Array.isArray(models)
    ? models.filter((item) => item && typeof item.id === "string" && item.id.trim() !== "")
            .filter((item) => !filter || String(item.id).toLowerCase().includes(filter))
    : [];

  if (safeModels.length === 0) {
    els.chatModel.innerHTML = `<option value="">${t("chat.no_models")}</option>`;
    els.chatModel.value = "";
    return;
  }

  els.chatModel.innerHTML = safeModels
    .map((item) => `<option value="${escapeHtml(item.id)}">${escapeHtml(item.id)}</option>`)
    .join("");

  if (selectedModel && safeModels.some((item) => item.id === selectedModel)) {
    els.chatModel.value = selectedModel;
    return;
  }

  const previous = els.chatModel.dataset.pendingValue || els.chatModel.value;
  if (previous && safeModels.some((item) => item.id === previous)) {
    els.chatModel.value = previous;
    return;
  }

  els.chatModel.value = safeModels[0].id;
}

function setUnifiedModelSelectorOptions(selectedModel = "") {
  const active = appState.providers.filter((p) => p.active);
  const filter = (appState.modelFilterQuery || "").trim().toLowerCase();
  const optionGroups = [];
  for (const provider of active) {
    const models = modelCache.get(provider.id) || [];
    if (models.length === 0) continue;
    const filtered = models.filter((item) => {
      const id = String(item.id || "");
      if (!filter) return true;
      return id.toLowerCase().includes(filter) || provider.name.toLowerCase().includes(filter);
    });
    if (filtered.length === 0) continue;
    const options = filtered
      .map((item) => `<option value="${escapeHtml(`${provider.id}::${item.id}`)}">${escapeHtml(item.id)}</option>`)
      .join("");
    optionGroups.push(`<optgroup label="${escapeHtml(provider.name)} (${providerStatusLabel(provider.id)})">${options}</optgroup>`);
  }
  if (optionGroups.length === 0) {
    els.chatModel.innerHTML = `<option value="">${t("chat.open_to_load")}</option>`;
    els.chatModel.value = "";
    return;
  }
  els.chatModel.innerHTML = optionGroups.join("");
  const selectedKey = appState.activeProviderId > 0
    ? `${appState.activeProviderId}::${selectedModel || els.chatModel.dataset.pendingValue || ""}`
    : "";
  if (selectedKey && els.chatModel.querySelector(`option[value="${CSS.escape(selectedKey)}"]`)) {
    els.chatModel.value = selectedKey;
    return;
  }
  const previous = els.chatModel.dataset.pendingValue || "";
  if (previous && els.chatModel.querySelector(`option[value="${CSS.escape(previous)}"]`)) {
    els.chatModel.value = previous;
    return;
  }
  const firstOption = els.chatModel.querySelector("option");
  if (firstOption) {
    els.chatModel.value = firstOption.value;
  }
}

export function rememberLastUsedModel(providerID, providerName, model) {
  const normalizedModel = String(model || "").trim();
  if (!normalizedModel) return;
  const normalizedProviderID = Number(providerID) || 0;
  appState.lastUsedModels = appState.lastUsedModels
    .filter((item) => !(item.providerId === normalizedProviderID && item.model === normalizedModel));
  appState.lastUsedModels.unshift({
    providerId: normalizedProviderID,
    providerName: providerName || (normalizedProviderID > 0 ? "provider" : "Automat"),
    model: normalizedModel,
    usedAt: Date.now(),
  });
  appState.lastUsedModels = appState.lastUsedModels.slice(0, 6);
  persistSettingsToStorage();
  renderLastUsedModels();
}

export function renderLastUsedModels() {
  if (!els.lastUsedModels) return;
  if (!Array.isArray(appState.lastUsedModels) || appState.lastUsedModels.length === 0) {
    els.lastUsedModels.innerHTML = "";
    return;
  }
  els.lastUsedModels.innerHTML = appState.lastUsedModels
    .map((entry, idx) => `<button class="last-used-chip" type="button" data-last-used-index="${idx}" title="${escapeHtml(entry.providerName)}">${escapeHtml(entry.model)}</button>`)
    .join("");
  els.lastUsedModels.querySelectorAll("button[data-last-used-index]").forEach((btn) => {
    btn.addEventListener("click", () => {
      const idx = Number(btn.dataset.lastUsedIndex);
      const item = appState.lastUsedModels[idx];
      if (!item) return;
      if (item.providerId > 0 && appState.providers.some((p) => p.id === item.providerId && p.active)) {
        appState.activeProviderId = item.providerId;
        els.chatProvider.value = String(item.providerId);
      }
      syncChatModelSelector(item.model);
      persistSettingsToStorage();
      els.status.textContent = t("sidebar.last_used_restored");
    });
  });
}

export function syncChatModelSelector(selectedModel = "") {
  if (!els.chatModel) return;

  if (appState.activeProviderId === 0) {
    setUnifiedModelSelectorOptions(selectedModel);
    return;
  }

  const prov = getActiveProvider();
  if (!prov) {
    els.chatModel.innerHTML = `<option value="">${t("chat.select_provider_first")}</option>`;
    els.chatModel.value = "";
    return;
  }

  const cached = modelCache.get(prov.id);
  if (cached && cached.length > 0) {
    setModelSelectorOptions(cached, selectedModel);
    return;
  }

  if (selectedModel) {
    els.chatModel.innerHTML = `<option value="${escapeHtml(selectedModel)}">${escapeHtml(selectedModel)} (saved)</option>`;
    els.chatModel.value = selectedModel;
    return;
  }

  els.chatModel.innerHTML = `<option value="">${t("chat.open_to_load")}</option>`;
  els.chatModel.value = "";
}

export function clearModelCache(providerID) {
  if (typeof providerID === "number" && providerID > 0) {
    modelCache.delete(providerID);
    return;
  }
  modelCache.clear();
}

// ── PROV-05: Test connection ───────────────────────────────────────────────
export async function testProviderConnection() {
  const id = Number(els.pfId.value) || 0;
  if (!id) {
    if (els.pfTestResult) {
      els.pfTestResult.textContent = t("providers.save_first");
      els.pfTestResult.style.color = "";
    }
    return;
  }
  if (els.pfTestResult) {
    els.pfTestResult.textContent = t("providers.testing");
    els.pfTestResult.style.color = "";
  }
  const result = await bridge.callService("TestConnection", { provider_id: id });
  if (!els.pfTestResult) return;
  if (result && result.ok) {
    setProviderStatus(id, "connected");
    els.pfTestResult.textContent = `✓ ${result.latency_ms} ms — ${result.model_count} model(s)`;
    els.pfTestResult.style.color = "var(--c-ok, #4caf50)";
  } else {
    setProviderStatus(id, "disconnected");
    const errMsg = (result && result.error) ? result.error : "connection failed";
    els.pfTestResult.textContent = `✗ ${errMsg}`;
    els.pfTestResult.style.color = "var(--c-err, #f44336)";
  }
  updateChatProviderSelector();
  renderProvidersList();
}

// ── Provider CRUD ──────────────────────────────────────────────────────────
export async function loadProviders() {
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
  renderLastUsedModels();
}

export function renderProvidersList() {
  if (!els.providersList) return;
  els.providersList.innerHTML = "";
  if (appState.providers.length === 0) {
    els.providersList.innerHTML = `<p class="empty-hint">${t("providers.no_providers")}</p>`;
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
        <span class="provider-item-type">${escapeHtml(p.type)} · ${providerStatusSymbol(p.id)} ${providerStatusLabel(p.id)}</span>
      </div>
      ${p.active ? "" : `<span class="inactive-badge">${t("providers.inactive_badge")}</span>`}
    `;
    item.addEventListener("click", () => openProviderForm(p));
    els.providersList.appendChild(item);
  });
}

export function openProviderForm(p) {
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
  // PROV-04: API key not returned from backend; show placeholder if a key is stored.
  els.pfKey.value         = "";
  els.pfKey.placeholder   = p.api_key_set ? "•••••••• (unchanged)" : "sk-...";
  els.pfDescription.value = p.description;
  els.pfRag.checked       = p.use_in_rag;
  els.pfActive.checked    = p.active;
  els.pfTemperature.value = String(p.temperature);
  els.pfNumCtx.value      = String(p.num_ctx);
  updateProviderURLPlaceholder();
  if (els.pfPreset)   els.pfPreset.value = "";
  if (els.pfDocsLink) els.pfDocsLink.classList.add("hidden");
  if (els.pfTestResult) { els.pfTestResult.textContent = ""; els.pfTestResult.style.color = ""; }
  clearFieldError(els.pfName, els.status);
}

export function hideProviderForm() {
  document.querySelectorAll(".provider-item").forEach((el) => el.classList.remove("selected"));
  els.providerFormPlaceholder.classList.remove("hidden");
  els.providerForm.classList.add("hidden");
}

export function showNewProviderForm() {
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
  updateProviderURLPlaceholder();
  if (els.pfPreset)     els.pfPreset.value = "";
  if (els.pfDocsLink)   els.pfDocsLink.classList.add("hidden");
  if (els.pfTestResult) { els.pfTestResult.textContent = ""; els.pfTestResult.style.color = ""; }
  els.pfName.focus();
}

export async function saveProvider(event) {
  event.preventDefault();
  const id   = Number(els.pfId.value) || 0;
  const temp = Number(els.pfTemperature.value);
  const ctx  = Number(els.pfNumCtx.value);
  const payload = {
    name:        els.pfName.value.trim(),
    type:        els.pfType.value,
    url:         els.pfUrl.value.trim(),
    api_key:     els.pfKey.value.trim(),
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
    clearModelCache(id || undefined);
    await loadProviders();
    const saved = appState.providers.find((p) => p.id === savedID);
    if (saved) openProviderForm(saved);
  } catch (err) {
    const { message, field } = parseWailsError(err);
    const fieldEl = field ? document.getElementById(field) : null;
    applyFieldError(fieldEl || els.pfName, els.status, message);
  }
}

// inlineConfirm: 1st click → button shows confirm title, 2nd click → runs onConfirm.
// Avoids window.confirm() which is silently blocked by Wails/WKWebView.
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

export async function deleteCurrentProvider() {
  const id = Number(els.pfId.value) || 0;
  if (!id) return;
  inlineConfirm(els.pfDeleteBtn, async () => {
    const result = await bridge.callService("DeleteProvider", { id });
    if (!result || !result.ok) {
      els.status.textContent = "delete failed";
      return;
    }
    if (appState.activeProviderId === id) {
      appState.activeProviderId = null;
    }
    clearModelCache(id);
    hideProviderForm();
    await loadProviders();
    els.status.textContent = "deleted";
  });
}

// ── Model listing (PROV-04: uses provider_id, not api_key) ────────────────
export async function loadModels(options = {}) {
  const { force = false, preferredModel = "" } = options;
  const prov = getActiveProvider();
  if (!prov) {
    if (appState.activeProviderId === 0) {
      syncChatModelSelector(preferredModel);
      els.status.textContent = "automat active";
    } else {
      syncChatModelSelector(preferredModel);
      els.status.textContent = "select a provider";
    }
    return;
  }

  if (!force && modelCache.has(prov.id)) {
    setModelSelectorOptions(modelCache.get(prov.id), preferredModel);
    els.status.textContent = "models cached";
    return;
  }

  els.status.textContent = "loading models...";
  let result = null;
  try {
    result = await bridge.callService("ListModels", { provider_id: prov.id });
  } catch {
    setProviderStatus(prov.id, "disconnected");
    syncChatModelSelector(preferredModel);
    els.status.textContent = "model load failed";
    updateChatProviderSelector();
    renderProvidersList();
    return;
  }
  if (!Array.isArray(result) || result.length === 0) {
    setProviderStatus(prov.id, "disconnected");
    modelCache.delete(prov.id);
    syncChatModelSelector(preferredModel);
    els.status.textContent = "no models found";
    updateChatProviderSelector();
    renderProvidersList();
    return;
  }

  setProviderStatus(prov.id, "connected");
  modelCache.set(prov.id, result);
  setModelSelectorOptions(result, preferredModel);
  updateChatProviderSelector();
  renderProvidersList();
  els.status.textContent = "models loaded";
}
