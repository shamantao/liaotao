<!--
  MessageBubble.svelte -- Single chat message bubble.
  Responsibilities: render user/assistant message with markdown,
  action buttons (copy, edit, delete), tool calls, response metadata,
  thinking indicator, think blocks.
-->
<script>
  import { renderMarkdown, enhance } from "../lib/markdown.js";
  import { isStreaming } from "../stores/chat.js";

  let { message = {}, ondelete = () => {}, onedit = () => {}, streaming = false } = $props();

  // Icons
  const iconCopy = '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect width="14" height="14" x="8" y="8" rx="2" ry="2"/><path d="M4 16c-1.1 0-2-.9-2-2V4c0-1.1.9-2 2-2h10c1.1 0 2 .9 2 2"/></svg>';
  const iconPencil = '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M21.174 6.812a1 1 0 0 0-3.986-3.987L3.842 16.174a2 2 0 0 0-.5.83l-1.321 4.352a.5.5 0 0 0 .623.622l4.353-1.32a2 2 0 0 0 .83-.497z"/><path d="m15 5 4 4"/></svg>';
  const iconTrash = '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M3 6h18"/><path d="M19 6v14c0 1-1 2-2 2H7c-1 0-2-1-2-2V6"/><path d="M8 6V4c0-1 1-2 2-2h4c1 0 2 1 2 2v2"/></svg>';

  let copyLabel = $state("Copy");

  function handleCopy() {
    navigator.clipboard.writeText(message.content || "").then(() => {
      copyLabel = "Copied!";
      setTimeout(() => (copyLabel = "Copy"), 1500);
    });
  }

  $effect(() => {
    // Reset copy label when message changes
    if (message.id) copyLabel = "Copy";
  });

  function formatMeta(msg) {
    if (!msg.meta) return "";
    const parts = [];
    if (msg.meta.provider_name) parts.push(msg.meta.provider_name);
    if (msg.meta.model) parts.push(msg.meta.model);
    if (msg.meta.tokens_used) parts.push(`~${msg.meta.tokens_used} tokens`);
    if (msg.meta.duration_ms) parts.push(`${(msg.meta.duration_ms / 1000).toFixed(1)}s`);
    return parts.join(" · ");
  }

  function formatStats(msg) {
    if (!msg.token_stats) return "";
    const s = msg.token_stats;
    const parts = [];
    if (s.tokens_in) parts.push(`in: ${s.tokens_in}`);
    if (s.tokens_out) parts.push(`out: ${s.tokens_out}`);
    if (s.speed) parts.push(`${s.speed.toFixed(1)} t/s`);
    return parts.join(" · ");
  }
</script>

