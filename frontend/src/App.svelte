<!--
  App.svelte — Root application shell.
  Provides the top-level layout: topbar, sidebar, main content (chat or settings).
-->
<script>
  import { onMount } from "svelte";
  import { activeTab } from "./stores/app.js";
  import { loadSettings } from "./stores/settings.js";
  import { loadProviderState } from "./stores/providers.js";
  import { initPlugins } from "./stores/plugins.js";
  import Topbar from "./components/Topbar.svelte";
  import Sidebar from "./components/Sidebar.svelte";
  import ChatView from "./components/ChatView.svelte";
  import Composer from "./components/Composer.svelte";
  import Settings from "./components/Settings.svelte";

  onMount(() => {
    loadSettings();
    loadProviderState();
    initPlugins();
  });
</script>

<div class="app-shell">
  <Topbar />

  <div class="main-area">
    {#if $activeTab === "chat"}
      <Sidebar />
      <section class="chat-panel">
        <ChatView />
        <Composer />
      </section>
    {:else}
      <Settings />
    {/if}
  </div>
</div>

<style>
  .app-shell {
    display: flex;
    flex-direction: column;
    height: 100vh;
    background: var(--bg, #14202e);
    color: var(--text, #E9E3D5);
    font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif;
  }

  .main-area {
    display: flex;
    flex: 1;
    overflow: hidden;
  }

  .chat-panel {
    display: flex;
    flex-direction: column;
    flex: 1;
    min-width: 0;
  }
</style>
