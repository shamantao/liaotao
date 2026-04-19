<!--
  ExportDialog.svelte -- Modal dialog for conversation export.
  Responsibilities: format selection (Markdown / PDF), optional scope,
  confirm/cancel actions with overlay backdrop.
-->
<script>
  import * as bridge from "../lib/bridge.js";

  let { open = false, conversationId = null, onclose = () => {} } = $props();

  let format = $state("markdown");
  let exporting = $state(false);

  async function handleExport() {
    if (exporting) return;
    exporting = true;
    try {
      await bridge.exportConversation({
        ConversationID: conversationId,
        Format: format,
      });
      onclose();
    } catch (e) {
      console.error("Export failed:", e);
    } finally {
      exporting = false;
    }
  }

  function handleKeydown(e) {
    if (e.key === "Escape") onclose();
  }

  function handleBackdropClick(e) {
    if (e.target === e.currentTarget) onclose();
  }
</script>

{#if open}
  <!-- svelte-ignore a11y_no_noninteractive_element_interactions -->
  <div
    class="dialog-backdrop"
    role="dialog"
    aria-modal="true"
    onkeydown={handleKeydown}
    onclick={handleBackdropClick}
  >
    <div class="dialog-content">
      <h3>Export conversation</h3>

      <div class="field-group">
        <label for="export-format">Format</label>
        <select id="export-format" bind:value={format}>
          <option value="markdown">Markdown (.md)</option>
          <option value="pdf">PDF</option>
        </select>
      </div>

      <div class="dialog-actions">
        <button class="cancel-btn" type="button" onclick={onclose}>Cancel</button>
        <button class="action-btn" type="button" onclick={handleExport} disabled={exporting}>
          {exporting ? "Exporting…" : "Export"}
        </button>
      </div>
    </div>
  </div>
{/if}

<style>
  .dialog-backdrop {
    position: fixed;
    inset: 0;
    background: rgba(0, 0, 0, 0.5);
    display: flex;
    align-items: center;
    justify-content: center;
    z-index: 1000;
  }

  .dialog-content {
    background: var(--surface-elevated, #1e3044);
    border-radius: 14px;
    padding: 1.25rem;
    min-width: 320px;
    max-width: 420px;
    box-shadow: 0 8px 32px rgba(0, 0, 0, 0.4);
  }

  .dialog-content h3 {
    margin: 0 0 1rem;
    font-size: 1rem;
    color: var(--text-primary, #E9E3D5);
  }

  .field-group {
    margin-bottom: 1rem;
  }

  .field-group label {
    display: block;
    margin-bottom: 0.25rem;
    color: var(--text-secondary, #beb8ad);
    font-size: 0.84rem;
  }

  .field-group select {
    width: 100%;
    border: none;
    border-radius: 9px;
    background: var(--surface-input, #122033);
    color: var(--text-primary, #E9E3D5);
    padding: 0.5rem;
    font-size: 0.87rem;
  }

  .dialog-actions {
    display: flex;
    justify-content: flex-end;
    gap: 0.5rem;
    padding-top: 0.5rem;
  }

  .cancel-btn {
    border: none;
    background: transparent;
    color: var(--text-secondary, #beb8ad);
    padding: 0.4rem 0.75rem;
    border-radius: 8px;
    cursor: pointer;
    font-size: 0.84rem;
  }

  .cancel-btn:hover {
    background: var(--surface-item-hover, #253d52);
    color: var(--text-primary, #E9E3D5);
  }

  .action-btn {
    border: none;
    background: var(--accent-default, #45998A);
    color: #fff;
    padding: 0.4rem 0.75rem;
    border-radius: 8px;
    cursor: pointer;
    font-size: 0.84rem;
  }

  .action-btn:hover {
    filter: brightness(1.1);
  }

  .action-btn:disabled {
    opacity: 0.6;
    cursor: default;
  }
</style>
