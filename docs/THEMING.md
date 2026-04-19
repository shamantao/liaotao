# Theming Guide

liaotao supports custom CSS themes. The built-in **Default Dark** theme uses the Aurica Circular palette.

## Architecture

```
frontend/src/themes/
├── tokens.css        # Primitive design tokens (palette, spacing, radius, typography)
├── semantic.css      # Semantic tokens mapped from primitives (surfaces, text, accent…)
└── default-dark.css  # Built-in theme: imports tokens + semantic
```

### Token layers

1. **Primitives** (`tokens.css`): raw values — colors (`--color-midnight-500`), spacing (`--space-4`), radius (`--radius-md`), fonts (`--font-sans`).
2. **Semantic** (`semantic.css`): purpose-driven mappings — `--surface-primary`, `--text-primary`, `--accent-default`, etc.
3. **Theme file**: imports both layers. Community themes override either layer.

### Semantic token contract

Components MUST reference only semantic tokens. Never use primitives directly.

| Category | Tokens |
|----------|--------|
| Surfaces | `--surface-primary`, `--surface-secondary`, `--surface-elevated`, `--surface-input`, `--surface-block`, `--surface-block-head`, `--surface-menu`, `--surface-item`, `--surface-item-hover`, `--surface-item-active`, `--surface-badge`, `--surface-active`, `--surface-code`, `--surface-code-border`, `--surface-tab`, `--surface-tab-hover`, `--surface-soft` |
| Text | `--text-primary`, `--text-secondary`, `--text-muted` |
| Accent | `--accent-default`, `--accent-hover` |
| Primary | `--primary-default`, `--primary-hover` |
| Borders | `--border-default`, `--border-subtle` |
| Status | `--danger-default`, `--warning-default`, `--ok-default` |
| Layout | `--sidebar-width`, `--chat-font-size`, `--chat-icon-size` |

## Creating a community theme

1. Create a CSS file (e.g. `solarized-light.css`).
2. Override the `:root` custom properties you want to change:

```css
/* solarized-light.css — Example community theme */
:root {
  --surface-primary: #fdf6e3;
  --surface-secondary: #eee8d5;
  --text-primary: #657b83;
  --text-secondary: #93a1a1;
  --accent-default: #268bd2;
  --accent-hover: #2aa198;
  --border-default: #d3cbb7;
  --danger-default: #dc322f;
  --ok-default: #859900;
  /* ... override as many tokens as needed */
}
```

3. Register the theme via the plugin API:

```javascript
import { registerTheme } from "../stores/theme.js";

registerTheme("solarized-light", "Solarized Light", "./themes/solarized-light.css");
```

4. The theme appears in **Settings > General > Theme**.

## Theme loader

The theme system is managed by `stores/theme.js`:

- `activeThemeId` — writable store holding the current theme id.
- `themes` — derived store listing all registered themes.
- `applyTheme(id)` — switches theme at runtime without page reload.
- `registerTheme(id, label, path)` — registers a community theme.
- `listThemes()` — returns a snapshot of registered themes.

The active theme is persisted in `localStorage` under the key `liaotao-theme`.

## Rules for theme authors

1. Override only `:root` custom properties — do not add component-specific selectors.
2. Provide all semantic tokens for a complete theme; missing tokens fall back to the built-in defaults.
3. Test with both short and long conversations, code blocks, and the Settings panel.
4. Keep file size under 5 KB.
