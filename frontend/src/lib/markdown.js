/**
 * markdown.js -- Pure Markdown/LaTeX rendering utilities for Svelte.
 * Responsibilities: escapeHtml, inline and block Markdown rendering,
 * code block extraction, table parsing, Prism/KaTeX post-processing.
 * No external state — all functions are pure or rely only on window globals.
 * Port of frontend/js/markdown.js for use with Svelte {@html} blocks.
 */

// ── HTML escaping ──────────────────────────────────────────────────────────

export function escapeHtml(text) {
  return String(text)
    .replaceAll("&", "&amp;")
    .replaceAll("<", "&lt;")
    .replaceAll(">", "&gt;")
    .replaceAll('"', "&quot;")
    .replaceAll("'", "&#39;");
}

// ── Inline Markdown ────────────────────────────────────────────────────────

function renderInlineMarkdown(text) {
  let html = escapeHtml(text);
  html = html.replace(/\*\*(.*?)\*\*/g, "<strong>$1</strong>");
  html = html.replace(/\*(.*?)\*/g, "<em>$1</em>");
  html = html.replace(/`([^`]+)`/g, "<code>$1</code>");
  html = html.replace(
    /\[(.*?)\]\((https?:\/\/[^\s)]+)\)/g,
    '<a href="$2" target="_blank" rel="noreferrer">$1</a>',
  );
  html = html.replace(
    /&lt;think&gt;([\s\S]*?)&lt;\/think&gt;/g,
    '<details class="think"><summary>Reasoning</summary><div>$1</div></details>',
  );
  return html;
}

// ── Table parsing ──────────────────────────────────────────────────────────

function parseTableBlock(lines) {
  if (lines.length < 2) return "";
  const header = lines[0].split("|").map((s) => s.trim()).filter(Boolean);
  const separator = lines[1].split("|").map((s) => s.trim()).filter(Boolean);
  if (
    !header.length ||
    separator.length !== header.length ||
    !separator.every((s) => /^-+:?$|^:-+:$|^:?-+$/.test(s))
  ) {
    return "";
  }
  const body = lines
    .slice(2)
    .map((row) => row.split("|").map((s) => s.trim()).filter(Boolean));
  const thead = `<thead><tr>${header.map((h) => `<th>${renderInlineMarkdown(h)}</th>`).join("")}</tr></thead>`;
  const tbody = `<tbody>${body.map((cols) => `<tr>${header.map((_, i) => `<td>${renderInlineMarkdown(cols[i] || "")}</td>`).join("")}</tr>`).join("")}</tbody>`;
  return `<table>${thead}${tbody}</table>`;
}

// ── Block-level Markdown ───────────────────────────────────────────────────

/**
 * Convert raw Markdown text to sanitized HTML.
 * Supports: headings, bold, italic, inline code, links, blockquotes,
 * ordered/unordered lists, tables, fenced code blocks, <think> blocks.
 *
 * Usage in Svelte: {@html renderMarkdown(content)}
 */
export function renderMarkdown(raw) {
  let text = raw || "";
  const codeBlocks = [];

  // Extract fenced code blocks first to avoid processing their content.
  text = text.replace(/```([\w-]*)\n([\s\S]*?)```/g, (_m, lang, code) => {
    const idx = codeBlocks.length;
    codeBlocks.push({ lang: lang || "text", code: escapeHtml(code) });
    return `__CODE_BLOCK_${idx}__`;
  });

  const lines = text.split("\n");
  const html = [];
  let i = 0;

  while (i < lines.length) {
    const line = lines[i];

    // Blank line → <br>
    if (/^\s*$/.test(line)) {
      html.push("<br>");
      i++;
      continue;
    }

    // Table detection
    if (line.includes("|") && i + 1 < lines.length && lines[i + 1].includes("|")) {
      const tableLines = [line, lines[i + 1]];
      let j = i + 2;
      while (j < lines.length && lines[j].includes("|")) {
        tableLines.push(lines[j]);
        j++;
      }
      const tbl = parseTableBlock(tableLines);
      if (tbl) {
        html.push(tbl);
        i = j;
        continue;
      }
    }

    // Headings (h1-h3)
    const heading = line.match(/^(#{1,3})\s+(.*)$/);
    if (heading) {
      html.push(
        `<h${heading[1].length}>${renderInlineMarkdown(heading[2])}</h${heading[1].length}>`,
      );
      i++;
      continue;
    }

    // Blockquotes
    if (/^>\s+/.test(line)) {
      html.push(
        `<blockquote>${renderInlineMarkdown(line.replace(/^>\s+/, ""))}</blockquote>`,
      );
      i++;
      continue;
    }

    // Ordered lists
    if (/^\d+\.\s+/.test(line)) {
      const items = [];
      let j = i;
      while (j < lines.length && /^\d+\.\s+/.test(lines[j])) {
        items.push(lines[j].replace(/^\d+\.\s+/, ""));
        j++;
      }
      html.push(
        `<ol>${items.map((it) => `<li>${renderInlineMarkdown(it)}</li>`).join("")}</ol>`,
      );
      i = j;
      continue;
    }

    // Unordered lists
    if (/^[-*]\s+/.test(line)) {
      const items = [];
      let j = i;
      while (j < lines.length && /^[-*]\s+/.test(lines[j])) {
        items.push(lines[j].replace(/^[-*]\s+/, ""));
        j++;
      }
      html.push(
        `<ul>${items.map((it) => `<li>${renderInlineMarkdown(it)}</li>`).join("")}</ul>`,
      );
      i = j;
      continue;
    }

    // Paragraph (default)
    html.push(`<p>${renderInlineMarkdown(line)}</p>`);
    i++;
  }

  // Re-insert code blocks
  let merged = html.join("\n");
  merged = merged.replace(/__CODE_BLOCK_(\d+)__/g, (_m, idxStr) => {
    const block = codeBlocks[Number(idxStr)];
    return `<pre><code class="language-${block.lang}">${block.code}</code></pre>`;
  });
  return merged;
}

// ── Post-processing (Prism + KaTeX) ────────────────────────────────────────

/**
 * Apply syntax highlighting (Prism) and math rendering (KaTeX) to a
 * DOM container after Svelte has rendered {@html} content.
 *
 * Usage: use:enhance on a container element, or call from onMount/afterUpdate.
 */
export function applyEnhancers(container) {
  if (!container) return;

  // Prism syntax highlighting
  if (window.Prism && typeof window.Prism.highlightAllUnder === "function") {
    window.Prism.highlightAllUnder(container);
  }

  // KaTeX math rendering
  if (window.renderMathInElement) {
    window.renderMathInElement(container, {
      delimiters: [
        { left: "$$", right: "$$", display: true },
        { left: "$", right: "$", display: false },
      ],
      throwOnError: false,
    });
    return;
  }

  // CSS-only math fallback when KaTeX auto-render is unavailable
  container.querySelectorAll("p,li,blockquote").forEach((node) => {
    node.innerHTML = node.innerHTML
      .replace(/\$\$([\s\S]+?)\$\$/g, '<span class="math-block">$1</span>')
      .replace(/\$(.+?)\$/g, '<span class="math-inline">$1</span>');
  });
}

/**
 * Svelte action for automatic post-processing.
 * Usage: <div use:enhance>{@html renderMarkdown(text)}</div>
 */
export function enhance(node) {
  applyEnhancers(node);
  return {
    update() {
      applyEnhancers(node);
    },
  };
}
