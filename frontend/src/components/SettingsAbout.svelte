<!--
  SettingsAbout.svelte -- About section.
  Responsibilities: display app version, build info, license.
-->
<script>
  import { onMount } from "svelte";
  import * as bridge from "../lib/bridge.js";

  let aboutInfo = $state(null);

  onMount(async () => {
    try {
      aboutInfo = await bridge.getAboutInfo();
    } catch (e) {
      console.error("Failed to load about info:", e);
    }
  });
</script>

<section class="settings-section">
  <h3>About</h3>

  {#if aboutInfo}
    <div class="about-content">
      <div class="about-row">
        <span class="about-label">Version</span>
        <span class="about-value">{aboutInfo.version || "—"}</span>
      </div>
      {#if aboutInfo.build_date}
        <div class="about-row">
          <span class="about-label">Build date</span>
          <span class="about-value">{aboutInfo.build_date}</span>
        </div>
      {/if}
      {#if aboutInfo.go_version}
        <div class="about-row">
          <span class="about-label">Go</span>
          <span class="about-value">{aboutInfo.go_version}</span>
        </div>
      {/if}
      {#if aboutInfo.os}
        <div class="about-row">
          <span class="about-label">OS</span>
          <span class="about-value">{aboutInfo.os}</span>
        </div>
      {/if}
      {#if aboutInfo.arch}
        <div class="about-row">
          <span class="about-label">Arch</span>
          <span class="about-value">{aboutInfo.arch}</span>
        </div>
      {/if}
      {#if aboutInfo.data_dir}
        <div class="about-row">
          <span class="about-label">Data directory</span>
          <span class="about-value mono">{aboutInfo.data_dir}</span>
        </div>
      {/if}
    </div>
  {:else}
    <p class="loading">Loading…</p>
  {/if}

  <div class="license">
    <p>liaotao — local-first AI chat client</p>
    <p>MIT License · <a href="https://github.com/pmusic/liaotao" target="_blank" rel="noreferrer">GitHub</a></p>
  </div>
</section>

<style>
  .settings-section h3 {
    margin: 0 0 1rem;
    font-size: 1.1rem;
    color: var(--text-primary, #E9E3D5);
  }

  .about-content {
    display: flex;
    flex-direction: column;
    gap: 0.5rem;
    margin-bottom: 1.5rem;
  }

  .about-row {
    display: flex;
    gap: 1rem;
    align-items: baseline;
  }

  .about-label {
    color: var(--text-secondary, #beb8ad);
    font-size: 0.84rem;
    min-width: 120px;
    flex-shrink: 0;
  }

  .about-value {
    color: var(--text-primary, #E9E3D5);
    font-size: 0.87rem;
  }

  .about-value.mono {
    font-family: ui-monospace, "SFMono-Regular", Menlo, Consolas, monospace;
    font-size: 0.82rem;
  }

  .loading {
    color: var(--text-secondary, #beb8ad);
    font-size: 0.85rem;
  }

  .license {
    color: var(--text-secondary, #beb8ad);
    font-size: 0.82rem;
    border-top: 1px solid var(--border-default, #2f4f6b);
    padding-top: 1rem;
    margin-top: 1rem;
  }

  .license p {
    margin: 0.25rem 0;
  }

  .license a {
    color: var(--accent-default, #45998A);
    text-decoration: none;
  }

  .license a:hover {
    text-decoration: underline;
  }
</style>
