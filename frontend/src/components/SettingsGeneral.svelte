<!--
  SettingsGeneral.svelte -- General settings section.
  Responsibilities: language, font size, expert mode, meta footer,
  auto-check updates, default system prompt, config export/import.
  Organized into visual groups: General, Display, Updates, System.
-->
<script>
  import { settings, updateSettings, FONT_SIZE_MAP } from "../stores/settings.js";
  import { themes, activeThemeId, applyTheme } from "../stores/theme.js";
  import * as bridge from "../lib/bridge.js";

  let importInput;

  function handleChange(key, value) {
    updateSettings({ [key]: value });
  }

  async function exportConfig() {
    try {
      await bridge.exportConfigurationToFile();
    } catch (e) {
      console.error("Export failed:", e);
    }
  }

  function triggerImport() {
    importInput?.click();
  }

  async function handleImport(e) {
    const file = e.target.files?.[0];
    if (!file) return;
    try {
      const text = await file.text();
      await bridge.importConfiguration({ Content: text });
      location.reload();
    } catch (err) {
      console.error("Import failed:", err);
    }
  }
</script>

<section class="settings-section">
  <h3>General</h3>

  <!-- Group: General -->
  <div class="settings-group">
    <h4 class="group-title">General</h4>

    <div class="field-group">
      <label for="language">Language</label>
      <select
        id="language"
        value={$settings.language}
        onchange={(e) => handleChange("language", e.target.value)}
      >
        <option value="en">English</option>
        <option value="fr">Français</option>
        <option value="zh-TW">繁體中文</option>
      </select>
    </div>

    <div class="field-group">
      <span class="field-label">Response footer</span>
      <label class="checkbox-row">
        <input
          type="checkbox"
          checked={$settings.showMetaFooter}
          onchange={(e) => handleChange("showMetaFooter", e.target.checked)}
        />
        <span>Show model &amp; token info after each response</span>
      </label>
    </div>
  </div>

  <!-- Group: Display -->
  <div class="settings-group">
    <h4 class="group-title">Display</h4>

    <div class="field-group">
      <label for="theme-select">Theme</label>
      <select
        id="theme-select"
        value={$activeThemeId}
        onchange={(e) => applyTheme(e.target.value)}
      >
        {#each $themes as t}
          <option value={t.id}>{t.label}</option>
        {/each}
      </select>
    </div>

    <div class="field-group">
      <label for="chat-font-size">Font size</label>
      <select
        id="chat-font-size"
        value={$settings.chatFontSize}
        onchange={(e) => handleChange("chatFontSize", e.target.value)}
      >
        {#each Object.keys(FONT_SIZE_MAP) as size}
          <option value={size}>{size.toUpperCase()}</option>
        {/each}
      </select>
    </div>
  </div>

  <!-- Group: Updates -->
  <div class="settings-group">
    <h4 class="group-title">Updates</h4>

    <div class="field-group">
      <span class="field-label">Auto-check for updates</span>
      <label class="checkbox-row">
        <input
          type="checkbox"
          checked={$settings.autoCheckUpdates}
          onchange={(e) => handleChange("autoCheckUpdates", e.target.checked)}
        />
        <span>Check for new versions on startup</span>
      </label>
    </div>
  </div>

  <!-- Group: System -->
  <div class="settings-group">
    <h4 class="group-title">System</h4>

    <div class="field-group">
      <span class="field-label">Expert mode</span>
      <label class="checkbox-row">
        <input
          type="checkbox"
          checked={$settings.expertMode}
          onchange={(e) => handleChange("expertMode", e.target.checked)}
        />
        <span>Show advanced prompt controls in chat</span>
      </label>
    </div>

    <div class="field-group">
      <label for="default-system-prompt">Default system prompt</label>
      <textarea
        id="default-system-prompt"
        rows="3"
        placeholder="Used for new conversations"
        value={$settings.defaultSystemPrompt}
        onchange={(e) => handleChange("defaultSystemPrompt", e.target.value)}
      ></textarea>
    </div>

    <div class="field-group">
      <span class="field-label">Configuration</span>
      <div class="btn-row">
        <button type="button" class="action-btn" onclick={exportConfig}>Export TOML</button>
        <button type="button" class="action-btn" onclick={triggerImport}>Import TOML</button>
        <input
          bind:this={importInput}
          type="file"
          accept=".toml,text/plain"
          class="hidden"
          onchange={handleImport}
        />
      </div>
    </div>
  </div>
</section>

<style>
  .settings-section h3 {
    margin: 0 0 1rem;
    font-size: 1.1rem;
    color: var(--text-primary, #E9E3D5);
  }

  .settings-group {
    margin-bottom: 1.5rem;
    padding-bottom: 1rem;
    border-bottom: 1px solid var(--border-default, #2a3a4a);
  }

  .settings-group:last-child {
    border-bottom: none;
    margin-bottom: 0;
    padding-bottom: 0;
  }

  .group-title {
    margin: 0 0 0.75rem;
    font-size: 0.78rem;
    font-weight: 600;
    text-transform: uppercase;
    letter-spacing: 0.05em;
    color: var(--accent-default, #45998A);
  }

  .field-group {
    margin-bottom: 1rem;
  }

  .field-group label,
  .field-label {
    display: block;
    margin-bottom: 0.25rem;
    color: var(--text-secondary, #beb8ad);
    font-size: 0.85rem;
  }

  .field-group select,
  .field-group textarea {
    width: 100%;
    border: none;
    border-radius: 9px;
    background: var(--surface-input, #122033);
    color: var(--text-primary, #E9E3D5);
    padding: 0.5rem;
    font-size: 0.87rem;
  }

  .field-group textarea {
    resize: vertical;
  }

  .checkbox-row {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    cursor: pointer;
    font-size: 0.85rem;
    color: var(--text-primary, #E9E3D5);
  }

  .btn-row {
    display: flex;
    gap: 0.5rem;
    flex-wrap: wrap;
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

  .action-btn:hover {
    background: var(--surface-item-hover, #253d52);
  }

  .hidden {
    display: none;
  }
</style>
