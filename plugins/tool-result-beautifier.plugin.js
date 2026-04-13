/*
  tool-result-beautifier.plugin.js -- Test plugin for renderTool hook.
  Pretty-prints JSON tool outputs for better readability.
*/

function tryPrettyJSON(value) {
  if (typeof value !== "string") return null;
  try {
    const parsed = JSON.parse(value);
    return JSON.stringify(parsed, null, 2);
  } catch {
    return null;
  }
}

export default {
  id: "test-tool-result-beautifier",
  name: "Test Tool Result Beautifier",
  description: "renderTool plugin that formats JSON tool output.",
  enabled: true,
  hooks: {
    renderTool(payload) {
      if (!payload || typeof payload.content !== "string") return payload;

      const pretty = tryPrettyJSON(payload.content);
      if (pretty) {
        return {
          ...payload,
          content: `TOOL VIEW (${payload.name || "unknown"})\n\n${pretty}`,
        };
      }

      return {
        ...payload,
        content: `TOOL VIEW (${payload.name || "unknown"})\n\n${payload.content}`,
      };
    },
  },
};
