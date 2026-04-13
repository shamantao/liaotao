/*
  updates.js -- UPD-02/UPD-03/UPD-05 Auto-update UI handler.
  Responsibilities: display update banner, handle dismissal, download & install, check button in About.
  Exports: checkForUpdates(), dismissUpdateBanner(), initializeUpdatesUI(), downloadAndInstallUpdate().
*/

import { appState } from "./state.js";
import { bridge } from "./bridge.js";

const UPDATE_DISMISSED_KEY = "liaotao.update-dismissed"; // dismissal timestamp key
const UPDATE_CHECK_INTERVAL = 24 * 60 * 60 * 1000; // 24 hours in ms

// ── Module state ───────────────────────────────────────────────────────────
let latestUpdateInfo = null; // cached UpdateCheckResult
let isDownloading = false;   // prevents multiple concurrent downloads

// ── Public API ─────────────────────────────────────────────────────────────

/**
 * Initialize update UI handlers (banner close, action button, settings button).
 * Call once on app load.
 */
export function initializeUpdatesUI() {
  const banner = document.getElementById("update-banner");
  const actionBtn = document.getElementById("update-banner-action");
  const closeBtn = document.getElementById("update-banner-close");

  if (!banner || !actionBtn || !closeBtn) {
    console.warn("Update banner elements not found (check index.html)");
    return;
  }

  // Banner close button
  closeBtn.addEventListener("click", () => dismissUpdateBanner());

  // Banner action button — download and install (UPD-03)
  actionBtn.addEventListener("click", () => {
    downloadAndInstallUpdate();
  });
}

/**
 * Check for updates asynchronously.
 * Calls go binding CheckForUpdate(), compares versions, displays banner if needed.
 * Respects appState.settings.autoCheckUpdates flag.
 */
export async function checkForUpdates() {
  // Honor autoCheckUpdates setting (can be disabled in Settings > General)
  if (!appState.settings.autoCheckUpdates) {
    console.debug("Auto-check updates disabled in settings");
    return;
  }

  try {
    // Call Go binding via Wails bridge
    const result = await bridge.callService("CheckForUpdate");
    if (!result) {
      console.debug("CheckForUpdate returned empty result");
      return;
    }

    latestUpdateInfo = result;

    // If error occurred during check, log silently (don't spam user)
    if (result.Error) {
      console.debug(`Update check error: ${result.Error}`);
      return;
    }

    // If new version available, show banner (unless dismissed)
    if (result.HasUpdate) {
      showUpdateBannerIfNotDismissed();
    }
  } catch (err) {
    console.debug(`Update check exception: ${err.message}`);
  }
}

/**
 * Dismiss the update banner and record dismissal timestamp.
 * User can dismiss for a 24-hour period or until a new version is released.
 */
export function dismissUpdateBanner() {
  const banner = document.getElementById("update-banner");
  if (banner) {
    banner.hidden = true;
    // Store dismissal timestamp: check again after 24h or on new version
    localStorage.setItem(UPDATE_DISMISSED_KEY, new Date().toISOString());
  }
}

/**
 * Called from Settings > About to trigger immediate update check.
 * Ignores dismissal state to allow user-initiated check.
 */
export async function checkForUpdatesManual() {
  await checkForUpdates();
  showUpdateBannerForceShow();
}

// ── Private helpers ────────────────────────────────────────────────────────

/**
 * Show banner if dismissal has expired or version changed.
 */
function showUpdateBannerIfNotDismissed() {
  const dismissedISO = localStorage.getItem(UPDATE_DISMISSED_KEY);

  if (!dismissedISO) {
    // Never dismissed
    showUpdateBanner();
    return;
  }

  const dismissedTime = new Date(dismissedISO).getTime();
  const now = new Date().getTime();
  const hoursSinceDismissal = (now - dismissedTime) / (1000 * 60 * 60);

  if (hoursSinceDismissal >= 24) {
    // 24 hours passed, show again
    showUpdateBanner();
  }
  // Otherwise stay hidden (user dismissed recently)
}

/**
 * Force show banner (for manual check button).
 */
function showUpdateBannerForceShow() {
  if (latestUpdateInfo?.HasUpdate) {
    showUpdateBanner();
  }
}

/**
 * Display the update banner element.
 */
function showUpdateBanner() {
  const banner = document.getElementById("update-banner");
  if (banner) {
    banner.hidden = false;
  }
}

/**
 * Download and install the latest update (UPD-03).
 * Calls DownloadAndInstallUpdate Go binding, displays progress/status.
 */
export async function downloadAndInstallUpdate() {
  if (isDownloading) {
    console.warn("Download already in progress");
    return;
  }

  if (!latestUpdateInfo?.HasUpdate) {
    console.warn("No update available");
    return;
  }

  isDownloading = true;
  const banner = document.getElementById("update-banner");
  const actionBtn = document.getElementById("update-banner-action");
  const originalText = actionBtn ? actionBtn.textContent : "";

  try {
    // Update UI to show progress
    if (actionBtn) {
      actionBtn.disabled = true;
      actionBtn.textContent = "Downloading...";
    }

    // Call Go binding to download and install
    const result = await bridge.callService("DownloadAndInstallUpdate");

    if (!result) {
      showStatusMessage("Download failed: no response from server");
      return;
    }

    if (result.Error) {
      showStatusMessage(`Download failed: ${result.Error}`);
      return;
    }

    if (result.Success) {
      showStatusMessage(`✓ ${result.Message}`);
      // Clear dismissal so banner reappears after restart with new version notice
      localStorage.removeItem(UPDATE_DISMISSED_KEY);
      // Suggest user restart
      if (banner) {
        setTimeout(() => {
          alert("liaotao has been updated successfully.\n\nPlease restart the application to use the new version.");
        }, 500);
      }
    } else {
      showStatusMessage(`Download failed: ${result.Message || "Unknown error"}`);
    }
  } catch (err) {
    console.error("Download error:", err);
    showStatusMessage(`Download failed: ${err.message}`);
  } finally {
    isDownloading = false;
    if (actionBtn) {
      actionBtn.disabled = false;
      actionBtn.textContent = originalText;
    }
  }
}

/**
 * Show a temporary status message in the banner.
 */
function showStatusMessage(message) {
  const banner = document.getElementById("update-banner");
  if (!banner) return;

  const content = banner.querySelector(".update-banner-content");
  if (!content) return;

  // Get original text to restore later
  const text = content.querySelector(".update-banner-text");
  if (!text) return;

  const originalText = text.textContent;

  // Show status
  text.textContent = message;

  // Auto-hide and restore after 5 seconds
  setTimeout(() => {
    text.textContent = originalText;
  }, 5000);
}

/**
 * Schedule periodic update checks (optional, for future use).
 */
export function schedulePeriodicUpdateCheck() {
  // Check on startup
  checkForUpdates();

  // Then check periodically (optional)
  setInterval(() => {
    checkForUpdates();
  }, UPDATE_CHECK_INTERVAL);
}
