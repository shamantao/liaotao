/*
  guardrail-prompt.plugin.js -- Test plugin for beforeSend hook.
  Adds a lightweight safety prefix and can block explicit banned terms.
*/

const BANNED_TERMS = ["DROP TABLE", "rm -rf /"];

export default {
  id: "test-guardrail-prompt",
  name: "Test Guardrail Prompt",
  description: "beforeSend plugin that prefixes prompts and blocks unsafe patterns.",
  enabled: true,
  hooks: {
    beforeSend(payload) {
      if (!payload || typeof payload.prompt !== "string") return payload;

      const prompt = payload.prompt.trim();
      if (!prompt) return payload;

      const upper = prompt.toUpperCase();
      for (const term of BANNED_TERMS) {
        if (upper.includes(term)) {
          return { ...payload, cancel: true };
        }
      }

      return {
        ...payload,
        prompt: `[TEST-GUARDRAIL]\n${prompt}`,
      };
    },
  },
};
