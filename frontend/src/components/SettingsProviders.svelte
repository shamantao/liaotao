<!--
  SettingsProviders.svelte -- Provider management section.
  Responsibilities: list providers, CRUD form with preset selector,
  test connection, delete/save actions.
-->
<script>
  import { providers, activeProviderId, setProviderStatus } from "../stores/providers.js";
  import * as bridge from "../lib/bridge.js";

  // Icons
  const iconPlus = '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M5 12h14"/><path d="M12 5v14"/></svg>';
  const iconCheck = '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M20 6 9 17l-5-5"/></svg>';
  const iconTrash = '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M3 6h18"/><path d="M19 6v14c0 1-1 2-2 2H7c-1 0-2-1-2-2V6"/><path d="M8 6V4c0-1 1-2 2-2h4c1 0 2 1 2 2v2"/></svg>';
  const iconZap = '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M4 14a1 1 0 0 1-.78-1.63l9.9-10.2a.5.5 0 0 1 .86.46l-1.92 6.02A1 1 0 0 0 13 10h7a1 1 0 0 1 .78 1.63l-9.9 10.2a.5.5 0 0 1-.86-.46l1.92-6.02A1 1 0 0 0 11 14z"/></svg>';

  let selectedId = $state(null);
  let presets = $state([]);
  let testResult = $state("");

  // Form fields
  let form = $state({
    id: null,
    preset: "",
    name: "",
    type: "openai-compatible",
    url: "",
    key: "",
    description: "",
    rag: false,
    active: true,
    temperature: 0.7,
    numCtx: 1024,
  });

  // Load presets on mount
  $effect(() => {
    loadPresets();
  });

  async function loadPresets() {
    try {
      const result = await bridge.listProviderProfiles();
      presets = result || [];
    } catch (e) {
      presets = [];
    }
  }

  function selectProvider(prov) {
    selectedId = prov.id;
    form = {
      id: prov.id,
      preset: "",
      name: prov.name || "",
      type: prov.type || "openai-compatible",
      url: prov.base_url || prov.url || "",
      key: prov.api_key || "",
      description: prov.description || "",
      rag: prov.use_in_rag || false,
      active: prov.active !== false,
      temperature: prov.temperature ?? 0.7,
      numCtx: prov.num_ctx ?? 1024,
    };
    testResult = "";
  }

  function newProvider() {
    selectedId = "new";
    form = {
      id: null, preset: "", name: "", type: "openai-compatible",
      url: "", key: "", description: "", rag: false, active: true,
      temperature: 0.7, numCtx: 1024,
    };
    testResult = "";
  }

  function applyPreset(e) {
    const p = presets.find((x) => x.id === e.target.value);
    if (!p) return;
    form.name = p.name || form.name;
    form.type = p.type || form.type;
    form.url = p.base_url || form.url;
    form.description = p.description || form.description;
  }

  async function handleSave() {
    const payload = {
      ID: form.id || 0,
      Name: form.name,
      Type: form.type,
      BaseURL: form.url,
      APIKey: form.key,
      Description: form.description,
      UseInRAG: form.rag,
      Active: form.active,
      Temperature: form.temperature,
      NumCtx: form.numCtx,
    };

    try {
      if (form.id) {
        await bridge.updateProvider(payload);
        providers.update((list) =>
          list.map((p) => (p.id === form.id ? { ...p, ...payload } : p)),
        );
      } else {
        const created = await bridge.createProvider(payload);
        if (created) {
          providers.update((list) => [...list, created]);
          selectedId = created.id;
          form.id = created.id;
        }
      }
    } catch (e) {
      console.error("Save failed:", e);
    }
  }

  async function handleDelete() {
    if (!form.id) return;
    try {
      await bridge.deleteProvider({ ID: form.id });
      providers.update((list) => list.filter((p) => p.id !== form.id));
      selectedId = null;
    } catch (e) {
      console.error("Delete failed:", e);
    }
  }

  async function handleTest() {
    testResult = "Testing…";
    try {
      const result = await bridge.testConnection({ ID: form.id || 0, BaseURL: form.url, APIKey: form.key, Type: form.type });
      testResult = result?.success ? "✓ Connected" : `✗ ${result?.error || "Failed"}`;
    } catch (e) {
      testResult = `✗ ${e.message || "Error"}`;
    }
  }
