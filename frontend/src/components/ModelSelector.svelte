<!--
  ModelSelector.svelte -- Provider and model selector toolbar.
  Responsibilities: provider dropdown, model dropdown with filter,
  refresh models button, response style selector, expert-mode advanced fields.
-->
<script>
  import { providers, activeProviderId, lastUsedModels, recordModelUsage, modelFilterQuery } from "../stores/providers.js";
  import { settings } from "../stores/settings.js";
  import * as bridge from "../lib/bridge.js";

  // Icons
  const iconRefresh = '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M21 12a9 9 0 1 1-9-9c2.52 0 4.93 1 6.74 2.74L21 8"/><path d="M21 3v5h-5"/></svg>';

  let models = $state([]);
  let selectedModel = $state("");
  let temperature = $state(0.7);
  let maxTokens = $state(0);
  let systemPrompt = $state("");
  let responseStyle = $state("balanced");

  // Load models when provider changes
  $effect(() => {
    const pid = $activeProviderId;
    if (pid) {
      loadModels(pid);
    } else {
      models = [];
    }
  });

  // Sync response style from settings
  $effect(() => {
    responseStyle = $settings.responseStyle || "balanced";
  });

  async function loadModels(providerId) {
    try {
      const result = await bridge.listModels({ ProviderID: providerId });
      models = result || [];
      if (models.length > 0 && !selectedModel) {
        selectedModel = models[0].id || models[0].ID || "";
      }
    } catch (e) {
      console.error("Failed to load models:", e);
      models = [];
    }
  }

  async function refreshModels() {
    const pid = $activeProviderId;
    if (pid) await loadModels(pid);
  }

  function handleProviderChange(e) {
    const id = Number(e.target.value);
    activeProviderId.set(id || null);
    selectedModel = "";
  }

  function handleModelChange(e) {
    selectedModel = e.target.value;
  }

  // Expose selected values for parent
  export function getSelections() {
    return {
      providerId: $activeProviderId,
      model: selectedModel,
      temperature,
      maxTokens,
      systemPrompt,
      responseStyle,
    };
  }

  let filteredModels = $derived.by(() => {
    const q = $modelFilterQuery.toLowerCase();
    if (!q) return models;
    return models.filter((m) => {
      const id = m.id || m.ID || "";
      return id.toLowerCase().includes(q);
    });
  });
</script>

<div class="chat-toolbar">
  <label for="chat-provider">Provider</label>
  <select id="chat-provider" onchange={handleProviderChange} value={$activeProviderId || ""}>
    <option value="">— Select —</option>
    <option value="0">✦ Automat</option>
    {#each $providers as prov}
      <option value={prov.id}>{prov.name}</option>
    {/each}
  </select>

  <label for="chat-model">Model</label>
  <select id="chat-model" onchange={handleModelChange} value={selectedModel}>
    {#each filteredModels as m}
      <option value={m.id || m.ID}>{m.id || m.ID}</option>
    {/each}
  </select>

  <button class="action-btn icon-only-btn" type="button" title="Refresh models" onclick={refreshModels}>
    {@html iconRefresh}
  </button>

  <label for="chat-response-style">Response</label>
  <select id="chat-response-style" bind:value={responseStyle}>
    <option value="precise">Precise</option>
    <option value="balanced">Balanced</option>
    <option value="creative">Creative</option>
  </select>

  {#if $settings.expertMode}
    <label for="chat-model-filter">Filter</label>
    <input
      id="chat-model-filter"
      type="search"
      placeholder="Filter models"
      oninput={(e) => modelFilterQuery.set(e.target.value)}
    />

    <label for="chat-temperature">Temp</label>
    <input
      id="chat-temperature"
      type="number"
      min="0"
      max="2"
      step="0.1"
      bind:value={temperature}
    />

    <label for="chat-max-tokens">Max tokens</label>
    <input
      id="chat-max-tokens"
      type="number"
      min="0"
      max="131072"
      step="1"
      bind:value={maxTokens}
    />

    <input
      id="chat-system-prompt"
      type="text"
      placeholder="System prompt for this conversation"
      bind:value={systemPrompt}
    />
  {/if}
</div>

<!-- Last used models -->
{#if $lastUsedModels.length > 0}
  <div class="last-used">
    {#each $lastUsedModels as lm}
      <button
        class="last-used-btn"
        type="button"
        title="{lm.providerName} · {lm.model}"
        onclick={() => {
          activeProviderId.set(lm.providerId);
          selectedModel = lm.model;
        }}
      >
        {lm.model}
      </button>
    {/each}
  </div>
{/if}

<style>
  .chat-toolbar {
    padding: 0.25rem 0 0;
    display: flex;
    gap: 0.5rem;
    align-items: center;
    background: transparent;
    flex-wrap: wrap;
  }

  .chat-toolbar label {
    color: var(--text-secondary, #beb8ad);
    font-size: 0.83rem;
  }

  .chat-toolbar select {
    border: none;
    background: var(--surface-input, #122033);
    color: var(--text-primary, #E9E3D5);
    border-radius: 8px;
    padding: 0.25rem 0.5rem;
    min-width: 140px;
    max-width: 36vw;
  }

  .chat-toolbar input[type="search"],
  .chat-toolbar input[type="number"],
  .chat-toolbar input[type="text"] {
    border: none;
    background: var(--surface-input, #122033);
    color: var(--text-primary, #E9E3D5);
    border-radius: 8px;
    padding: 0.25rem 0.5rem;
    min-width: 120px;
  }

  #chat-system-prompt {
    min-width: 240px;
    flex: 1;
  }

  .action-btn {
    border: none;
    color: var(--text-secondary, #beb8ad);
    background: transparent;
    border-radius: 8px;
    cursor: pointer;
    display: inline-flex;
    align-items: center;
    justify-content: center;
    min-width: var(--chat-icon-size, 26px);
    width: var(--chat-icon-size, 26px);
    height: var(--chat-icon-size, 26px);
    padding: 0;
  }

  .action-btn:hover {
    color: var(--text-primary, #E9E3D5);
  }

  .action-btn :global(svg) {
    width: calc(var(--chat-icon-size, 26px) * 0.65);
    height: calc(var(--chat-icon-size, 26px) * 0.65);
  }

  .last-used {
    display: flex;
    gap: 0.25rem;
    flex-wrap: wrap;
    padding: 0.25rem 0;
  }

  .last-used-btn {
    border: none;
    background: var(--surface-item, #1e2d3d);
    color: var(--text-secondary, #beb8ad);
    border-radius: 8px;
    padding: 0.25rem 0.5rem;
    font-size: 0.72rem;
    cursor: pointer;
    white-space: nowrap;
    max-width: 140px;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .last-used-btn:hover {
    background: var(--surface-item-hover, #253d52);
    color: var(--text-primary, #E9E3D5);
  }
</style>
