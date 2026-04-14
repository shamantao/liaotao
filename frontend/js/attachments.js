/*
  attachments.js -- Conversation attachment upload and sidebar rendering.
  Responsibilities: drag-and-drop upload on prompt textarea, backend sync,
  and attachment list rendering for the active conversation.
*/

import { appState, els } from "./state.js";
import { bridge } from "./bridge.js";
import { t } from "./i18n.js";
import { emitHook } from "./plugins.js";

function escapeHTML(value) {
  return String(value)
    .replace(/&/g, "&amp;")
    .replace(/</g, "&lt;")
    .replace(/>/g, "&gt;")
    .replace(/\"/g, "&quot;")
    .replace(/'/g, "&#39;");
}

function humanFileSize(sizeBytes) {
  const size = Number(sizeBytes) || 0;
  if (size < 1024) return `${size} B`;
  if (size < 1024 * 1024) return `${(size / 1024).toFixed(1)} KB`;
  return `${(size / (1024 * 1024)).toFixed(1)} MB`;
}

function activeConversationId() {
  return Number(appState.activeConversationId) || 0;
}

function renderAttachmentList() {
  if (!els.attachmentList) return;
  const items = Array.isArray(appState.activeAttachments) ? appState.activeAttachments : [];
  if (items.length === 0) {
    els.attachmentList.innerHTML = `<p class="attachment-empty">${escapeHTML(t("sidebar.no_attachments"))}</p>`;
    return;
  }

  els.attachmentList.innerHTML = items.map((item) => `
    <div class="attachment-item" title="${escapeHTML(item.fileName)}">
      <span class="attachment-name">${escapeHTML(item.fileName)}</span>
      <span class="attachment-meta">${escapeHTML(humanFileSize(item.sizeBytes))}</span>
      <button class="attachment-share-btn" type="button" data-attachment-id="${item.id}" data-shared="${item.sharedInProject ? "1" : "0"}">
        ${item.sharedInProject ? escapeHTML(t("sidebar.attachment_shared_on")) : escapeHTML(t("sidebar.attachment_shared_off"))}
      </button>
    </div>
  `).join("");

  els.attachmentList.querySelectorAll(".attachment-share-btn").forEach((btn) => {
    btn.addEventListener("click", async (event) => {
      event.preventDefault();
      const attachmentID = Number(btn.dataset.attachmentId) || 0;
      if (attachmentID <= 0) return;
      const shared = btn.dataset.shared !== "1";
      await bridge.callService("SetAttachmentProjectScope", {
        attachment_id: attachmentID,
        shared,
      });
      await loadActiveConversationAttachments();
      window.dispatchEvent(new CustomEvent("liaotao:project-dashboard-refresh"));
    });
  });
}

function mapAttachment(item) {
  return {
    id: Number(item.id) || 0,
    conversationId: Number(item.conversation_id) || 0,
    projectId: Number(item.project_id) || 0,
    fileName: String(item.file_name || ""),
    mimeType: String(item.mime_type || ""),
    sizeBytes: Number(item.size_bytes) || 0,
    sharedInProject: Boolean(item.shared_in_project),
    storagePath: String(item.storage_path || ""),
    createdAt: String(item.created_at || ""),
  };
}

function fileToBase64(file) {
  return new Promise((resolve, reject) => {
    const reader = new FileReader();
    reader.onload = () => {
      const dataUrl = String(reader.result || "");
      const comma = dataUrl.indexOf(",");
      if (comma < 0) {
        reject(new Error("invalid file payload"));
        return;
      }
      resolve(dataUrl.slice(comma + 1));
    };
    reader.onerror = () => reject(reader.error || new Error("file read failed"));
    reader.readAsDataURL(file);
  });
}

export async function loadActiveConversationAttachments() {
  const conversationID = activeConversationId();
  if (conversationID <= 0) {
    appState.activeAttachments = [];
    renderAttachmentList();
    return;
  }

  const result = await bridge.callService("ListAttachments", {
    conversation_id: conversationID,
    limit: 200,
  });
  appState.activeAttachments = Array.isArray(result)
    ? result.map(mapAttachment)
    : [];
  renderAttachmentList();
}

export async function uploadConversationFiles(files) {
  const conversationID = activeConversationId();
  if (conversationID <= 0 || !files || files.length === 0) return;

  for (const file of files) {
    const contentBase64 = await fileToBase64(file);
    const uploaded = await bridge.callService("UploadAttachment", {
      conversation_id: conversationID,
      file_name: file.name,
      mime_type: file.type || "",
      content_base64: contentBase64,
    });

    await emitHook("onFileUpload", {
      id: uploaded && uploaded.id ? uploaded.id : 0,
      name: file.name,
      size: file.size,
      type: file.type || "",
      conversationId: conversationID,
    });
  }

  await loadActiveConversationAttachments();
  window.dispatchEvent(new CustomEvent("liaotao:project-dashboard-refresh"));
  if (els.status) {
    els.status.textContent = t("sidebar.attachments_uploaded", { count: String(files.length) });
  }
}

function setPromptDropActive(active) {
  if (!els.prompt) return;
  els.prompt.classList.toggle("drop-active", active);
}

export function bindAttachmentEvents() {
  if (!els.prompt) return;

  els.prompt.addEventListener("dragenter", (event) => {
    event.preventDefault();
    setPromptDropActive(true);
  });

  els.prompt.addEventListener("dragover", (event) => {
    event.preventDefault();
    setPromptDropActive(true);
  });

  els.prompt.addEventListener("dragleave", () => {
    setPromptDropActive(false);
  });

  els.prompt.addEventListener("drop", async (event) => {
    event.preventDefault();
    setPromptDropActive(false);
    const files = event.dataTransfer && event.dataTransfer.files
      ? [...event.dataTransfer.files]
      : [];
    if (!files.length) return;

    try {
      await uploadConversationFiles(files);
    } catch (err) {
      if (els.status) {
        const message = err && err.message ? err.message : String(err);
        els.status.textContent = `upload failed: ${message}`;
      }
    }
  });
}
