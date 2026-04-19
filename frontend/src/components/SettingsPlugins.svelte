<!--
  SettingsPlugins.svelte -- Plugins and prompt library section.
  Responsibilities: list installed plugins with enable/disable,
  prompt template CRUD, conversation export buttons.
-->
<script>
  import * as bridge from "../lib/bridge.js";

  let plugins = $state([]);
  let promptList = $state([]);
  let promptName = $state("");
  let promptContent = $state("");
  let selectedPrompt = $state("");

  $effect(() => {
    loadPlugins();
    loadPrompts();
  });

  async function loadPlugins() {
    try {
      const result = await bridge.listPluginScripts();
      plugins = result || [];
    } catch (e) {
      plugins = [];
    }
  }

  async function loadPrompts() {
    try {
      const result = await bridge.getGeneralSettings();
      promptList = result?.prompt_templates || [];
    } catch (e) {
      promptList = [];
    }
  }

  async function reloadPlugins() {
    await loadPlugins();
  }

  function insertPrompt() {
    const tpl = promptList.find((p) => p.name === selectedPrompt);
    if (tpl) {
      promptName = tpl.name;
      promptContent = tpl.content;
    }
  }

  async function savePrompt() {
    if (!promptName.trim() || !promptContent.trim()) return;
    try {
      await bridge.updateGeneralSettings({
        prompt_template: { name: promptName.trim(), content: promptContent.trim() },
      });
      await loadPrompts();
      promptName = "";
      promptContent = "";
    } catch (e) {
      console.error("Save prompt failed:", e);
    }
  }

  async function deletePrompt() {
    if (!selectedPrompt) return;
    try {
      await bridge.updateGeneralSettings({
        delete_prompt_template: selectedPrompt,
      });
      await loadPrompts();
      selectedPrompt = "";
      promptName = "";
      promptContent = "";
    } catch (e) {
      console.error("Delete prompt failed:", e);
    }
  }

  async function exportMarkdown() {
    try {
      await bridge.exportConversation({ Format: "markdown" });
    } catch (e) {
      console.error("Export failed:", e);
    }
  }

  async function exportPDF() {
    try {
      await bridge.exportConversation({ Format: "pdf" });
    } catch (e) {
      console.error("Export failed:", e);
    }
  }
</script>

<section class="settings-section">
  <h3>Plugins</h3>

  <p class="help">A plugin can intercept messages or tool output. Enable only what you need for your workflow.</p>

  <div class="field-group">
    <span class="field-label">Installed plugins</span>
    <div class="btn-row">
      <button type="button" class="action-btn" onclick={reloadPlugins}>Reload plugins</button>
    </div>
    <div class="plugins-list">
      {#if plugins.length === 0}
        <p class="empty">No plugins installed</p>
      {:else}
        {#each plugins as plugin}
          <div class="plugin-item">
            <span class="plugin-name">{plugin.name}</span>
            <span class="plugin-status" class:active={plugin.enabled}>
              {plugin.enabled ? "enabled" : "disabled"}
            </span>
          </div>
        {/each}
      {/if}
    </div>
  </div>

  <div class="field-group">
    <span class="field-label">Prompt library</span>
    <p class="help">Default templates: summarize = concise 5-bullet summary, translate_fr = translate latest answer to French. Use /tpl &lt;name&gt; in chat.</p>
    <div class="prompt-editor">
      <input bind:value={promptName} placeholder="template_name" />
      <textarea rows="2" bind:value={promptContent} placeholder="Prompt template content"></textarea>
    </div>
    <div class="btn-row">
      <select bind:value={selectedPrompt}>
        <option value="">— Select template —</option>
        {#each promptList as p}
          <option value={p.name}>{p.name}</option>
        {/each}
      </select>
      <button type="button" class="action-btn" onclick={insertPrompt}>Insert</button>
      <button type="button" class="action-btn" onclick={savePrompt}>Save</button>
      <button type="button" class="danger-btn" onclick={deletePrompt}>Delete</button>
    </div>
  </div>

  <div class="field-group">
    <span class="field-label">Conversation export</span>
    <div class="btn-row">
      <button type="button" class="action-btn" onclick={exportMarkdown}>Export Markdown</button>
      <button type="button" class="action-btn" onclick={exportPDF}>Export PDF</button>
    </div>
  </div>
</section>

<style>
  .settings-section h3 { margin: 0 0 1rem; font-size: 1.1rem; color: var(--text-primary, #E9E3D5); }

  .help {
    color: var(--text-secondary, #beb8ad);
    font-size: 0.82rem;
    margin: 0 0 0.75rem;
  }

  .field-group { margin-bottom: 1.25rem; }
  .field-group label,
  .field-label { display: block; margin-bottom: 0.25rem; color: var(--text-secondary, #beb8ad); font-size: 0.85rem; }

  .plugins-list { margin-top: 0.5rem; }

  .empty {
    color: var(--text-secondary, #beb8ad);
    font-size: 0.82rem;
  }

  .plugin-item {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 0.35rem 0.5rem;
    border-radius: 8px;
    font-size: 0.84rem;
  }

  .plugin-name { color: var(--text-primary, #E9E3D5); }
  .plugin-status { color: var(--text-secondary, #beb8ad); font-size: 0.78rem; }
  .plugin-status.active { color: var(--ok-default, #45998A); }

  .prompt-editor {
    display: flex;
    flex-direction: column;
    gap: 0.5rem;
    margin-bottom: 0.5rem;
  }

  .prompt-editor input,
  .prompt-editor textarea {
    border: none;
    border-radius: 8px;
    background: var(--surface-input, #122033);
    color: var(--text-primary, #E9E3D5);
    padding: 0.4rem 0.5rem;
    font-size: 0.85rem;
  }

  .prompt-editor textarea { resize: vertical; }

  .btn-row {
    display: flex;
    gap: 0.5rem;
    flex-wrap: wrap;
    align-items: center;
  }

  .btn-row select {
    border: none;
    border-radius: 8px;
    background: var(--surface-input, #122033);
    color: var(--text-primary, #E9E3D5);
    padding: 0.35rem 0.5rem;
    font-size: 0.84rem;
    min-width: 140px;
  }

  .action-btn {
    border: none;
    background: var(--surface-item, #1e2d3d);
    color: var(--text-primary, #E9E3D5);
    padding: 0.4rem 0.75rem;
    border-radius: 8px;
    font-size: 0.82rem;
    cursor: pointer;
  }

  .action-btn:hover { background: var(--surface-item-hover, #253d52); }

  .danger-btn {
    border: none;
    background: transparent;
    color: var(--danger-default, #e74c3c);
    padding: 0.4rem 0.75rem;
    border-radius: 8px;
    cursor: pointer;
    font-size: 0.82rem;
  }

  .danger-btn:hover { background: rgba(231, 76, 60, 0.15); }
</style>
