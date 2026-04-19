<!--
  ConversationItem.svelte -- Single conversation entry in sidebar.
  Responsibilities: display conversation title, time, tags, inline rename,
  context menu trigger, selection highlight.
-->
<script>
  let {
    conv,
    active = false,
    renaming = false,
    renameValue = $bindable(""),
    onselect,
    oncontextmenu,
    oncommitRename,
    oncancelRename,
  } = $props();

  // Icons (Lucide SVG inline)
  const iconEllipsis = '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="1"/><circle cx="19" cy="12" r="1"/><circle cx="5" cy="12" r="1"/></svg>';

  function formatTime(dateStr) {
    if (!dateStr) return "";
    const d = new Date(dateStr);
    return d.toLocaleTimeString(undefined, { hour: "2-digit", minute: "2-digit" });
  }

  function handleKeydown(e) {
    if (e.key === "Enter" || e.key === " ") {
      e.preventDefault();
      onselect?.();
    }
  }

  function handleRenameKeydown(e) {
    if (e.key === "Enter") {
      e.preventDefault();
      oncommitRename?.();
    } else if (e.key === "Escape") {
      oncancelRename?.();
    }
  }
</script>

<div
  class="conversation-item"
  class:active
  role="button"
  tabindex="0"
  onclick={() => onselect?.()}
  onkeydown={handleKeydown}
  oncontextmenu={oncontextmenu}
>
  {#if renaming}
    <input
      class="rename-input"
      type="text"
      bind:value={renameValue}
      onblur={() => oncommitRename?.()}
      onkeydown={handleRenameKeydown}
    />
  {:else}
    <div class="conversation-main">
      <span class="conversation-title">{conv.title || "New conversation"}</span>
      <span class="conversation-meta">{formatTime(conv.updated_at)}</span>
    </div>
    <div class="conversation-actions">
      <button
        class="conv-menu-btn"
        type="button"
        title="More"
        onclick={oncontextmenu}
      >
        {@html iconEllipsis}
      </button>
    </div>
  {/if}
  {#if conv.tags && conv.tags.length > 0}
    <div class="tags-row">
      {#each conv.tags as tag}
        <span class="tag-pill" style="background: {tag.color || '#2a5271'}">{tag.name}</span>
      {/each}
    </div>
  {/if}
</div>

<style>
  .conversation-item {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    padding: 0.5rem;
    border-radius: 10px;
    cursor: pointer;
    flex-wrap: wrap;
    transition: background 120ms ease;
  }

  .conversation-item:hover {
    background: var(--surface-item-hover, #253d52);
  }

  .conversation-item.active {
    background: var(--surface-item-active, #1e3d35);
  }

  .conversation-main {
    flex: 1;
    min-width: 0;
    display: flex;
    flex-direction: column;
  }

  .conversation-title {
    font-size: 0.84rem;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .conversation-meta {
    font-size: 0.68rem;
    color: var(--text-secondary, #beb8ad);
    opacity: 0.8;
    line-height: 1.2;
  }

  .conversation-actions {
    display: inline-flex;
    gap: 0.25rem;
  }

  .conv-menu-btn {
    border: none;
    background: transparent;
    color: var(--text-secondary, #beb8ad);
    border-radius: 7px;
    width: 26px;
    height: 26px;
    display: inline-flex;
    align-items: center;
    justify-content: center;
    cursor: pointer;
    opacity: 0;
    transition: opacity 120ms ease;
    padding: 0;
  }

  .conversation-item:hover .conv-menu-btn,
  .conversation-item.active .conv-menu-btn {
    opacity: 1;
  }

  .conv-menu-btn :global(svg) {
    width: 16px;
    height: 16px;
  }

  .rename-input {
    width: 100%;
    border-radius: 8px;
    background: var(--surface-input, #122033);
    color: var(--text-primary, #E9E3D5);
    padding: 0.25rem 0.5rem;
    font-size: 0.84rem;
    border: none;
  }

  .tags-row {
    display: flex;
    flex-wrap: wrap;
    gap: 0.25rem;
    width: 100%;
  }

  .tag-pill {
    display: inline-block;
    padding: 0 0.25rem;
    border-radius: 8px;
    font-size: 0.62rem;
    color: var(--text-primary, #E9E3D5);
    opacity: 0.9;
    white-space: nowrap;
    max-width: 80px;
    overflow: hidden;
    text-overflow: ellipsis;
  }
</style>
