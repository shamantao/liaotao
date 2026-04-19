/**
 * bridge.test.js -- Unit tests for lib/bridge.js.
 * Covers: eventsOn/eventsEmit fallback (CustomEvent-based),
 * callService error when Wails runtime is absent.
 */

import { describe, it, expect, beforeEach, vi } from "vitest";
import { eventsOn, eventsEmit } from "../lib/bridge.js";

describe("lib/bridge.js", () => {
  beforeEach(() => {
    // Ensure no Wails runtime in test environment
    delete window.wails;
  });

  describe("eventsOn / eventsEmit (DOM fallback)", () => {
    it("eventsOn receives payloads emitted by eventsEmit", () => {
      const received = [];
      const unsub = eventsOn("test-event", (data) => received.push(data));

      eventsEmit("test-event", { msg: "hello" });
      eventsEmit("test-event", { msg: "world" });

      expect(received).toHaveLength(2);
      expect(received[0]).toEqual({ msg: "hello" });
      expect(received[1]).toEqual({ msg: "world" });

      unsub();
    });

    it("unsubscribe stops receiving events", () => {
      const received = [];
      const unsub = eventsOn("unsub-test", (data) => received.push(data));

      eventsEmit("unsub-test", "first");
      unsub();
      eventsEmit("unsub-test", "second");

      expect(received).toHaveLength(1);
      expect(received[0]).toBe("first");
    });

    it("different event names are isolated", () => {
      const aEvents = [];
      const bEvents = [];
      const unsubA = eventsOn("event-a", (d) => aEvents.push(d));
      const unsubB = eventsOn("event-b", (d) => bEvents.push(d));

      eventsEmit("event-a", 1);
      eventsEmit("event-b", 2);

      expect(aEvents).toEqual([1]);
      expect(bEvents).toEqual([2]);

      unsubA();
      unsubB();
    });
  });

  describe("callService (no Wails runtime)", () => {
    it("throws 'no-wails-runtime' when window.wails is absent", async () => {
      // Use dynamic import with short timeout to avoid 4s wait
      vi.resetModules();
      const mod = await import("../lib/bridge.js");
      // Monkey-patch waitForWailsRuntime timeout via direct call
      await expect(mod.health()).rejects.toThrow("no-wails-runtime");
    }, 5000);
  });

  describe("callService (mocked Wails runtime)", () => {
    it("calls the correct FQN with payload", async () => {
      const mockByName = vi.fn().mockResolvedValue({ ok: true });
      window.wails = { Call: { ByName: mockByName } };

      vi.resetModules();
      const mod = await import("../lib/bridge.js");
      const result = await mod.health();

      expect(mockByName).toHaveBeenCalledWith(
        "liaotao/internal/bindings.Service.Health",
      );
      expect(result).toEqual({ ok: true });

      delete window.wails;
    });
  });
});
