---
name: paca-format
description: Formatting rules for any text you send as a Markdown string to create_task, update_task, or write_doc. Paca stores descriptions and doc content as BlockNote rich-text blocks, not raw Markdown — the string you write is converted by a lossy parser that only understands a small subset of Markdown. Always active; no invocation needed.
---

Every `description` field on `create_task`/`update_task`, and every `content` field on `write_doc`, is a Markdown **string** on the wire — but Paca converts it into BlockNote block JSON before storing it, using a minimal parser that only recognizes a specific subset of Markdown. Anything outside that subset is either dropped, flattened into a wrong block, or shown to the user as literal punctuation instead of formatting. This is not cosmetic: it is the most common cause of garbled task/epic descriptions.

Treat the list below as the entire language available to you. If you want to express something not on this list, fall back to the closest supported construct (usually a heading + paragraph) instead of inventing syntax.

## Safe — use these freely

- **Headings**: `#`, `##`, `###` on their own line, one space after the hashes.
- **Paragraphs**: plain text, separated from other blocks by exactly one blank line.
- **Bold** `**text**`, *italic* `*text*`, ~~strikethrough~~ `~~text~~`. Don't nest more than one style on the same span.
- **Unordered lists**: `- item`, one level deep. Start a new top-level list rather than indenting a sub-list.
- **Ordered lists**: `1. item`, `2. item`, one level deep — same rule, no nested numbering.
- **Checklists**: `- [ ] item` / `- [x] item`, one level deep, on their own lines (don't mix inside a bullet/numbered list).
- **Fenced code blocks**: triple backticks, optionally with a language tag (` ```ts `). Never use inline four-space-indented code blocks.
- **Tables**: basic GFM pipe syntax —
  ```
  | Column A | Column B |
  | --- | --- |
  | value | value |
  ```
  Keep the separator row as plain `---` per column; don't rely on `:---:` alignment syntax rendering correctly.
- **Links** `[text](https://...)` and images `![alt](https://...)`.
- **Inline code** `` `like this` ``.

## Avoid — these silently break

- **Blockquotes** (`> text`). Known to parse incorrectly — content can end up moved outside the quote block entirely. Use a heading + paragraph instead of quoting something.
- **Nested lists** (a list inside a list item, or a checklist inside a bullet list). Conversion is lossy and often drops the nesting or the items. Keep every list flat; use a new heading to start a logically "nested" group instead.
- **Raw HTML** (`<div>`, `<br>`, `<details>`, etc.). Not interpreted — it shows up as literal text to the reader.
- **Footnotes, definition lists, MDX/Pandoc-style extensions.** Not part of the supported subset; they render as literal text.
- **Horizontal rules** (`---`, `***`) used as a visual divider. Use a heading to start a new section instead of a rule.
- **Multiple consecutive blank lines** or manual indentation to control layout. Normalize to a single blank line between blocks — extra whitespace doesn't survive the round-trip and can shift content into the wrong block.

## Worked example

A well-formed epic/task description using only safe constructs:

```markdown
## Goal
One paragraph describing the outcome.

## Scope
**In:** short list of what's included.
**Out:** short list of what's explicitly excluded.

## Acceptance Criteria
- [ ] First observable outcome
- [ ] Second observable outcome

## Open Questions
- Anything still unresolved
```

Before writing any `description` or doc `content`, re-read what you're about to send and check it only uses the "Safe" constructs above — if it contains a blockquote, nested list, HTML, or a horizontal rule, rewrite that part rather than sending it as-is.
