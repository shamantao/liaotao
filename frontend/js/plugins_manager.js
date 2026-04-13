/*
  plugins_manager.js -- Settings UI for plugin management (PLUG-03).
  Responsibilities: render installed plugins, toggle enable state, load external
  plugin files, and provide built-in plugin controls.
*/

import { els } from "./state.js";
import { t } from "./i18n.js";
import {
  emitHook,
  listPlugins,
  loadPluginsFromDirectory,
  setPluginEnabled,
} from "./plugins.js";
import {
  deletePromptTemplate,
  exportCurrentConversationMarkdown,
  exportCurrentConversationPDF,
  insertPromptTemplate,
  listPromptTemplates,
  upsertPromptTemplate,
} from "./plugins_builtin.js";

function formatHookLabel(hookName) {
  const key = `plugins.hook_${hookName}`;
  const translated = t(key);
  return translated === key ? hookName : translated;
}

function renderPromptLibraryUI() {
  if (!els.pluginsPromptList) return;
  const templates = listPromptTemplates();
  const names = Object.keys(templates).sort();
  els.pluginsPromptList.innerHTML = names
    .map((name) => `<option value="${name}">${name}</option>`)
    .join("");
}

function renderPluginsList() {
  if (!els.pluginsList) return;
  const plugins = listPlugins();
  if (!plugins.length) {
    els.pluginsList.innerHTML = `<p class="provider-form-placeholder">${t("plugins.none")}</p>`;
    return;
  }

  els.pluginsList.innerHTML = plugins.map((plugin) => `
    <div class="plugin-item">
      <div class="plugin-main">
        <div class="plugin-name">${plugin.name}</div>
        <div class="plugin-meta">${plugin.source} · ${plugin.hooks.map(formatHookLabel).join(" · ")}</div>
        <div class="plugin-description">${plugin.description || t("plugins.no_description")}</div>
      </div>
      <label class="checkbox-row" title="${plugin.description || ""}">
        <input type="checkbox" data-plugin-id="${plugin.id}" ${plugin.enabled ? "checked" : ""}>
        <span>${plugin.enabled ? t("plugins.enabled") : t("plugins.disabled")}</span>
      </label>
    </div>
  `).join("");

  els.pluginsList.querySelectorAll("input[data-plugin-id]").forEach((input) => {
    input.addEventListener("change", () => {
      setPluginEnabled(input.dataset.pluginId, input.checked);
      renderPluginsList();
    });
  });
}

function bindPromptLibraryControls() {
  if (!els.pluginsPromptSaveBtn || !els.pluginsPromptDeleteBtn || !els.pluginsPromptInsertBtn) return;

  els.pluginsPromptSaveBtn.addEventListener("click", () => {
    const name = els.pluginsPromptName ? els.pluginsPromptName.value : "";
    const content = els.pluginsPromptContent ? els.pluginsPromptContent.value : "";
    if (!upsertPromptTemplate(name, content)) return;
    renderPromptLibraryUI();
    renderPluginsList();
  });

  els.pluginsPromptDeleteBtn.addEventListener("click", () => {
    const name = els.pluginsPromptList ? els.pluginsPromptList.value : "";
    if (!deletePromptTemplate(name)) return;
    renderPromptLibraryUI();
  });

  els.pluginsPromptInsertBtn.addEventListener("click", () => {
    const name = els.pluginsPromptList ? els.pluginsPromptList.value : "";
    insertPromptTemplate(name);
  });
}

function bindExportControls() {
  if (els.pluginsExportMdBtn) {
    els.pluginsExportMdBtn.addEventListener("click", () => {
      exportCurrentConversationMarkdown();
    });
  }
  if (els.pluginsExportPdfBtn) {
    els.pluginsExportPdfBtn.addEventListener("click", () => {
      exportCurrentConversationPDF();
    });
  }
}

export function bindPluginManagerEvents() {
  if (els.pluginsReloadBtn) {
    els.pluginsReloadBtn.addEventListener("click", async () => {
      await loadPluginsFromDirectory();
      renderPluginsList();
    });
  }

  if (els.prompt) {
    els.prompt.addEventListener("drop", async (event) => {
      event.preventDefault();
      const files = event.dataTransfer && event.dataTransfer.files ? [...event.dataTransfer.files] : [];
      if (!files.length) return;
      for (const file of files) {
        await emitHook("onFileUpload", { name: file.name, size: file.size, type: file.type || "" });
      }
    });
    els.prompt.addEventListener("dragover", (event) => event.preventDefault());
  }

  bindPromptLibraryControls();
  bindExportControls();
}

export async function loadPluginManager() {
  renderPluginsList();
  renderPromptLibraryUI();
}
