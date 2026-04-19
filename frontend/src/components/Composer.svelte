<!--
  Composer.svelte -- Chat message input area.
  Responsibilities: auto-resizing textarea, send/stop buttons,
  ModelSelector toolbar, file attachments display.
-->
<script>
  import { isStreaming, startStreaming, stopStreaming, activeConversationId, activeAttachments, conversations } from "../stores/chat.js";
  import * as bridge from "../lib/bridge.js";
  import ModelSelector from "./ModelSelector.svelte";
  import { onMount } from "svelte";

  // Icons
  const iconSend = '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M14.536 21.686a.5.5 0 0 0 .937-.024l6.5-19a.496.496 0 0 0-.635-.635l-19 6.5a.5.5 0 0 0-.024.937l7.93 3.18a2 2 0 0 1 1.112 1.11z"/><path d="m21.854 2.147-10.94 10.939"/></svg>';
  const iconStop = '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect x="6" y="6" width="12" height="12" rx="2"/></svg>';
  const iconX = '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M18 6 6 18"/><path d="m6 6 12 12"/></svg>';

  let textareaEl;
  let prompt = $state("");
  let modelSelector;

  function autoResize() {
    if (!textareaEl) return;
    textareaEl.style.height = "auto";
    textareaEl.style.height = Math.min(textareaEl.scrollHeight, 200) + "px";
  }

  function handleInput(e) {
    prompt = e.target.value;
    autoResize();
  }

  function handleKeydown(e) {
    if (e.key === "Enter" && !e.shiftKey) {
      e.preventDefault();
      sendMessage();
    }
  }

  async function sendMessage() {
    const content = prompt.trim();
    if (!content || $isStreaming) return;

    const convId = $activeConversationId;
    if (!convId) return;

    const sel = modelSelector?.getSelections?.() || {};

    startStreaming();
    prompt = "";
    if (textareaEl) textareaEl.style.height = "auto";

    try {
      await bridge.sendMessage({
        ConversationID: convId,
        Content: content,
        ProviderID: sel.providerId || 0,
        Model: sel.model || "",
        Temperature: sel.temperature ?? 0.7,
        MaxTokens: sel.maxTokens ?? 0,
        SystemPrompt: sel.systemPrompt || "",
        ResponseStyle: sel.responseStyle || "balanced",
      });
    } catch (e) {
      console.error("Send failed:", e);
      stopStreaming();
    }
  }

  async function handleStop() {
    try {
      await bridge.cancelGeneration();
    } catch (e) {
      console.error("Stop failed:", e);
    }
    stopStreaming();
  }

  function removeAttachment(index) {
    activeAttachments.update((list) => list.filter((_, i) => i !== index));
  }

  // Listen for edit-message events from ChatView
  onMount(() => {
    function onEdit(e) {
      const msg = e.detail;
      if (msg && msg.content) {
        prompt = msg.content;
        if (textareaEl) {
          textareaEl.focus();
          autoResize();
        }
      }
    }
    document.addEventListener("liaotao:edit-message", onEdit);
    return () => document.removeEventListener("liaotao:edit-message", onEdit);
  });
</script>

<div class="composer-area">
  <ModelSelector bind:this={modelSelector} />

  {#if $activeAttachments.length > 0}
    <div class="attachments">
      {#each $activeAttachments as att, i}
        <div class="attachment-chip">
          <span class="attachment-name">{att.name || att.path}</span>
          <button class="remove-btn" type="button" onclick={() => removeAttachment(i)} title="Remove">
            {@html iconX}
          </button>
        </div>
      {/each}
    </div>
  {/if}

  <div class="composer-main">
    <textarea
      bind:this={textareaEl}
      class="composer-textarea"
      rows="1"
      placeholder="Type a message…"
      value={prompt}
      oninput={handleInput}
      onkeydown={handleKeydown}
      disabled={$isStreaming}
    ></textarea>

    {#if $isStreaming}
      <button class="stop-btn" type="button" title="Stop" onclick={handleStop}>
        {@html iconStop}
      </button>
    {:else}
      <button
        class="send-btn"
        type="button"
        title="Send"
        onclick={sendMessage}
        disabled={!prompt.trim()}
      >
        {@html iconSend}
      </button>
    {/if}
  </div>
</div>

<style>
  .composer-area {
    padding: 0 1rem 1rem;
    background: var(--surface-primary, #14202e);
    display: flex;
    flex-direction: column;
    gap: 0.5rem;
    flex-shrink: 0;
  }

  .attachments {
    display: flex;
    gap: 0.5rem;
    flex-wrap: wrap;
  }

  .attachment-chip {
    display: inline-flex;
    align-items: center;
    gap: 0.25rem;
    background: var(--surface-badge, #2a3c52);
    border-radius: 8px;
    padding: 0.25rem 0.5rem;
    font-size: 0.78rem;
    color: var(--text-primary, #E9E3D5);
    max-width: 200px;
  }

  .attachment-name {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .remove-btn {
    border: none;
    background: transparent;
    color: var(--text-secondary, #beb8ad);
    cursor: pointer;
    display: inline-flex;
    align-items: center;
    width: 16px;
    height: 16px;
    padding: 0;
    flex-shrink: 0;
  }

  .remove-btn :global(svg) {
    width: 12px;
    height: 12px;
  }

  .composer-main {
    display: grid;
    grid-template-columns: 1fr auto;
    gap: 0.25rem;
    align-items: end;
    background: var(--surface-input, #122033);
    border-radius: 14px;
    padding: 0.5rem;
    border: 1px solid var(--border-default, #2f4f6b);
  }

  .composer-textarea {
    background: transparent;
    color: var(--text-primary, #E9E3D5);
    border: none;
    resize: none;
    font-size: 0.92rem;
    line-height: 1.5;
    padding: 0.25rem;
    min-height: 28px;
    max-height: 200px;
    outline: none;
    scrollbar-width: thin;
  }

  .composer-textarea::placeholder {
    color: var(--text-secondary, #beb8ad);
  }

  .send-btn,
  .stop-btn {
    width: 36px;
    height: 36px;
    border: none;
    border-radius: 9px;
    cursor: pointer;
    display: inline-flex;
    align-items: center;
    justify-content: center;
    padding: 0;
    flex-shrink: 0;
  }

  .send-btn {
    background: var(--accent-default, #45998A);
    color: #fff;
  }

  .send-btn:disabled {
    background: transparent;
    color: var(--text-secondary, #beb8ad);
    opacity: 0.5;
    cursor: default;
  }

  .stop-btn {
    background: var(--danger-default, #e74c3c);
    color: #fff;
    animation: spin 1.5s linear infinite;
  }

  @keyframes spin {
    0% { transform: none; }
    100% { transform: rotate(360deg); }
  }

  .send-btn :global(svg),
  .stop-btn :global(svg) {
    width: 18px;
    height: 18px;
  }
</style>
