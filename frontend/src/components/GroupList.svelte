<!--
  GroupList.svelte -- Groups (projects) sidebar block.
  Responsibilities: display project/group list with expand/collapse,
  context menus for rename/archive/export, new group creation.
-->
<script>
  import { projects, activeProjectId } from "../stores/chat.js";
  import { groupsExpanded } from "../stores/app.js";
  import ContextMenu from "./ContextMenu.svelte";
  import * as bridge from "../lib/bridge.js";

  // Icons (Lucide SVG inline)
  const iconPlus = '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M5 12h14"/><path d="M12 5v14"/></svg>';
  const iconEllipsis = '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="1"/><circle cx="19" cy="12" r="1"/><circle cx="5" cy="12" r="1"/></svg>';
  const iconPencil = '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M21.174 6.812a1 1 0 0 0-3.986-3.987L3.842 16.174a2 2 0 0 0-.5.83l-1.321 4.352a.5.5 0 0 0 .623.622l4.353-1.32a2 2 0 0 0 .83-.497z"/><path d="m15 5 4 4"/></svg>';
  const iconTrash = '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M3 6h18"/><path d="M19 6v14c0 1-1 2-2 2H7c-1 0-2-1-2-2V6"/><path d="M8 6V4c0-1 1-2 2-2h4c1 0 2 1 2 2v2"/></svg>';
  const iconChevron = '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="m6 9 6 6 6-6"/></svg>';

  let contextMenu = $state(null);
  let renamingId = $state(null);
  let renameValue = $state("");

  function toggleExpanded() {
    groupsExpanded.update((v) => !v);
  }

  function selectProject(id) {
    activeProjectId.update((current) => (current === id ? 0 : id));
  }

  async function handleNewGroup() {
    const name = "New Group";
    try {
      const proj = await bridge.createProject({ Name: name, Description: "" });
      if (proj) {
        projects.update((list) => [...list, proj]);
      }
    } catch (e) {
      console.error("Failed to create group:", e);
    }
  }

  function openContextMenu(e, proj) {
    e.preventDefault();
    e.stopPropagation();
    contextMenu = {
      x: e.clientX,
      y: e.clientY,
      items: [
        { label: "Rename", icon: iconPencil, action: () => startRename(proj) },
        { separator: true },
        { label: "Archive", icon: iconTrash, danger: true, action: () => handleArchive(proj) },
      ],
    };
  }

  function startRename(proj) {
    renamingId = proj.id;
    renameValue = proj.name || "";
  }

  async function commitRename(projId) {
    if (!renameValue.trim()) {
      renamingId = null;
      return;
    }
    try {
      await bridge.renameProject({ ProjectID: projId, Name: renameValue.trim() });
      projects.update((list) =>
        list.map((p) => (p.id === projId ? { ...p, name: renameValue.trim() } : p)),
      );
    } catch (e) {
      console.error("Rename failed:", e);
    }
    renamingId = null;
  }

  function handleRenameKeydown(e, projId) {
    if (e.key === "Enter") {
      e.preventDefault();
      commitRename(projId);
    } else if (e.key === "Escape") {
      renamingId = null;
    }
  }

  async function handleArchive(proj) {
    try {
      await bridge.archiveProject({ ProjectID: proj.id, Archived: true });
      projects.update((list) => list.filter((p) => p.id !== proj.id));
      activeProjectId.update((id) => (id === proj.id ? 0 : id));
    } catch (e) {
      console.error("Archive failed:", e);
    }
  }
</script>

<section class="sidebar-block">
  <div class="sidebar-block-head">
    <button
      class="sidebar-block-toggle"
      type="button"
      aria-expanded={$groupsExpanded}
      onclick={toggleExpanded}
    >
      <span class="caret" class:expanded={$groupsExpanded}>{@html iconChevron}</span>
      <span class="sidebar-block-label">Groups</span>
    </button>
    <button class="head-menu-btn" type="button" title="New group" onclick={handleNewGroup}>
      {@html iconPlus}
    </button>
  </div>

  {#if $groupsExpanded}
    <div class="sidebar-block-content">
      {#if $projects.length === 0}
        <p class="empty">No groups</p>
      {:else}
        {#each $projects as proj (proj.id)}
          <div
            class="group-item"
            class:active={$activeProjectId === proj.id}
            role="button"
            tabindex="0"
            onclick={() => selectProject(proj.id)}
            onkeydown={(e) => { if (e.key === 'Enter' || e.key === ' ') { e.preventDefault(); selectProject(proj.id); } }}
            oncontextmenu={(e) => openContextMenu(e, proj)}
          >
            {#if renamingId === proj.id}
              <input
                class="rename-input"
                type="text"
                bind:value={renameValue}
                onblur={() => commitRename(proj.id)}
                onkeydown={(e) => handleRenameKeydown(e, proj.id)}
              />
            {:else}
              <span class="group-name">{proj.name}</span>
              <button
                class="group-menu-btn"
                type="button"
                title="More"
                onclick={(e) => openContextMenu(e, proj)}
              >
                {@html iconEllipsis}
              </button>
            {/if}
          </div>
        {/each}
      {/if}
    </div>
  {/if}
</section>

{#if contextMenu}
  <ContextMenu
    items={contextMenu.items}
    x={contextMenu.x}
    y={contextMenu.y}
    onclose={() => (contextMenu = null)}
  />
{/if}

<style>
  .sidebar-block-head {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 0.25rem 0;
  }

  .sidebar-block-toggle {
    border: none;
    background: transparent;
    color: var(--text-primary, #E9E3D5);
    cursor: pointer;
    display: flex;
    align-items: center;
    gap: 0.5rem;
    font-size: 0.82rem;
    font-weight: 600;
    padding: 0.25rem;
  }

  .caret {
    display: inline-flex;
    width: 14px;
    height: 14px;
    transition: transform 140ms ease;
    transform: rotate(-90deg);
  }

  .caret.expanded {
    transform: rotate(0deg);
  }

  .caret :global(svg) {
    width: 14px;
    height: 14px;
  }

  .sidebar-block-label {
    font-size: 0.82rem;
  }

  .head-menu-btn {
    border: none;
    background: transparent;
    color: var(--text-secondary, #beb8ad);
    cursor: pointer;
    display: inline-flex;
    align-items: center;
    justify-content: center;
    width: 26px;
    height: 26px;
    border-radius: 7px;
    padding: 0;
  }

  .head-menu-btn:hover {
    color: var(--text-primary, #E9E3D5);
  }

  .head-menu-btn :global(svg) {
    width: 16px;
    height: 16px;
  }

  .sidebar-block-content {
    display: flex;
    flex-direction: column;
    gap: 0.25rem;
  }

  .empty {
    color: var(--text-secondary, #beb8ad);
    font-size: 0.82rem;
    margin: 0.25rem 0;
  }

  .group-item {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    padding: 0.5rem;
    border-radius: 10px;
    cursor: pointer;
    transition: background 120ms ease;
  }

  .group-item:hover {
    background: var(--surface-item-hover, #253d52);
  }

  .group-item.active {
    background: var(--surface-item-active, #1e3d35);
  }

  .group-name {
    flex: 1;
    font-size: 0.84rem;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .group-menu-btn {
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

  .group-item:hover .group-menu-btn,
  .group-item.active .group-menu-btn {
    opacity: 1;
  }

  .group-menu-btn :global(svg) {
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
</style>
