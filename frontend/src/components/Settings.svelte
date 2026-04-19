<!--
  Settings.svelte -- Settings panel with navigation sections.
  Responsibilities: section navigation (General, Providers, MCP, Plugins, About),
  delegates each section to its own sub-component for maintainability.
-->
<script>
  import { settingsSection } from "../stores/app.js";
  import SettingsGeneral from "./SettingsGeneral.svelte";
  import SettingsProviders from "./SettingsProviders.svelte";
  import SettingsMCP from "./SettingsMCP.svelte";
  import SettingsPlugins from "./SettingsPlugins.svelte";
  import SettingsAbout from "./SettingsAbout.svelte";

  const sections = [
    { id: "general",   label: "General" },
    { id: "providers", label: "Providers" },
    { id: "mcp",       label: "MCP Servers" },
    { id: "plugins",   label: "Plugins" },
    { id: "about",     label: "About" },
  ];

  function switchSection(id) {
    settingsSection.set(id);
  }
</script>

<div class="settings-layout">
  <nav class="settings-nav">
    {#each sections as s}
      <button
        class="settings-nav-btn"
        class:active={$settingsSection === s.id}
        onclick={() => switchSection(s.id)}
      >
        {s.label}
      </button>
    {/each}
  </nav>

  <div class="settings-content">
    {#if $settingsSection === "general"}
      <SettingsGeneral />
    {:else if $settingsSection === "providers"}
      <SettingsProviders />
    {:else if $settingsSection === "mcp"}
      <SettingsMCP />
    {:else if $settingsSection === "plugins"}
      <SettingsPlugins />
    {:else if $settingsSection === "about"}
      <SettingsAbout />
    {/if}
  </div>
</div>

<style>
  .settings-layout {
    display: grid;
    grid-template-columns: 160px 1fr;
    gap: 1.5rem;
    padding: 1rem;
    height: 100%;
    overflow: hidden;
  }

  .settings-nav {
    display: flex;
    flex-direction: column;
    gap: 0.25rem;
    padding-top: 0.5rem;
  }

  .settings-nav-btn {
    border: none;
    background: transparent;
    color: var(--text-secondary, #beb8ad);
    font-size: 0.87rem;
    padding: 0.5rem 0.75rem;
    text-align: left;
    border-radius: 9px;
    cursor: pointer;
    transition: background 120ms ease;
  }

  .settings-nav-btn:hover {
    background: var(--surface-item-hover, #253d52);
    color: var(--text-primary, #E9E3D5);
  }

  .settings-nav-btn.active {
    background: var(--surface-tab, #233246);
    color: var(--text-primary, #E9E3D5);
    font-weight: 600;
  }

  .settings-content {
    overflow-y: auto;
    padding-right: 1rem;
  }
</style>
