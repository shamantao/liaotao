<!--
  ConversationList.svelte -- Conversations sidebar block.
  Responsibilities: display filtered conversation list grouped by date
  (year → month → day), context menus, rename inline.
-->
<script>
  import { conversations, filteredConversations, activeConversationId, setActiveConversation } from "../stores/chat.js";
  import { conversationsExpanded } from "../stores/app.js";
  import ContextMenu from "./ContextMenu.svelte";
  import ConversationItem from "./ConversationItem.svelte";
  import * as bridge from "../lib/bridge.js";

  // Icons (Lucide SVG inline)
  const iconPlus = '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M5 12h14"/><path d="M12 5v14"/></svg>';
  const iconPencil = '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M21.174 6.812a1 1 0 0 0-3.986-3.987L3.842 16.174a2 2 0 0 0-.5.83l-1.321 4.352a.5.5 0 0 0 .623.622l4.353-1.32a2 2 0 0 0 .83-.497z"/><path d="m15 5 4 4"/></svg>';
  const iconTrash = '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M3 6h18"/><path d="M19 6v14c0 1-1 2-2 2H7c-1 0-2-1-2-2V6"/><path d="M8 6V4c0-1 1-2 2-2h4c1 0 2 1 2 2v2"/></svg>';
  const iconExport = '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/><polyline points="7 10 12 15 17 10"/><line x1="12" x2="12" y1="15" y2="3"/></svg>';
  const iconChevron = '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="m6 9 6 6 6-6"/></svg>';

  // Context menu state
  let contextMenu = $state(null);

  // Rename state
  let renamingId = $state(null);
  let renameValue = $state("");

  function toggleExpanded() {
    conversationsExpanded.update((v) => !v);
  }

  async function handleNewConversation() {
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

  function openContextMenu(e, conv) {
    e.preventDefault();
    e.stopPropagation();
    contextMenu = {
      x: e.clientX,
      y: e.clientY,
      items: [
        { label: "Rename", icon: iconPencil, action: () => startRename(conv) },
        { label: "Export", icon: iconExport, action: () => handleExport(conv) },
        { separator: true },
        { label: "Delete", icon: iconTrash, danger: true, action: () => handleDelete(conv) },
      ],
    };
  }

  function startRename(conv) {
    renamingId = conv.id;
    renameValue = conv.title || "";
  }

  async function commitRename(convId) {
    if (!renameValue.trim()) { renamingId = null; return; }
    try {
      await bridge.renameConversation({ ConversationID: convId, Title: renameValue.trim() });
      conversations.update((list) =>
        list.map((c) => (c.id === convId ? { ...c, title: renameValue.trim() } : c)),
      );
    } catch (e) {
      console.error("Rename failed:", e);
    }
    renamingId = null;
  }

  async function handleDelete(conv) {
    try {
      await bridge.deleteConversation(conv.id);
      conversations.update((list) => list.filter((c) => c.id !== conv.id));
      activeConversationId.update((id) => (id === conv.id ? null : id));
    } catch (e) {
      console.error("Delete failed:", e);
    }
  }

  async function handleExport(conv) {
    try {
      await bridge.exportConversation({ ConversationID: conv.id, Format: "markdown" });
    } catch (e) {
      console.error("Export failed:", e);
    }
  }

  // --- Date hierarchy (year → month → day) ---
  const monthNames = ["Jan", "Feb", "Mar", "Apr", "May", "Jun", "Jul", "Aug", "Sep", "Oct", "Nov", "Dec"];
  let expandedDateGroups = $state(new Set());

  function toggleDateGroup(key) {
    expandedDateGroups = new Set(expandedDateGroups);
    if (expandedDateGroups.has(key)) expandedDateGroups.delete(key);
    else expandedDateGroups.add(key);
  }

  function buildDateHierarchy(convs) {
    const map = new Map();
    for (const conv of convs) {
      const d = new Date(conv.updated_at || conv.created_at);
      const y = d.getFullYear(), m = d.getMonth(), day = d.getDate();
      if (!map.has(y)) map.set(y, new Map());
      const yMap = map.get(y);
      if (!yMap.has(m)) yMap.set(m, new Map());
      const mMap = yMap.get(m);
      if (!mMap.has(day)) mMap.set(day, []);
      mMap.get(day).push(conv);
    }
    const result = [];
    for (const [year, months] of [...map.entries()].sort((a, b) => b[0] - a[0])) {
      const monthArr = [];
      for (const [month, days] of [...months.entries()].sort((a, b) => b[0] - a[0])) {
        const dayArr = [];
        for (const [day, convs2] of [...days.entries()].sort((a, b) => b[0] - a[0])) {
          dayArr.push({ day, conversations: convs2 });
        }
        monthArr.push({ month, monthLabel: monthNames[month], days: dayArr });
      }
      result.push({ year, months: monthArr });
    }
    return result;
  }

  // Auto-expand the most recent date group on first render
  let initialExpanded = $state(false);
  $effect(() => {
    const convs = $filteredConversations;
    if (!initialExpanded && convs.length > 0) {
      initialExpanded = true;
      const d = new Date(convs[0].updated_at || convs[0].created_at);
      const y = d.getFullYear(), pm = String(d.getMonth()).padStart(2, "0");
      const pd = String(d.getDate()).padStart(2, "0");
      expandedDateGroups = new Set([`${y}`, `${y}-${pm}`, `${y}-${pm}-${pd}`]);
    }
  });
</script>

<section class="sidebar-block">
  <div class="sidebar-block-head">
    <button class="sidebar-block-toggle" type="button" aria-expanded={$conversationsExpanded} onclick={toggleExpanded}>
      <span class="caret" class:expanded={$conversationsExpanded}>{@html iconChevron}</span>
      <span class="sidebar-block-label">Conversations</span>
    </button>
    <button class="head-menu-btn" type="button" title="New conversation" onclick={handleNewConversation}>
      {@html iconPlus}
    </button>
  </div>

  {#if $conversationsExpanded}
    <div class="sidebar-block-content">
      {#if $filteredConversations.length === 0}
        <p class="empty">No conversations</p>
      {:else}
        {#each buildDateHierarchy($filteredConversations) as yearGroup (yearGroup.year)}
          {@const yearKey = `${yearGroup.year}`}
          <div class="date-group">
            <button class="date-toggle" type="button" onclick={() => toggleDateGroup(yearKey)}>
              <span class="caret" class:expanded={expandedDateGroups.has(yearKey)}>{@html iconChevron}</span>
              <span class="date-label">{yearGroup.year}</span>
            </button>
            {#if expandedDateGroups.has(yearKey)}
              {#each yearGroup.months as monthGroup (monthGroup.month)}
                {@const monthKey = `${yearGroup.year}-${String(monthGroup.month).padStart(2, "0")}`}
                <div class="date-group date-sub">
                  <button class="date-toggle" type="button" onclick={() => toggleDateGroup(monthKey)}>
                    <span class="caret" class:expanded={expandedDateGroups.has(monthKey)}>{@html iconChevron}</span>
                    <span class="date-label">{monthGroup.monthLabel}</span>
                  </button>
                  {#if expandedDateGroups.has(monthKey)}
                    {#each monthGroup.days as dayGroup (dayGroup.day)}
                      {@const dayKey = `${monthKey}-${String(dayGroup.day).padStart(2, "0")}`}
                      <div class="date-group date-sub-2">
                        <button class="date-toggle" type="button" onclick={() => toggleDateGroup(dayKey)}>
                          <span class="caret" class:expanded={expandedDateGroups.has(dayKey)}>{@html iconChevron}</span>
                          <span class="date-label">{dayGroup.day}</span>
                        </button>
                        {#if expandedDateGroups.has(dayKey)}
                          {#each dayGroup.conversations as conv (conv.id)}
                            <ConversationItem
                              {conv}
                              active={$activeConversationId === conv.id}
                              renaming={renamingId === conv.id}
                              bind:renameValue
                              onselect={() => setActiveConversation(conv.id)}
                              oncontextmenu={(e) => openContextMenu(e, conv)}
                              oncommitRename={() => commitRename(conv.id)}
                              oncancelRename={() => { renamingId = null; }}
                            />
                          {/each}
                        {/if}
                      </div>
                    {/each}
                  {/if}
                </div>
              {/each}
            {/if}
          </div>
        {/each}
      {/if}
    </div>
  {/if}
</section>

{#if contextMenu}
  <ContextMenu items={contextMenu.items} x={contextMenu.x} y={contextMenu.y} onclose={() => (contextMenu = null)} />
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

  .caret.expanded { transform: rotate(0deg); }

  .caret :global(svg) {
    width: 14px;
    height: 14px;
  }

  .sidebar-block-label { font-size: 0.82rem; }

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

  .head-menu-btn:hover { color: var(--text-primary, #E9E3D5); }
  .head-menu-btn :global(svg) { width: 16px; height: 16px; }

  .sidebar-block-content {
    display: flex;
    flex-direction: column;
    gap: 0.25rem;
  }

  .date-group { display: flex; flex-direction: column; }
  .date-sub { padding-left: 0.75rem; }
  .date-sub-2 { padding-left: 0.75rem; }

  .date-toggle {
    border: none;
    background: transparent;
    color: var(--text-secondary, #beb8ad);
    cursor: pointer;
    display: flex;
    align-items: center;
    gap: 0.25rem;
    font-size: 0.75rem;
    font-weight: 500;
    padding: 0.2rem 0.25rem;
  }

  .date-toggle:hover { color: var(--text-primary, #E9E3D5); }
  .date-label { font-size: 0.75rem; }

  .empty {
    color: var(--text-secondary, #beb8ad);
    font-size: 0.82rem;
    margin: 0.25rem 0;
  }
</style>
