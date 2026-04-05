/*
  providers.js -- Provider CRUD, preset profiles, connection test, and model listing.
  Responsibilities: load/create/update/delete providers from SQLite via Go bindings,
  populate Settings UI, manage preset profiles (PROV-08), test connection (PROV-05),
  list models for the active provider.
*/

import { appState, els, persistSettingsToStorage } from "./state.js";
import { bridge }     from "./bridge.js";
import { escapeHtml } from "./markdown.js";

// Module-level cache for PROV-08 preset profiles.
let providerProfiles = [];

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
      `<option value="" disabled>No providers – add one in Settings</option>`;
    appState.activeProviderId = 0;
    persistSettingsToStorage();
    return;
  }

  els.chatProvider.innerHTML = automatOption +
    active.map((p) => `<option value="${p.id}">${escapeHtml(p.name)}</option>`).join("");

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
    return;
  }
  els.pfUrl.value  = prof.base_url;
  els.pfType.value = prof.type;
  if (!els.pfName.value.trim()) {
    els.pfName.value = prof.name;
  }
  if (prof.docs_url && els.pfDocsLink) {
    els.pfDocsLink.href        = prof.docs_url;
    els.pfDocsLink.textContent = `Get ${prof.name} API key ↗`;
    els.pfDocsLink.classList.remove("hidden");
  }
}

// ── PROV-05: Test connection ───────────────────────────────────────────────
export async function testProviderConnection() {
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
}

export function renderProvidersList() {
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
  if (els.pfPreset)   els.pfPreset.value = "";
  if (els.pfDocsLink) els.pfDocsLink.classList.add("hidden");
  if (els.pfTestResult) { els.pfTestResult.textContent = ""; els.pfTestResult.style.color = ""; }
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

export async function deleteCurrentProvider() {
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

// ── Model listing (PROV-04: uses provider_id, not api_key) ────────────────
export async function loadModels() {
  const prov = getActiveProvider();
  if (!prov) {
    if (appState.activeProviderId === 0) {
      // Automat mode: the router picks the provider; we can't pre-fetch models.
      els.chatModel.innerHTML = `<option value="">— router selects —</option>`;
      els.status.textContent  = "automat active";
    } else {
      els.status.textContent = "select a provider";
    }
    return;
  }
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