<div class="bubble {message.role}" class:streaming>
  <!-- Tool call indicators -->
  {#if message.tool_calls && message.tool_calls.length > 0}
    {#each message.tool_calls as tc}
      <div class="tool-call" class:done={tc.result}>
        ⚙ {tc.status === "calling" ? "calling" : "called"}: {tc.name}
      </div>
      {#if tc.result}
        <details class="tool-result">
          <summary>▶ {tc.name} result</summary>
          <pre class="tool-result-content">{tc.result}</pre>
        </details>
      {/if}
    {/each}
  {/if}

  <!-- Message content -->
  <div class="markdown" use:enhance>
    {@html renderMarkdown(message.content || "")}
  </div>

  <!-- Thinking indicator (during streaming) -->
  {#if streaming && message.role === "assistant" && !message.content}
    <div class="thinking-indicator">
      <span class="thinking-dot"></span>
      <span class="thinking-dot"></span>
      <span class="thinking-dot"></span>
      <span class="thinking-text">Thinking…</span>
    </div>
  {/if}

  <!-- Action buttons (not during streaming) -->
  {#if !$isStreaming && message.content}
    <div class="actions">
      <button class="action-btn" title={copyLabel} onclick={handleCopy}>
        {@html iconCopy}
      </button>
      {#if message.role === "user"}
        <button class="action-btn" title="Edit" onclick={() => onedit(message)}>
          {@html iconPencil}
        </button>
      {/if}
      <button class="action-btn" title="Delete" onclick={() => ondelete(message)}>
        {@html iconTrash}
      </button>
    </div>
  {/if}

  <!-- Response metadata footer -->
  {#if message.role === "assistant" && message.meta}
    <div class="msg-meta">{formatMeta(message)}</div>
  {/if}

  <!-- Token stats -->
  {#if message.token_stats && formatStats(message)}
    <div class="msg-stats">{formatStats(message)}</div>
  {/if}
</div>

<style>
  .bubble {
    max-width: min(88ch, 87%);
    border: none;
    border-radius: 14px;
    padding: 0.75rem;
    line-height: 1.55;
    background: var(--surface-secondary, #1f2f43);
    animation: rise 180ms ease-out;
  }

  .bubble.user {
    margin-left: auto;
    background: var(--surface-active, #23435f);
  }

  .bubble.assistant {
    margin-right: auto;
  }

  @keyframes rise {
    from { opacity: 0; transform: translateY(8px); }
    to   { opacity: 1; transform: translateY(0); }
  }

  /* Markdown styles */
  .markdown :global(pre) {
    margin: 0.5rem 0;
    background: var(--surface-code, #0d1522);
    border-radius: 10px;
    padding: 0.5rem;
    overflow-x: auto;
    color: var(--text-primary, #E9E3D5);
  }

  .markdown :global(code) {
    font-family: ui-monospace, "SFMono-Regular", Menlo, Consolas, monospace;
  }

  .markdown :global(blockquote) {
    margin: 0.5rem 0;
    padding-left: 0.75rem;
    border-left: 3px solid var(--accent-default, #45998A);
    color: var(--text-secondary, #beb8ad);
  }

  .markdown :global(table) {
    width: 100%;
    border-collapse: collapse;
    margin: 0.5rem 0;
    font-size: 0.92rem;
  }

  .markdown :global(th),
  .markdown :global(td) {
    border: 1px solid var(--border-default, #2f4f6b);
    padding: 0.25rem 0.5rem;
  }

  .markdown :global(ul),
  .markdown :global(ol) {
    margin: 0.5rem 0;
    padding-left: 1.25rem;
  }

  .markdown :global(details.think) {
    margin-top: 0.5rem;
    border-radius: 8px;
    padding: 0.25rem 0.5rem;
    color: var(--text-secondary, #beb8ad);
    background: rgba(24, 34, 50, 0.5);
  }

  /* Tool calls */
  .tool-call {
    font-size: 0.75rem;
    color: var(--accent-default, #45998A);
    margin: 0.5rem 0;
    opacity: 0.85;
  }

  .tool-call.done {
    color: var(--text-secondary, #beb8ad);
  }

  .tool-result {
    margin: 0.5rem 0;
    border: 1px solid var(--border-default, #2f4f6b);
    border-radius: 8px;
    overflow: hidden;
  }

  .tool-result summary {
    padding: 0.25rem 0.5rem;
    font-size: 0.75rem;
    color: var(--text-secondary, #beb8ad);
    cursor: pointer;
    user-select: none;
  }

  .tool-result-content {
    padding: 0.5rem 0.75rem;
    font-size: 0.78rem;
    background: rgba(0, 0, 0, 0.2);
    white-space: pre-wrap;
    word-break: break-word;
    margin: 0;
  }

  /* Actions */
  .actions {
    margin-top: 0.5rem;
    display: flex;
    gap: 0.5rem;
    flex-wrap: wrap;
  }

  .action-btn {
    font-size: 0.75rem;
    border: none;
    color: var(--text-secondary, #beb8ad);
    background: transparent;
    border-radius: 8px;
    padding: 0.25rem 0.5rem;
    cursor: pointer;
    display: inline-flex;
    align-items: center;
    min-width: var(--chat-icon-size, 26px);
    width: var(--chat-icon-size, 26px);
    height: var(--chat-icon-size, 26px);
    justify-content: center;
  }

  .action-btn:hover {
    color: var(--text-primary, #E9E3D5);
  }

  .action-btn :global(svg) {
    width: calc(var(--chat-icon-size, 26px) * 0.55);
    height: calc(var(--chat-icon-size, 26px) * 0.55);
  }

  /* Meta footer */
  .msg-meta {
    font-size: 0.75rem;
    color: var(--text-secondary, #beb8ad);
    text-align: right;
    margin-top: 0.5rem;
    padding-top: 0.25rem;
    border-top: 1px solid var(--border-default, #2f4f6b);
    opacity: 0.65;
  }

  .msg-stats {
    font-size: 0.68rem;
    color: var(--text-secondary, #beb8ad);
    text-align: right;
    margin-top: 0.5rem;
    opacity: 0.78;
  }

  /* Thinking indicator */
  .thinking-indicator {
    display: inline-flex;
    align-items: center;
    gap: 0.25rem;
    margin-top: 0.5rem;
    color: var(--text-secondary, #beb8ad);
    font-size: 0.82rem;
  }

  .thinking-dot {
    width: 6px;
    height: 6px;
    border-radius: 50%;
    background: var(--accent-default, #45998A);
    opacity: 0.35;
    animation: thinking-pulse 1.2s infinite ease-in-out;
  }

  .thinking-dot:nth-child(2) { animation-delay: 0.15s; }
  .thinking-dot:nth-child(3) { animation-delay: 0.3s; }

  .thinking-text {
    margin-left: 0.25rem;
  }

  @keyframes thinking-pulse {
    0%, 80%, 100% { transform: translateY(0); opacity: 0.3; }
    40% { transform: translateY(-4px); opacity: 1; }
  }

  /* KaTeX */
  .markdown :global(.math-inline),
  .markdown :global(.math-block) {
    font-family: "Times New Roman", serif;
    color: var(--text-primary, #E9E3D5);
    background: rgba(244, 229, 178, 0.08);
    border-radius: 8px;
    padding: 0 0.25rem;
  }

  .markdown :global(.math-block) {
    display: block;
    margin: 0.5rem 0;
    padding: 0.5rem;
  }
</style>
