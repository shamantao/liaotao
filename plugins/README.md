# Plugin Directory

This directory is scanned at startup by `ListPluginScripts`.

## File Naming

Use `.plugin.js` suffix (recommended), for example:

- `my-plugin.plugin.js`

Plain `.js` files are also loaded.

## Plugin Module Contract

Export a default object with this shape:

```javascript
export default {
  id: "my-plugin-id",
  name: "My Plugin",
  description: "What it does",
  enabled: true,
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
    }
  }
};
```

## Hook Payloads

- `beforeSend`: `{ conversationId, prompt, providerId, model }`
- `afterReceive`: `{ conversationId, content, message }`
- `onFileUpload`: `{ name, size, type }`
- `renderTool`: `{ toolCallId, name, content }`
- `onSaveConv`: `{ conversationId, providerId, model }`

## Notes

- Returning `undefined` keeps the previous payload.
- Returning `{ cancel: true }` from `beforeSend` cancels sending.
- Plugin errors are isolated and logged in console; they do not crash chat flow.
