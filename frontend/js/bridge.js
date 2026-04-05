/*
  bridge.js -- Wails v3 runtime bridge for frontend ↔ Go service calls.
  Responsibilities: wrap window.wails.Call.ByName and window.wails.Events,
  poll for runtime readiness, and provide a DOM-event fallback for offline dev.

  IMPORTANT: Wails v3 runtime.js is an ES Module. Load it as:
    <script type="module" src="/wails/runtime.js">
  The entry script (app.js) must also use type="module".

  ByName convention: only pass payload when the Go method has a non-context argument.
    bridge.callService("MethodName")           // Go: func(ctx) only
    bridge.callService("MethodName", payload)  // Go: func(ctx, T) with payload T
*/

// FQN format: "{go-module-path}.{TypeName}.{MethodName}"
const SERVICE_FQN = "liaotao/internal/bindings.Service.";

// waitForWailsRuntime polls until window.wails.Call.ByName is available.
async function waitForWailsRuntime(timeoutMs = 4000) {
  const start = Date.now();
  while (Date.now() - start < timeoutMs) {
    if (window.wails?.Call?.ByName) return;
    await new Promise((resolve) => setTimeout(resolve, 60));
  }
  console.error("[liaotao] waitForWailsRuntime: timeout. window.wails =", window.wails);
}

export const bridge = {
  eventsOn(name, cb) {
    if (window.wails?.Events?.On) {
      // Wails v3 event callback receives a WailsEvent object — unwrap .data for callers.
      window.wails.Events.On(name, (e) => cb(e && e.data !== undefined ? e.data : e));
      return true;
    }
    // DOM event fallback (offline / test mode without Wails runtime)
    document.addEventListener(`liaotao:${name}`, (e) => cb(e.detail));
    return false;
  },

  eventsEmit(name, payload) {
    if (window.wails?.Events?.Emit) {
      window.wails.Events.Emit(name, payload);
      return;
    }
    document.dispatchEvent(new CustomEvent(`liaotao:${name}`, { detail: payload }));
  },

  async callService(method, payload) {
    await waitForWailsRuntime();
    if (!window.wails?.Call?.ByName) {
      console.error("[liaotao] callService: no-wails-runtime — window.wails =", window.wails);
      throw new Error("no-wails-runtime");
    }
    const fqn = SERVICE_FQN + method;
    console.debug("[liaotao] →", fqn, payload);
    try {
      // Only pass payload as arg when provided — Go-side argument count must match.
      const result = payload !== undefined && payload !== null
        ? await window.wails.Call.ByName(fqn, payload)
        : await window.wails.Call.ByName(fqn);
      console.debug("[liaotao] ←", fqn, result);
      return result;
    } catch (err) {
      console.error("[liaotao] call failed:", fqn, err);
      throw err;
    }
  },
};
