<!--
  ContextMenu.svelte -- Reusable positioned context menu component.
  Responsibilities: display a floating menu at dynamic position,
  close on outside click or Escape. Used by sidebar items.
-->
<script>
  import { onMount } from "svelte";

  /** @type {{ label: string, icon?: string, action: () => void, danger?: boolean, separator?: boolean }[]} */
  let { items = [], x = 0, y = 0, onclose = () => {} } = $props();

  let menuEl;

  function handleClick(item) {
    if (item.separator) return;
    item.action();
    onclose();
  }

  function handleKeydown(e) {
    if (e.key === "Escape") onclose();
  }

  function handleOutsideClick(e) {
    if (menuEl && !menuEl.contains(e.target)) {
      onclose();
    }
  }

  onMount(() => {
    // Adjust position if menu would overflow viewport
    if (menuEl) {
      const rect = menuEl.getBoundingClientRect();
      if (rect.right > window.innerWidth) {
        menuEl.style.left = `${window.innerWidth - rect.width - 8}px`;
      }
      if (rect.bottom > window.innerHeight) {
        menuEl.style.top = `${window.innerHeight - rect.height - 8}px`;
      }
    }
    document.addEventListener("click", handleOutsideClick, true);
    document.addEventListener("keydown", handleKeydown);
    return () => {
      document.removeEventListener("click", handleOutsideClick, true);
      document.removeEventListener("keydown", handleKeydown);
    };
  });
</script>

<div
  class="context-menu"
  bind:this={menuEl}
  style="left: {x}px; top: {y}px;"
  role="menu"
>
  {#each items as item}
    {#if item.separator}
      <div class="menu-sep"></div>
    {:else}
      <button
        class="menu-item"
        class:danger={item.danger}
        role="menuitem"
        onclick={() => handleClick(item)}
      >
        {#if item.icon}
          <span class="menu-icon">{@html item.icon}</span>
        {/if}
        <span>{item.label}</span>
      </button>
    {/if}
  {/each}
</div>

<style>
  .context-menu {
    position: fixed;
    z-index: 300;
    background: var(--surface-elevated, #1a2e45);
    border-radius: 10px;
    padding: 0.25rem;
    display: flex;
    flex-direction: column;
    min-width: 130px;
    box-shadow: 0 8px 24px rgba(0, 0, 0, 0.55);
  }

  .menu-item {
    border: none;
    background: transparent;
    color: var(--text-primary, #E9E3D5);
    text-align: left;
    padding: 0.5rem 0.5rem;
    border-radius: 7px;
    font-size: 0.85rem;
    cursor: pointer;
    white-space: nowrap;
    display: flex;
    align-items: center;
    gap: 0.5rem;
    line-height: 1;
  }

  .menu-item:hover {
    background: rgba(62, 142, 126, 0.18);
  }

  .menu-item.danger {
    color: var(--danger-default, #cc6b64);
  }

  .menu-item.danger:hover {
    background: rgba(204, 107, 100, 0.18);
  }

  .menu-sep {
    height: 1px;
    background: var(--border-default, #2f4f6b);
    margin: 0.25rem 0.5rem;
  }

  .menu-icon {
    display: inline-flex;
    width: 14px;
    height: 14px;
  }

  .menu-icon :global(svg) {
    width: 14px;
    height: 14px;
  }
</style>
