/**
 * chat.js -- Chat & conversation store.
 * Responsibilities: active conversation, messages, streaming state,
 * conversations list, projects, tags, attachments.
 * Reactive Svelte writable stores replacing appState chat properties.
 */

import { writable, derived } from "svelte/store";

// ── Conversations ──────────────────────────────────────────────────────────
export const conversations = writable([]);
export const activeConversationId = writable(null);
export const conversationSearchQuery = writable("");

// ── Projects ───────────────────────────────────────────────────────────────
export const projects = writable([]);
export const activeProjectId = writable(0);
export const activeProjectDashboard = writable(null);

// ── Tags ───────────────────────────────────────────────────────────────────
export const tags = writable([]);
export const activeTagId = writable(0);

// ── Attachments (per active conversation) ──────────────────────────────────
export const activeAttachments = writable([]);

// ── Streaming state ────────────────────────────────────────────────────────
export const isStreaming = writable(false);
export const lastUserPrompt = writable("");

// ── Derived: filtered conversations ────────────────────────────────────────
export const filteredConversations = derived(
  [conversations, conversationSearchQuery, activeProjectId, activeTagId],
  ([$conversations, $query, $projectId, $tagId]) => {
    let result = $conversations;

    // Filter by project
    if ($projectId > 0) {
      result = result.filter((c) => c.project_id === $projectId);
    }

    // Filter by tag
    if ($tagId > 0) {
      result = result.filter(
        (c) => c.tags && c.tags.some((t) => t.id === $tagId),
      );
    }

    // Filter by search query
    if ($query.trim()) {
      const q = $query.toLowerCase();
      result = result.filter(
        (c) =>
          (c.title && c.title.toLowerCase().includes(q)) ||
          (c.preview && c.preview.toLowerCase().includes(q)),
      );
    }

    return result;
  },
);

// ── Actions ────────────────────────────────────────────────────────────────

export function setActiveConversation(id) {
  activeConversationId.set(id);
}

export function startStreaming() {
  isStreaming.set(true);
}

export function stopStreaming() {
  isStreaming.set(false);
}
