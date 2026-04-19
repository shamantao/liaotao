<!--
  Topbar.svelte -- Application top bar component.
  Responsibilities: brand logo + name, Chat/Settings tab navigation,
  plugin topbar action indicators, status indicator.
-->
<script>
  import { activeTab, switchTab } from "../stores/app.js";
  import { topbarActions } from "../stores/plugins.js";

  const tabs = [
    { id: "chat", label: "Chat" },
    { id: "settings", label: "Settings" },
  ];
</script>

<header class="topbar">
  <div class="brand">
    <img class="brand-logo" src="./assets/logo-liaotao.svg" alt="Liaotao logo">
    <span class="brand-name">liaotao</span>
  </div>
  <nav class="tabs">
    {#each tabs as tab}
      <button
        class="tab-btn"
        class:active={$activeTab === tab.id}
        onclick={() => switchTab(tab.id)}
      >
        {tab.label}
      </button>
    {/each}
  </nav>
  <div class="topbar-actions">
    {#each $topbarActions as action (action.id)}
      <span class="topbar-indicator" title={action.tooltip || action.label}>
        <span class="indicator-dot" style="background: {action.color}"></span>
        <span class="indicator-label">{action.label}</span>
      </span>
    {/each}
  </div>
  <div class="status">ready</div>
</header>

<style>
  .topbar {
    display: grid;
    grid-template-columns: 1fr auto auto auto;
    align-items: center;
    gap: 1rem;
    padding: 0.75rem 1rem;
    background: var(--surface-secondary, #141414);
    min-height: 58px;
  }

  .brand {
    display: inline-flex;
    align-items: center;
    gap: 0.75rem;
    font-weight: 700;
    letter-spacing: 0.03em;
  }

  .brand-logo {
    width: 34px;
    height: 34px;
    object-fit: contain;
    filter: drop-shadow(0 4px 14px rgba(62, 142, 126, 0.22));
  }

  .brand-name {
    font-size: 1.02rem;
  }

  .tabs {
    display: flex;
    gap: 0.5rem;
  }

  .tab-btn {
    border: none;
    background: var(--surface-tab, #252525);
    color: var(--text-secondary, #beb8ad);
    font-size: 0.75rem;
    border-radius: 14px;
    padding: 0.5rem 0.75rem;
    cursor: pointer;
    transition: color 140ms ease, background-color 140ms ease, transform 140ms ease;
  }

  .tab-btn:hover {
    color: var(--text-primary, #E9E3D5);
    background: var(--surface-tab-hover, #2e2e2e);
    transform: translateY(-1px);
  }

  .tab-btn.active {
    background: var(--primary-default, #1E3D59);
    color: var(--text-primary, #E9E3D5);
    box-shadow: inset 0 -4px 0 var(--accent-default, #45998A);
  }

  .tab-btn:focus-visible {
    outline: none;
    box-shadow: 0 0 0 4px var(--accent-default, #45998A);
  }

  .status {
    font-family: ui-monospace, "SFMono-Regular", Menlo, Consolas, monospace;
    color: var(--ok-default, #45998A);
    font-size: 0.75rem;
    justify-self: end;
  }

  /* Plugin topbar action indicators */
  .topbar-actions {
    display: flex;
    gap: 0.75rem;
    align-items: center;
  }

  .topbar-indicator {
    display: inline-flex;
    align-items: center;
    gap: 0.35rem;
    cursor: default;
  }

  .indicator-dot {
    width: 8px;
    height: 8px;
    border-radius: 50%;
    flex-shrink: 0;
  }

  .indicator-label {
    font-size: 0.72rem;
    color: var(--text-secondary, #beb8ad);
    font-family: ui-monospace, "SFMono-Regular", Menlo, Consolas, monospace;
  }
</style>
