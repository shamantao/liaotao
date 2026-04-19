/*
  aitao-connect.plugin.js -- Connector plugin for aitao RAG backend.
  Responsibilities: auto-configure aitao provider and MCP server,
  display connection status indicator in topbar (green/orange/red).
*/

// Default aitao connection settings
const AITAO_PROVIDER_NAME = "aitao";
const AITAO_MCP_NAME = "aitao";
const AITAO_URL = "http://localhost:8201";
const POLL_INTERVAL_MS = 30000;

export default {
  id: "aitao-connect",
  name: "aitao Connector",
  description:
    "Auto-configures the aitao RAG provider and MCP server. " +
    "Shows a topbar indicator: green = fully connected, " +
    "orange = partial, red = unreachable.",
  enabled: true,

  hooks: {
    async onInit(ctx) {
      const { bridge } = ctx;

      // ── Find or create aitao provider ────────────────────────────────
      let providerId = null;
      try {
        const providers = await bridge.listProviders();
        const existing = (providers || []).find(
          (p) => p.name?.toLowerCase() === AITAO_PROVIDER_NAME,
        );
        if (existing) {
          providerId = existing.id;
        } else {
          const created = await bridge.createProvider({
            name: AITAO_PROVIDER_NAME,
            type: "openai-compatible",
            url: AITAO_URL,
            api_key: "",
            description: "aitao RAG backend (auto-configured)",
            use_in_rag: true,
            active: true,
            temperature: 0.7,
            num_ctx: 1024,
          });
          providerId = created?.id ?? null;
        }
      } catch (e) {
        console.warn("[aitao-connect] Provider setup failed:", e);
      }

      // ── Find or create aitao MCP server ──────────────────────────────
      let mcpId = null;
      try {
        const servers = await bridge.listMCPServers();
        const existing = (servers || []).find(
          (s) => s.name?.toLowerCase() === AITAO_MCP_NAME,
        );
        if (existing) {
          mcpId = existing.id;
        } else {
          const result = await bridge.saveMCPServer({
            id: 0,
            name: AITAO_MCP_NAME,
            transport: "http",
            url: AITAO_URL,
            command: "",
            args: "[]",
            active: true,
          });
          mcpId = result?.id ?? null;
        }
      } catch (e) {
        console.warn("[aitao-connect] MCP server setup failed:", e);
      }

      // ── Status polling ───────────────────────────────────────────────
      ctx.registerTopbarAction({
        id: "aitao",
        label: "aitao",
        color: "#beb8ad",
        tooltip: "Checking aitao status…",
      });

      async function checkStatus() {
        let providerOk = false;
        let mcpOk = false;

        // Check provider connectivity
        if (providerId) {
          try {
            const result = await bridge.testConnection({
              provider_id: providerId,
            });
            providerOk = result?.ok === true;
          } catch (_) {
            providerOk = false;
          }
        }

        // Check MCP server connectivity
        if (mcpId) {
          try {
            const result = await bridge.pingMCPServer(mcpId);
            mcpOk = result?.ok === true;
          } catch (_) {
            mcpOk = false;
          }
        }

        // Determine color:
        //   green  — provider AND mcp both respond
        //   orange — provider responds but MCP does not (partial)
        //   red    — provider does not respond (aitao down / not started)
        let color, tooltip;
        if (providerOk && mcpOk) {
          color = "#45998A"; // green (accent)
          tooltip = "aitao: connected";
        } else if (providerOk) {
          color = "#E5A54B"; // orange
          tooltip = "aitao: partial — MCP server unreachable";
        } else {
          color = "#C45B5B"; // red
          tooltip = "aitao: not connected";
        }

        ctx.updateTopbarAction("aitao", { color, tooltip });
      }

      // Initial check then poll
      await checkStatus();
      ctx.setInterval(checkStatus, POLL_INTERVAL_MS);
    },
  },
};
