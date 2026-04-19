/**
 * markdown.test.js -- Unit tests for lib/markdown.js.
 * Covers: escapeHtml, renderMarkdown (headings, bold, italic, code,
 * links, lists, blockquotes, tables, code blocks, think blocks).
 */

import { describe, it, expect } from "vitest";
import { escapeHtml, renderMarkdown } from "../lib/markdown.js";

describe("lib/markdown.js", () => {
  describe("escapeHtml", () => {
    it("escapes all dangerous characters", () => {
      expect(escapeHtml('<script>alert("xss")</script>')).toBe(
        "&lt;script&gt;alert(&quot;xss&quot;)&lt;/script&gt;",
      );
    });

    it("escapes ampersands", () => {
      expect(escapeHtml("A & B")).toBe("A &amp; B");
    });

    it("escapes single quotes", () => {
      expect(escapeHtml("it's")).toBe("it&#39;s");
    });

    it("handles empty string", () => {
      expect(escapeHtml("")).toBe("");
    });
  });

  describe("renderMarkdown", () => {
    it("renders bold text", () => {
      const html = renderMarkdown("Hello **world**!");
      expect(html).toContain("<strong>world</strong>");
    });

    it("renders italic text", () => {
      const html = renderMarkdown("Hello *world*!");
      expect(html).toContain("<em>world</em>");
    });

    it("renders inline code", () => {
      const html = renderMarkdown("Use `console.log()` here");
      expect(html).toContain("<code>console.log()</code>");
    });

    it("renders links with target=_blank", () => {
      const html = renderMarkdown("Visit [Google](https://google.com)");
      expect(html).toContain('href="https://google.com"');
      expect(html).toContain('target="_blank"');
      expect(html).toContain("rel=\"noreferrer\"");
    });

    it("renders headings h1-h3", () => {
      expect(renderMarkdown("# Title")).toContain("<h1>Title</h1>");
      expect(renderMarkdown("## Sub")).toContain("<h2>Sub</h2>");
      expect(renderMarkdown("### Minor")).toContain("<h3>Minor</h3>");
    });

    it("renders blockquotes", () => {
      const html = renderMarkdown("> Quote here");
      expect(html).toContain("<blockquote>Quote here</blockquote>");
    });

    it("renders unordered lists", () => {
      const html = renderMarkdown("- item 1\n- item 2");
      expect(html).toContain("<ul>");
      expect(html).toContain("<li>item 1</li>");
      expect(html).toContain("<li>item 2</li>");
    });

    it("renders ordered lists", () => {
      const html = renderMarkdown("1. first\n2. second");
      expect(html).toContain("<ol>");
      expect(html).toContain("<li>first</li>");
      expect(html).toContain("<li>second</li>");
    });

    it("renders fenced code blocks", () => {
      const md = "```js\nconsole.log('hi');\n```";
      const html = renderMarkdown(md);
      expect(html).toContain("<pre>");
      expect(html).toContain("<code");
      expect(html).toContain("console.log");
    });

    it("renders tables", () => {
      const md = "| Name | Age |\n|------|-----|\n| Alice | 30 |";
      const html = renderMarkdown(md);
      expect(html).toContain("<table>");
      expect(html).toContain("<th>Name</th>");
      expect(html).toContain("<td>Alice</td>");
    });

    it("renders <think> blocks as collapsible details", () => {
      const md = "<think>some reasoning</think>";
      const html = renderMarkdown(md);
      expect(html).toContain("<details");
      expect(html).toContain("Reasoning");
      expect(html).toContain("some reasoning");
    });

    it("escapes HTML in regular text (XSS prevention)", () => {
      const html = renderMarkdown("Hello <script>alert(1)</script>");
      expect(html).not.toContain("<script>");
      expect(html).toContain("&lt;script&gt;");
    });

    it("handles null/undefined input", () => {
      expect(renderMarkdown(null)).toBeDefined();
      expect(renderMarkdown(undefined)).toBeDefined();
      expect(renderMarkdown("")).toBeDefined();
    });
  });
});
