<!--
  Sidebar.svelte -- Main sidebar component.
  Responsibilities: search input, GroupList, ConversationList,
  collapse/expand toggle, resize handle.
-->
<script>
  import { sidebarCollapsed, toggleSidebar, sidebarWidth, expandedSidebarWidth } from "../stores/app.js";
  import { conversationSearchQuery, conversations, setActiveConversation } from "../stores/chat.js";
  import GroupList from "./GroupList.svelte";
  import ConversationList from "./ConversationList.svelte";
  import * as bridge from "../lib/bridge.js";

  // Icons
  const iconMenu = '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M4 12h16"/><path d="M4 6h16"/><path d="M4 18h16"/></svg>';
  const iconPlus = '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M5 12h14"/><path d="M12 5v14"/></svg>';
  const iconSearch = '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="11" cy="11" r="8"/><path d="m21 21-4.3-4.3"/></svg>';

  let searchValue = $state("");
  let resizing = $state(false);

  function handleSearch(e) {
    searchValue = e.target.value;
    conversationSearchQuery.set(searchValue);
  }

  function handleSearchKeydown(e) {
    if (e.key === "Escape") {
      searchValue = "";
      conversationSearchQuery.set("");
    }
  }

  async function handleNewChat() {
    try {
      const conv = await bridge.createConversation({ Title: "", ProjectID: 0, ProviderID: 0, Model: "" });
      if (conv) {
        conversations.update((list) => [conv, ...list]);
        setActiveConversation(conv.id);
      }
    } catch (e) {
      console.error("Failed to create conversation:", e);
    }
  }

  // Resize handling
  function startResize(e) {
    resizing = true;
    const startX = e.clientX;
    let startWidth;
    sidebarWidth.subscribe((v) => (startWidth = v))();

    function onMove(ev) {
      const newWidth = Math.max(180, Math.min(460, startWidth + ev.clientX - startX));
      sidebarWidth.set(newWidth);
      expandedSidebarWidth.set(newWidth);
    }

    function onUp() {
      resizing = false;
      document.removeEventListener("mousemove", onMove);
      document.removeEventListener("mouseup", onUp);
    }

    document.addEventListener("mousemove", onMove);
    document.addEventListener("mouseup", onUp);
  }
</script>

<div class="sidebar-wrap" style="width: {$sidebarCollapsed ? 72 : $sidebarWidth}px;">
  <aside class="sidebar" class:collapsed={$sidebarCollapsed}>
    <!-- Header -->
    <div class="sidebar-header">
      <div class="sidebar-header-actions">
        <button class="icon-btn" title="Toggle sidebar" onclick={toggleSidebar}>
          {@html iconMenu}
        </button>
        {#if !$sidebarCollapsed}
          <button class="icon-btn new-chat" type="button" title="New chat" onclick={handleNewChat}>
            {@html iconPlus}
          </button>
        {/if}
      </div>
    </div>

    {#if $sidebarCollapsed}
      <!-- Collapsed rail: quick actions -->
      <div class="sidebar-quick-btns">
        <button class="icon-btn sidebar-quick-btn" title="Search" onclick={() => { sidebarCollapsed.set(false); }}>
          {@html iconSearch}
        </button>
        <button class="icon-btn sidebar-quick-btn" title="New chat" onclick={handleNewChat}>
          {@html iconPlus}
        </button>
      </div>
    {:else}
      <!-- Search -->
      <input
        class="conversation-search"
        type="search"
        placeholder="Search conversations"
        value={searchValue}
        oninput={handleSearch}
        onkeydown={handleSearchKeydown}
      />

      <!-- Groups -->
      <GroupList />

      <!-- Conversations -->
      <ConversationList />
    {/if}
  </aside>

  <!-- Resize handle -->
  {#if !$sidebarCollapsed}
    <!-- svelte-ignore a11y_no_noninteractive_tabindex -->
    <!-- svelte-ignore a11y_no_noninteractive_element_interactions -->
    <div
      class="sidebar-resizer"
      role="separator"
      aria-orientation="vertical"
      tabindex="0"
      title="Resize sidebar"
      onmousedown={startResize}
    ></div>
  {/if}
</div>

<style>
  .sidebar-wrap {
    position: relative;
    min-width: 72px;
    max-width: 460px;
    background: var(--surface-secondary, #1f2f43);
    display: flex;
    flex-shrink: 0;
    transition: width 140ms ease;
  }

  .sidebar {
    width: 100%;
    padding: 0.75rem;
    overflow-y: auto;
    display: flex;
    flex-direction: column;
    gap: 0.5rem;
  }

  .sidebar.collapsed {
    align-items: center;
    padding: 0.75rem 0.25rem;
  }

  .sidebar-header {
    display: flex;
    align-items: center;
    justify-content: flex-start;
    gap: 0.5rem;
  }

  .sidebar-header-actions {
    display: flex;
    align-items: center;
    gap: 0.25rem;
  }

  .icon-btn {
    border: none;
    background: transparent;
    color: var(--text-primary, #E9E3D5);
    cursor: pointer;
    display: inline-flex;
    align-items: center;
    justify-content: center;
    width: 32px;
    height: 32px;
    border-radius: 8px;
    padding: 0;
  }

  .icon-btn:hover {
    background: var(--surface-item-hover, #253d52);
  }

  .icon-btn :global(svg) {
    width: 16px;
    height: 16px;
  }

  .sidebar-quick-btns {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 0.5rem;
    padding-top: 0.5rem;
  }

  .sidebar-quick-btn {
    width: 44px;
    height: 44px;
    border-radius: 999px;
    background: var(--surface-item, #1e2d3d);
  }

  .sidebar-quick-btn :global(svg) {
    width: 20px;
    height: 20px;
  }

  .conversation-search {
    width: 100%;
    border: none;
    border-radius: 9px;
    background: var(--surface-input, #122033);
    color: var(--text-primary, #E9E3D5);
    padding: 0.5rem;
    font-size: 0.82rem;
  }

  .conversation-search::placeholder {
    color: var(--text-secondary, #beb8ad);
  }

  .sidebar-resizer {
    width: 4px;
    cursor: col-resize;
    background: transparent;
    transition: background 120ms ease;
  }

  .sidebar-resizer:hover {
    background: var(--accent-default, #45998A);
  }
</style>
