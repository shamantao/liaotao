# Desktop Packaged Smoke-Test Checklist

Date: 2026-05-07
Scope: Validate packaged desktop binaries before release sign-off.

## 1. Pre-Run

1. Verify binary artifact is generated for target OS.
2. Verify checksum/signature artifact exists when applicable.
3. Verify release notes mention the tested build version.

## 2. Install

1. Install package on a clean machine or VM.
2. Confirm installer completes without crash.
3. Confirm app icon/name is correct in launcher.

## 3. First Launch

1. Launch app from OS launcher.
2. Confirm main window opens within acceptable delay.
3. Confirm no blocking error dialog appears at startup.

## 4. Core UX Sanity

1. Open Settings screen.
2. Validate at least one provider connection status check.
3. Send one chat message and confirm timeline update.
4. Confirm execution attempts are visible in chat UI.

## 5. Data and Portability

1. Export a single conversation.
2. Export project conversations.
3. Import the exported package.
4. Confirm imported conversation count matches expectation.

## 6. Shutdown and Relaunch

1. Close app gracefully.
2. Relaunch app.
3. Confirm previous history remains available.

## 7. Result

1. Mark smoke test as PASS only if all checks succeed.
2. Record failures with OS/version/build metadata.