/**
 * chat.test.js -- Unit tests for stores/chat.js.
 * Covers: conversation/project state, streaming state,
 * filteredConversations derived store with search/project/tag filtering.
 */

import { describe, it, expect, beforeEach } from "vitest";
import { get } from "svelte/store";
import {
  conversations,
  activeConversationId,
  conversationSearchQuery,
  projects,
  activeProjectId,
  activeTagId,
  isStreaming,
  filteredConversations,
  setActiveConversation,
  startStreaming,
  stopStreaming,
} from "../stores/chat.js";

const MOCK_CONVERSATIONS = [
  { id: 1, title: "Hello World", preview: "first message", project_id: 0, tags: [] },
  { id: 2, title: "Svelte Migration", preview: "refactoring UI", project_id: 1, tags: [{ id: 10 }] },
  { id: 3, title: "Go Backend", preview: "bindings layer", project_id: 1, tags: [{ id: 20 }] },
  { id: 4, title: "Random Chat", preview: "nothing special", project_id: 2, tags: [{ id: 10 }, { id: 20 }] },
];

describe("stores/chat.js", () => {
  beforeEach(() => {
    conversations.set([...MOCK_CONVERSATIONS]);
    activeConversationId.set(null);
    conversationSearchQuery.set("");
    activeProjectId.set(0);
    activeTagId.set(0);
    isStreaming.set(false);
  });

  it("setActiveConversation updates the active id", () => {
    setActiveConversation(2);
    expect(get(activeConversationId)).toBe(2);
  });

  it("startStreaming / stopStreaming toggle state", () => {
    expect(get(isStreaming)).toBe(false);
    startStreaming();
    expect(get(isStreaming)).toBe(true);
    stopStreaming();
    expect(get(isStreaming)).toBe(false);
  });

  it("filteredConversations returns all when no filters", () => {
    expect(get(filteredConversations)).toHaveLength(4);
  });

  it("filteredConversations filters by project", () => {
    activeProjectId.set(1);
    const result = get(filteredConversations);
    expect(result).toHaveLength(2);
    expect(result.map((c) => c.id)).toEqual([2, 3]);
  });

  it("filteredConversations filters by tag", () => {
    activeTagId.set(10);
    const result = get(filteredConversations);
    expect(result).toHaveLength(2);
    expect(result.map((c) => c.id)).toEqual([2, 4]);
  });

  it("filteredConversations filters by search query (title)", () => {
    conversationSearchQuery.set("svelte");
    const result = get(filteredConversations);
    expect(result).toHaveLength(1);
    expect(result[0].id).toBe(2);
  });

  it("filteredConversations filters by search query (preview)", () => {
    conversationSearchQuery.set("bindings");
    const result = get(filteredConversations);
    expect(result).toHaveLength(1);
    expect(result[0].id).toBe(3);
  });

  it("filteredConversations combines project + search filters", () => {
    activeProjectId.set(1);
    conversationSearchQuery.set("go");
    const result = get(filteredConversations);
    expect(result).toHaveLength(1);
    expect(result[0].title).toBe("Go Backend");
  });

  it("filteredConversations combines project + tag filters", () => {
    activeProjectId.set(1);
    activeTagId.set(20);
    const result = get(filteredConversations);
    expect(result).toHaveLength(1);
    expect(result[0].id).toBe(3);
  });

  it("filteredConversations returns empty for no match", () => {
    conversationSearchQuery.set("zzz_no_match_zzz");
    expect(get(filteredConversations)).toHaveLength(0);
  });
});
