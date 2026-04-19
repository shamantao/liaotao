/*
  template.plugin.js -- Reference plugin template for third-party authors.
  Available hooks:
    - beforeSend(payload)     — intercept outgoing message payload
    - afterReceive(payload)   — intercept incoming assistant payload
    - onFileUpload(payload)   — intercept file upload payload
    - renderTool(payload)     — transform tool-call output for display
    - onSaveConv(payload)     — intercept conversation save payload
    - onInit(ctx)             — called once at startup; ctx provides:
        ctx.bridge                          — all bridge functions
        ctx.registerTopbarAction(action)    — show an indicator in topbar
        ctx.updateTopbarAction(id, updates) — update an indicator
        ctx.setInterval(fn, ms)             — managed interval (auto-cleanup)
*/

export default {
  id: "template-plugin",
  name: "Template Plugin",
  description: "Demonstrates all available hooks.",
  enabled: false,
  hooks: {
    beforeSend(payload) {
      return payload;
    },
    afterReceive(payload) {
      return payload;
    },
    onFileUpload(payload) {
      return payload;
    },
    renderTool(payload) {
      return payload;
    },
    onSaveConv(payload) {
      return payload;
    },
    async onInit(ctx) {
      // Example: register a topbar indicator
      // ctx.registerTopbarAction({ id: "my-plugin", label: "My Plugin", color: "#45998A" });
    },
  },
};
