# Visual Regression Checklist

Run through this checklist **after every CSS or HTML change** before committing.

## Sidebar — Expanded

- [ ] "+" new-chat button is visible in the sidebar header
- [ ] Groups header: only "⋯" menu button visible, no redundant chevron
- [ ] Conversations header: only "⋯" menu button visible, no redundant chevron
- [ ] Clicking the toggle (section title) collapses/expands the section content
- [ ] "⋯" menu opens on click and shows items (Rename, Delete…)
- [ ] No visible borders on sidebar blocks, topbar, or tab bar
- [ ] Conversation items show the "⋯" icon on hover only
- [ ] Conversation context menu (.conv-menu) appears below the "⋯" button, not clipped

## Sidebar — Collapsed

- [ ] Sidebar shrinks to icon-only width
- [ ] Clicking a section icon opens the hierarchy overlay
- [ ] "+" button is hidden

## Chat Area

- [ ] Bubbles have no visible border (borderless look)
- [ ] User bubble has a distinct darker background (--surface-active)
- [ ] Code blocks use --surface-code background, no border
- [ ] Math blocks (KaTeX) render without a border
- [ ] Think/reasoning blocks have no dashed border
- [ ] Composer textarea has no border, uses --surface-input background
- [ ] Send button gradient is visible and text is readable

## Settings

- [ ] Input fields have no visible border, use --surface-input background
- [ ] Action buttons have no border
- [ ] Error states use --danger color

## General

- [ ] All surfaces use CSS var() tokens (no hardcoded hex outside :root)
- [ ] Topbar has no bottom border
- [ ] Tab bar buttons have no box-shadow or border
- [ ] Renaming a conversation does NOT change its date in the sidebar

## Verification Command

```bash
# Must return exit code 1 (no matches) for all CSS files:
grep -rn '#[0-9a-fA-F]\{3,8\}' frontend/css/base.css frontend/css/chat.css frontend/css/settings.css
```
