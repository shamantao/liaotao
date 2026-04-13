/*
  updates.js -- UPD-02/UPD-05 Auto-update UI handler.
  Responsibilities: display update banner, handle dismissal, wire "Check for updates" button.
  Exports: checkForUpdates(), dismissUpdateBanner(), initializeUpdatesUI().
*/

import { appState } from "./state.js";
import { bridge } from "./bridge.js";

const UPDATE_DISMISSED_KEY = "liaotao.update-dismissed"; // dismissal timestamp key
const UPDATE_CHECK_INTERVAL = 24 * 60 * 60 * 1000; // 24 hours in ms

// ── Module state ───────────────────────────────────────────────────────────
let latestUpdateInfo = null; // cached UpdateCheckResult

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

  // Banner action button — open release URL
  actionBtn.addEventListener("click", () => {
    if (latestUpdateInfo?.LatestRelease?.TagName) {
      const releaseURL = `https://github.com/shamantao/liaotao/releases/tag/${latestUpdateInfo.LatestRelease.TagName}`;
      window.open(releaseURL, "_blank");
    }
  });
}

/**
 * Check for updates asynchronously.
 * Calls go binding CheckForUpdate(), compares versions, displays banner if needed.
 */
export async function checkForUpdates() {
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
