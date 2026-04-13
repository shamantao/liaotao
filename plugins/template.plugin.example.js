/*
  template.plugin.js -- Reference plugin template for third-party authors.
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
  },
};