</script>

<section class="settings-section">
  <div class="providers-layout">
    <aside class="providers-list-panel">
      <div class="providers-list-header">
        <h4>Providers</h4>
        <button class="icon-btn" title="Add provider" onclick={newProvider}>
          {@html iconPlus}
        </button>
      </div>
      <div class="providers-list">
        {#each $providers as prov (prov.id)}
          <button
            class="provider-item"
            class:active={selectedId === prov.id}
            onclick={() => selectProvider(prov)}
          >
            {prov.name}
          </button>
        {/each}
      </div>
    </aside>

    <div class="provider-form-panel">
      {#if selectedId === null}
        <p class="placeholder">Select a provider or click + to add one.</p>
      {:else}
        <form class="provider-form" onsubmit={(e) => { e.preventDefault(); handleSave(); }}>
          <div class="field-group">
            <label for="pf-preset">Load preset</label>
            <select id="pf-preset" onchange={applyPreset}>
              <option value="">— Custom / Local —</option>
              {#each presets as p}
                <option value={p.id}>{p.name}</option>
              {/each}
            </select>
          </div>

          <div class="field-group">
            <label for="pf-name">Name *</label>
            <input id="pf-name" bind:value={form.name} placeholder="My Ollama" required />
          </div>

          <div class="field-group">
            <label for="pf-type">Provider type</label>
            <select id="pf-type" bind:value={form.type}>
              <option value="openai-compatible">OpenAI-compatible</option>
              <option value="ollama">Ollama</option>
              <option value="gemini">Gemini</option>
              <option value="anthropic">Anthropic</option>
              <option value="mistral">Mistral</option>
            </select>
          </div>

          <div class="field-group">
            <label for="pf-url">Base URL</label>
            <input id="pf-url" bind:value={form.url} placeholder="http://localhost:11434/v1" />
          </div>

          <div class="field-group">
            <label for="pf-key">API Key (optional)</label>
            <input id="pf-key" type="password" bind:value={form.key} placeholder="sk-..." />
          </div>

          <div class="field-group">
            <label for="pf-description">Description</label>
            <textarea id="pf-description" rows="2" bind:value={form.description} placeholder="What this provider is for..."></textarea>
          </div>

          <div class="field-row">
            <label class="check-label">
              <input type="checkbox" bind:checked={form.rag} /> Use in RAG
            </label>
            <label class="check-label">
              <input type="checkbox" bind:checked={form.active} /> Active
            </label>
          </div>

          <div class="field-group">
            <label for="pf-temperature">Temperature</label>
            <input id="pf-temperature" type="number" min="0" max="2" step="0.1" bind:value={form.temperature} />
          </div>

          <div class="field-group">
            <label for="pf-num-ctx">Context window (num_ctx)</label>
            <input id="pf-num-ctx" type="number" min="128" max="131072" step="128" bind:value={form.numCtx} />
          </div>

          <div class="test-row">
            <button type="button" class="action-btn secondary" title="Test connection" onclick={handleTest}>
              {@html iconZap}
            </button>
            {#if testResult}
              <span class="test-result" class:ok={testResult.startsWith("✓")} class:fail={testResult.startsWith("✗")}>
                {testResult}
              </span>
            {/if}
          </div>

          <div class="form-actions">
            {#if form.id}
              <button type="button" class="danger-btn" title="Delete provider" onclick={handleDelete}>
                {@html iconTrash}
              </button>
            {/if}
            <button type="submit" class="action-btn" title="Save provider">
              {@html iconCheck}
            </button>
          </div>
        </form>
      {/if}
    </div>
  </div>
</section>

<style>
  .providers-layout {
    display: grid;
    grid-template-columns: 180px 1fr;
    gap: 1rem;
    min-height: 300px;
  }

  .providers-list-panel {
    border-right: 1px solid var(--border-default, #2f4f6b);
    padding-right: 0.75rem;
  }

  .providers-list-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 0.5rem;
  }

  .providers-list-header h4 {
    margin: 0;
    font-size: 0.9rem;
    color: var(--text-primary, #E9E3D5);
  }

  .icon-btn {
    border: none;
    background: transparent;
    color: var(--text-secondary, #beb8ad);
    cursor: pointer;
    display: inline-flex;
    width: 26px;
    height: 26px;
    align-items: center;
    justify-content: center;
    padding: 0;
    border-radius: 6px;
  }

  .icon-btn:hover { color: var(--text-primary, #E9E3D5); }
  .icon-btn :global(svg) { width: 16px; height: 16px; }

  .providers-list {
    display: flex;
    flex-direction: column;
    gap: 0.25rem;
  }

  .provider-item {
    border: none;
    background: transparent;
    color: var(--text-secondary, #beb8ad);
    text-align: left;
    padding: 0.4rem 0.5rem;
    border-radius: 8px;
    cursor: pointer;
    font-size: 0.84rem;
  }

  .provider-item:hover { background: var(--surface-item-hover, #253d52); color: var(--text-primary, #E9E3D5); }
  .provider-item.active { background: var(--surface-item-active, #1e3d35); color: var(--text-primary, #E9E3D5); }

  .placeholder {
    color: var(--text-secondary, #beb8ad);
    font-size: 0.87rem;
    padding: 1rem;
  }

  .provider-form {
    display: flex;
    flex-direction: column;
    gap: 0.75rem;
  }

  .field-group { display: flex; flex-direction: column; gap: 0.25rem; }
  .field-group label { color: var(--text-secondary, #beb8ad); font-size: 0.82rem; }

  .field-group input,
  .field-group select,
  .field-group textarea {
    border: none;
    border-radius: 8px;
    background: var(--surface-input, #122033);
    color: var(--text-primary, #E9E3D5);
    padding: 0.4rem 0.5rem;
    font-size: 0.85rem;
  }

  .field-group textarea { resize: vertical; }

  .field-row {
    display: flex;
    gap: 1.5rem;
  }

  .check-label {
    display: flex;
    align-items: center;
    gap: 0.4rem;
    color: var(--text-primary, #E9E3D5);
    font-size: 0.84rem;
    cursor: pointer;
  }

  .test-row {
    display: flex;
    align-items: center;
    gap: 0.5rem;
  }

  .test-result { font-size: 0.82rem; }
  .test-result.ok { color: var(--ok-default, #45998A); }
  .test-result.fail { color: var(--danger-default, #e74c3c); }

  .form-actions {
    display: flex;
    justify-content: flex-end;
    gap: 0.5rem;
    padding-top: 0.5rem;
    border-top: 1px solid var(--border-default, #2f4f6b);
  }

  .action-btn {
    border: none;
    background: var(--surface-item, #1e2d3d);
    color: var(--text-primary, #E9E3D5);
    padding: 0.4rem 0.6rem;
    border-radius: 8px;
    cursor: pointer;
    display: inline-flex;
    align-items: center;
    gap: 0.25rem;
    font-size: 0.82rem;
  }

  .action-btn:hover { background: var(--surface-item-hover, #253d52); }
  .action-btn :global(svg) { width: 16px; height: 16px; }

  .danger-btn {
    border: none;
    background: transparent;
    color: var(--danger-default, #e74c3c);
    padding: 0.4rem 0.6rem;
    border-radius: 8px;
    cursor: pointer;
    display: inline-flex;
    align-items: center;
    font-size: 0.82rem;
  }

  .danger-btn:hover { background: rgba(231, 76, 60, 0.15); }
  .danger-btn :global(svg) { width: 16px; height: 16px; }
</style>
