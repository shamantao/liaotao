<!--
  SettingsMCP.svelte -- MCP server management section.
  Responsibilities: list MCP servers, CRUD form with transport switching,
  ping test, tools list display, copy config.
-->
<script>
  import * as bridge from "../lib/bridge.js";

  // Icons
  const iconPlus = '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M5 12h14"/><path d="M12 5v14"/></svg>';
  const iconCheck = '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M20 6 9 17l-5-5"/></svg>';
  const iconTrash = '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M3 6h18"/><path d="M19 6v14c0 1-1 2-2 2H7c-1 0-2-1-2-2V6"/><path d="M8 6V4c0-1 1-2 2-2h4c1 0 2 1 2 2v2"/></svg>';
  const iconZap = '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M4 14a1 1 0 0 1-.78-1.63l9.9-10.2a.5.5 0 0 1 .86.46l-1.92 6.02A1 1 0 0 0 13 10h7a1 1 0 0 1 .78 1.63l-9.9 10.2a.5.5 0 0 1-.86-.46l1.92-6.02A1 1 0 0 0 11 14z"/></svg>';
  const iconCopy = '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect width="14" height="14" x="8" y="8" rx="2" ry="2"/><path d="M4 16c-1.1 0-2-.9-2-2V4c0-1.1.9-2 2-2h10c1.1 0 2 .9 2 2"/></svg>';

  let servers = $state([]);
  let selectedId = $state(null);
  let pingResult = $state("");
  let toolsList = $state([]);

  let form = $state({
    id: null,
    name: "",
    transport: "http",
    url: "",
    command: "",
    args: "",
    active: true,
  });

  let showUrlField = $derived(form.transport !== "stdio");
  let showCmdFields = $derived(form.transport === "stdio");

  $effect(() => {
    loadServers();
  });

  async function loadServers() {
    try {
      const result = await bridge.listMCPServers();
      servers = result || [];
    } catch (e) {
      servers = [];
    }
  }

  function selectServer(srv) {
    selectedId = srv.id;
    form = {
      id: srv.id,
      name: srv.name || "",
      transport: srv.transport || "http",
      url: srv.url || "",
      command: srv.command || "",
      args: srv.args ? JSON.stringify(srv.args) : "",
      active: srv.active !== false,
    };
    pingResult = "";
    toolsList = [];
  }

  function newServer() {
    selectedId = "new";
    form = { id: null, name: "", transport: "http", url: "", command: "", args: "", active: true };
    pingResult = "";
    toolsList = [];
  }

  async function handleSave() {
    let parsedArgs = [];
    if (form.args) {
      try {
        parsedArgs = JSON.parse(form.args);
      } catch {
        parsedArgs = [];
      }
    }

    const payload = {
      ID: form.id || 0,
      Name: form.name,
      Transport: form.transport,
      URL: form.url,
      Command: form.command,
      Args: parsedArgs,
      Active: form.active,
    };

    try {
      const saved = await bridge.saveMCPServer(payload);
      if (form.id) {
        servers = servers.map((s) => (s.id === form.id ? { ...s, ...payload } : s));
      } else if (saved) {
        servers = [...servers, saved];
        selectedId = saved.id;
        form.id = saved.id;
      }
    } catch (e) {
      console.error("Save failed:", e);
    }
  }

  async function handleDelete() {
    if (!form.id) return;
    try {
      await bridge.deleteMCPServer({ ID: form.id });
      servers = servers.filter((s) => s.id !== form.id);
      selectedId = null;
    } catch (e) {
      console.error("Delete failed:", e);
    }
  }

  async function handlePing() {
    pingResult = "Pinging…";
    try {
      const result = await bridge.pingMCPServer({ ID: form.id || 0, URL: form.url, Transport: form.transport });
      pingResult = result?.success ? "✓ Connected" : `✗ ${result?.error || "Failed"}`;
      if (result?.tools) {
        toolsList = result.tools;
      }
    } catch (e) {
      pingResult = `✗ ${e.message || "Error"}`;
    }
  }

  function handleCopyConfig() {
    const config = {
      name: form.name,
      transport: form.transport,
      url: form.url,
      command: form.command,
      args: form.args,
    };
    navigator.clipboard.writeText(JSON.stringify(config, null, 2));
  }
</script>

