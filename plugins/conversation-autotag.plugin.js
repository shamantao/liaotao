/*
  conversation-autotag.plugin.js -- Test plugin for onSaveConv hook.
  Stores lightweight save history in localStorage for diagnostics.
*/

const STORAGE_KEY = "liaotao.plugin.test.autotag.v1";

function loadHistory() {
  try {
    const raw = localStorage.getItem(STORAGE_KEY);
    const parsed = raw ? JSON.parse(raw) : [];
    return Array.isArray(parsed) ? parsed : [];
  } catch {
    return [];
  }
}

function saveHistory(items) {
  localStorage.setItem(STORAGE_KEY, JSON.stringify(items.slice(0, 20)));
}

export default {
  id: "test-conversation-autotag",
  name: "Test Conversation AutoTag",
  description: "onSaveConv plugin that tracks recent saved conversations.",
  enabled: true,
  hooks: {
    onSaveConv(payload) {
      const entry = {
        conversationId: payload && payload.conversationId ? payload.conversationId : 0,
        providerId: payload && payload.providerId ? payload.providerId : 0,
        model: payload && payload.model ? payload.model : "",
        savedAt: new Date().toISOString(),
      };
      const history = loadHistory();
      history.unshift(entry);
      saveHistory(history);
      return payload;
    },
  },
};
