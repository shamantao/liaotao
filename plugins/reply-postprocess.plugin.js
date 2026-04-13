/*
  reply-postprocess.plugin.js -- Test plugin for afterReceive hook.
  Appends a visible marker to confirm plugin post-processing.
*/

export default {
  id: "test-reply-postprocess",
  name: "Test Reply Post-Processor",
  description: "afterReceive plugin that appends a diagnostic footer.",
  enabled: true,
  hooks: {
    afterReceive(payload) {
      if (!payload || typeof payload.content !== "string") return payload;
      return {
        ...payload,
        content: `${payload.content}\n\n---\nplugin: postprocess applied`,
      };
    },
  },
};