<section class="settings-section">
  <div class="providers-layout">
    <aside class="providers-list-panel">
      <div class="providers-list-header">
        <h4>MCP Servers</h4>
        <button class="icon-btn" title="Add MCP server" onclick={newServer}>
          {@html iconPlus}
        </button>
      </div>
      <div class="providers-list">
        {#each servers as srv (srv.id)}
          <button
            class="provider-item"
            class:active={selectedId === srv.id}
            onclick={() => selectServer(srv)}
          >
            {srv.name}
          </button>
        {/each}
      </div>
    </aside>

    <div class="provider-form-panel">
      {#if selectedId === null}
        <p class="placeholder">Select a server or click + to add one.</p>
      {:else}
        <form class="provider-form" onsubmit={(e) => { e.preventDefault(); handleSave(); }}>
          <div class="field-group">
            <label for="msf-name">Name *</label>
            <input id="msf-name" bind:value={form.name} placeholder="aitao local" required />
          </div>

          <div class="field-group">
            <label for="msf-transport">Transport *</label>
            <select id="msf-transport" bind:value={form.transport}>
              <option value="http">HTTP (Streamable HTTP / MCP 1.0)</option>
              <option value="sse">SSE (legacy FastMCP / aitao)</option>
              <option value="stdio">stdio (spawn process)</option>
            </select>
          </div>

          {#if showUrlField}
            <div class="field-group">
              <label for="msf-url">URL</label>
              <input id="msf-url" bind:value={form.url} placeholder="http://localhost:8201" />
            </div>
          {/if}

          {#if showCmdFields}
            <div class="field-group">
              <label for="msf-command">Command</label>
              <input id="msf-command" bind:value={form.command} placeholder="aitao" />
            </div>
            <div class="field-group">
              <label for="msf-args">Args (JSON array)</label>
              <input id="msf-args" bind:value={form.args} placeholder='["mcp", "stdio"]' />
            </div>
          {/if}

          <label class="check-label">
            <input type="checkbox" bind:checked={form.active} /> Active
          </label>

          <div class="test-row">
            <button type="button" class="action-btn secondary" title="Test connection" onclick={handlePing}>
              {@html iconZap}
            </button>
            <button type="button" class="action-btn secondary" title="Copy config" onclick={handleCopyConfig}>
              {@html iconCopy}
            </button>
            {#if pingResult}
              <span class="test-result" class:ok={pingResult.startsWith("✓")} class:fail={pingResult.startsWith("✗")}>
                {pingResult}
              </span>
            {/if}
          </div>

          {#if toolsList.length > 0}
            <div class="tools-list">
              <h5>Available tools ({toolsList.length})</h5>
              {#each toolsList as tool}
                <div class="tool-item">{tool.name || tool}</div>
              {/each}
            </div>
          {/if}

          <div class="form-actions">
            {#if form.id}
              <button type="button" class="danger-btn" title="Delete MCP server" onclick={handleDelete}>
                {@html iconTrash}
              </button>
            {/if}
            <button type="submit" class="action-btn" title="Save MCP server">
              {@html iconCheck}
            </button>
          </div>
        </form>
      {/if}
    </div>
  </div>
</section>

<style>
  .providers-layout {
    display: grid;
    grid-template-columns: 180px 1fr;
    gap: 1rem;
    min-height: 300px;
  }

  .providers-list-panel {
    border-right: 1px solid var(--border-default, #2f4f6b);
    padding-right: 0.75rem;
  }

  .providers-list-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 0.5rem;
  }

  .providers-list-header h4 { margin: 0; font-size: 0.9rem; color: var(--text-primary, #E9E3D5); }

  .icon-btn {
    border: none; background: transparent; color: var(--text-secondary, #beb8ad);
    cursor: pointer; display: inline-flex; width: 26px; height: 26px;
    align-items: center; justify-content: center; padding: 0; border-radius: 6px;
  }
  .icon-btn:hover { color: var(--text-primary, #E9E3D5); }
  .icon-btn :global(svg) { width: 16px; height: 16px; }

  .providers-list { display: flex; flex-direction: column; gap: 0.25rem; }

  .provider-item {
    border: none; background: transparent; color: var(--text-secondary, #beb8ad);
    text-align: left; padding: 0.4rem 0.5rem; border-radius: 8px;
    cursor: pointer; font-size: 0.84rem;
  }
  .provider-item:hover { background: var(--surface-item-hover, #253d52); color: var(--text-primary, #E9E3D5); }
  .provider-item.active { background: var(--surface-item-active, #1e3d35); color: var(--text-primary, #E9E3D5); }

  .placeholder { color: var(--text-secondary, #beb8ad); font-size: 0.87rem; padding: 1rem; }

  .provider-form { display: flex; flex-direction: column; gap: 0.75rem; }
  .field-group { display: flex; flex-direction: column; gap: 0.25rem; }
  .field-group label { color: var(--text-secondary, #beb8ad); font-size: 0.82rem; }

  .field-group input,
  .field-group select {
    border: none; border-radius: 8px;
    background: var(--surface-input, #122033); color: var(--text-primary, #E9E3D5);
    padding: 0.4rem 0.5rem; font-size: 0.85rem;
  }

  .check-label {
    display: flex; align-items: center; gap: 0.4rem;
    color: var(--text-primary, #E9E3D5); font-size: 0.84rem; cursor: pointer;
  }

  .test-row { display: flex; align-items: center; gap: 0.5rem; }
  .test-result { font-size: 0.82rem; }
  .test-result.ok { color: var(--ok-default, #45998A); }
  .test-result.fail { color: var(--danger-default, #e74c3c); }

  .tools-list {
    border: 1px solid var(--border-default, #2f4f6b);
    border-radius: 8px;
    padding: 0.5rem;
  }
  .tools-list h5 { margin: 0 0 0.25rem; font-size: 0.8rem; color: var(--text-secondary, #beb8ad); }
  .tool-item { font-size: 0.78rem; color: var(--text-primary, #E9E3D5); padding: 0.15rem 0; }

  .form-actions {
    display: flex; justify-content: flex-end; gap: 0.5rem;
    padding-top: 0.5rem; border-top: 1px solid var(--border-default, #2f4f6b);
  }

  .action-btn {
    border: none; background: var(--surface-item, #1e2d3d); color: var(--text-primary, #E9E3D5);
    padding: 0.4rem 0.6rem; border-radius: 8px; cursor: pointer;
    display: inline-flex; align-items: center; gap: 0.25rem; font-size: 0.82rem;
  }
  .action-btn:hover { background: var(--surface-item-hover, #253d52); }
  .action-btn :global(svg) { width: 16px; height: 16px; }

  .danger-btn {
    border: none; background: transparent; color: var(--danger-default, #e74c3c);
    padding: 0.4rem 0.6rem; border-radius: 8px; cursor: pointer;
    display: inline-flex; align-items: center; font-size: 0.82rem;
  }
  .danger-btn:hover { background: rgba(231, 76, 60, 0.15); }
  .danger-btn :global(svg) { width: 16px; height: 16px; }
</style>
