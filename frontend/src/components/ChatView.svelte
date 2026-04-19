<!--
  ChatView.svelte -- Main chat message area.
  Responsibilities: display message list, handle Wails streaming events,
  auto-scroll on new content, load messages on conversation switch.
-->
<script>
  import { onMount } from "svelte";
  import { activeConversationId, isStreaming, startStreaming, stopStreaming } from "../stores/chat.js";
  import * as bridge from "../lib/bridge.js";
  import MessageBubble from "./MessageBubble.svelte";

  let messages = $state([]);
  let streamingContent = $state("");
  let messagesEl;

  // Load messages when active conversation changes
  $effect(() => {
    const convId = $activeConversationId;
    if (convId) {
      loadMessages(convId);
    } else {
      messages = [];
    }
  });

  async function loadMessages(convId) {
    try {
      const result = await bridge.listMessages({ ConversationID: convId, Limit: 500 });
      messages = result || [];
      scrollToBottom();
    } catch (e) {
      console.error("Failed to load messages:", e);
    }
  }

  function scrollToBottom() {
    requestAnimationFrame(() => {
      if (messagesEl) {
        messagesEl.scrollTop = messagesEl.scrollHeight;
      }
    });
  }

  async function handleDelete(msg) {
    const convId = $activeConversationId;
    if (!convId) return;
    try {
      await bridge.deleteMessage({ ConversationID: convId, MessageID: msg.id });
      messages = messages.filter((m) => m.id !== msg.id);
    } catch (e) {
      console.error("Delete failed:", e);
    }
  }

  function handleEdit(msg) {
    // Dispatch edit event — composer will pick it up
    document.dispatchEvent(
      new CustomEvent("liaotao:edit-message", { detail: msg }),
    );
  }

  // Wire up Wails streaming events
  onMount(() => {
    const unsubs = [];

    unsubs.push(
      bridge.eventsOn("chat:chunk", (data) => {
        if (!data) return;
        streamingContent += data.content || "";
        scrollToBottom();
      }),
    );

    unsubs.push(
      bridge.eventsOn("chat:done", (data) => {
        if (streamingContent) {
          // Add streamed message to list
          messages = [
            ...messages,
            { id: Date.now(), role: "assistant", content: streamingContent },
          ];
          streamingContent = "";
        }
        stopStreaming();
        // Reload messages to get server-persisted version with stats
        const convId = $activeConversationId;
        if (convId) {
          setTimeout(() => loadMessages(convId), 200);
        }
      }),
    );

    unsubs.push(
      bridge.eventsOn("chat:meta", (data) => {
        if (!data) return;
        // Attach metadata to last assistant message
        messages = messages.map((m, i) => {
          if (i === messages.length - 1 && m.role === "assistant") {
            return { ...m, meta: data };
          }
          return m;
        });
      }),
    );

    unsubs.push(
      bridge.eventsOn("chat:tool_call", (data) => {
        if (!data) return;
        // Append tool call indicator to streaming
        streamingContent += `\n⚙ calling: ${data.tool_name}…`;
        scrollToBottom();
      }),
    );

    unsubs.push(
      bridge.eventsOn("chat:error", (data) => {
        if (!data) return;
        messages = [
          ...messages,
          {
            id: Date.now(),
            role: "assistant",
            content: `**Error:** ${data.message || "Unknown error"}`,
          },
        ];
        streamingContent = "";
        stopStreaming();
      }),
    );

    return () => {
      unsubs.forEach((fn) => fn && fn());
    };
  });
</script>

<div class="messages" bind:this={messagesEl}>
  {#if messages.length === 0 && !$isStreaming}
    <div class="empty-chat">
      <p>Start a conversation</p>
    </div>
  {/if}

  {#each messages as msg (msg.id)}
    <MessageBubble
      message={msg}
      ondelete={handleDelete}
      onedit={handleEdit}
    />
  {/each}

  <!-- Streaming bubble -->
  {#if $isStreaming && streamingContent}
    <MessageBubble
      message={{ id: "streaming", role: "assistant", content: streamingContent }}
      streaming={true}
    />
  {:else if $isStreaming}
    <MessageBubble
      message={{ id: "streaming", role: "assistant", content: "" }}
      streaming={true}
    />
  {/if}
</div>

<style>
  .messages {
    font-size: var(--chat-font-size, 1rem);
    padding: 1rem;
    overflow-y: auto;
    display: flex;
    flex-direction: column;
    gap: 0.75rem;
    min-height: 0;
    flex: 1;
  }

  .empty-chat {
    display: flex;
    align-items: center;
    justify-content: center;
    flex: 1;
    color: var(--text-secondary, #beb8ad);
    font-size: 0.92rem;
    opacity: 0.6;
  }
</style>
