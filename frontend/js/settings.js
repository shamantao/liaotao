/*
  settings.js -- Settings-tab helpers for general settings, import/export and About section.
  Responsibilities: load/save General settings from backend bindings, export/import TOML,
  and render About metadata.
*/

import { appState, els, persistSettingsToStorage, applySettingsToUI } from "./state.js";
import { bridge } from "./bridge.js";

function escapeHTML(value) {
  return String(value || "")
    .replace(/&/g, "&amp;")
    .replace(/</g, "&lt;")
    .replace(/>/g, "&gt;")
    .replace(/\"/g, "&quot;")
    .replace(/'/g, "&#39;");
}

function downloadTextFile(filename, content) {
  const blob = new Blob([content], { type: "text/plain;charset=utf-8" });
  const url = URL.createObjectURL(blob);
  const link = document.createElement("a");
  link.href = url;
  link.download = filename;
  document.body.appendChild(link);
  link.click();
  link.remove();
  URL.revokeObjectURL(url);
}

async function copyTextToClipboard(text) {
  if (!navigator.clipboard?.writeText) {
    throw new Error("clipboard unavailable");
  }
  await navigator.clipboard.writeText(text);
}

export async function loadGeneralSettings() {
  const settings = await bridge.callService("GetGeneralSettings");
  if (!settings || typeof settings !== "object") return;
  appState.settings.language = settings.language || appState.settings.language || "en";
  appState.settings.theme = settings.theme || "dark";
  appState.settings.defaultSystemPrompt = settings.default_system_prompt || "";
  appState.settings.expertMode = Boolean(settings.expert_mode);
  appState.settings.responseStyle = settings.response_style || appState.settings.responseStyle || "balanced";
  applySettingsToUI();
  persistSettingsToStorage();
}

export async function saveGeneralSettings() {
  const payload = {
    language: appState.settings.language,
    theme: appState.settings.theme,
    default_system_prompt: appState.settings.defaultSystemPrompt || "",
    expert_mode: Boolean(appState.settings.expertMode),
    response_style: appState.settings.responseStyle || "balanced",
  };
  const updated = await bridge.callService("UpdateGeneralSettings", payload);
  if (updated && typeof updated === "object") {
    appState.settings.language = updated.language || appState.settings.language;
    appState.settings.theme = updated.theme || appState.settings.theme;
    appState.settings.defaultSystemPrompt = updated.default_system_prompt || "";
    appState.settings.expertMode = Boolean(updated.expert_mode);
    appState.settings.responseStyle = updated.response_style || appState.settings.responseStyle || "balanced";
    persistSettingsToStorage();
    applySettingsToUI();
  }
}

export async function exportSettingsTOML() {
  try {
    const fileResult = await bridge.callService("ExportConfigurationToFile");
    if (fileResult && fileResult.ok && fileResult.path) {
      els.status.textContent = `configuration exported: ${fileResult.path}`;
      return;
    }

    // Fallback for environments without backend file export.
    const toml = await bridge.callService("ExportConfiguration");
    if (!toml || typeof toml !== "string") {
      els.status.textContent = "export failed";
      return;
    }
    downloadTextFile("liaotao-config.toml", toml);
    try {
      await copyTextToClipboard(toml);
      els.status.textContent = "configuration exported and copied";
    } catch {
      els.status.textContent = "configuration exported";
    }
  } catch (err) {
    els.status.textContent = `export failed: ${String(err && err.message ? err.message : err)}`;
  }
}

export async function importSettingsTOML(file) {
  if (!file) return;
  const content = await file.text();
  const result = await bridge.callService("ImportConfiguration", { toml: content });
  if (!result || result.ok !== true) {
    els.status.textContent = "import failed";
    return;
  }
  await loadGeneralSettings();
  els.status.textContent = "configuration imported";
}

export async function loadAboutInfo() {
  const info = await bridge.callService("GetAboutInfo");
  if (!info || !els.aboutContent) return;

  const links = info.links && typeof info.links === "object"
    ? Object.entries(info.links)
        .map(([label, href]) => `<li><a href="${escapeHTML(href)}" target="_blank" rel="noreferrer">${escapeHTML(label)}</a></li>`)
        .join("")
    : "";
  const credits = Array.isArray(info.credits)
    ? info.credits.map((item) => `<li>${escapeHTML(item)}</li>`).join("")
    : "";

  els.aboutContent.innerHTML = `
    <p><strong>${escapeHTML(info.name || "liaotao")}</strong> v${escapeHTML(info.version || "dev")}</p>
    <p>${escapeHTML(info.description || "")}</p>
    <h4>Links</h4>
    <ul>${links}</ul>
    <h4>Credits</h4>
    <ul>${credits}</ul>
  `;
}
