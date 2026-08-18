---
id: lesson-062-memory-md-files-break-yaml-parsing-with-currentdat
type: lesson
status: active
created: "2026-05-27"
owner: manu
tags: [lesson, dotfiles]
---

# Lesson 062: MEMORY.md files break YAML parsing with `# currentDate` and `---` separators

**Context**: 23 `MEMORY.md` files across all projects had invalid frontmatter. `vault_health --checks frontmatter` reported errors on every one.

**Problem**: Three distinct YAML-breaking patterns:
1. **`# currentDate` after frontmatter** — YAML sees `# currentDate` as a second document (comments after `---` close the first doc).
2. **`> Updated:` block scalar** — YAML interprets `>` as a folded block scalar, then chokes on the next line (e.g., `**Last task:**` doesn't start with a comment or line break).
3. **`---` in body** — Session handoff blocks use `---` as dividers, which YAML interprets as a new document separator.

**Solution**: Wrap the entire body in a single YAML `content: |` literal block scalar under the frontmatter. **Do NOT use a trailing `---`** — that starts a second document. The file is still valid Markdown (Obsidian renders it correctly). Pattern:
```yaml
---
id: "<project>-memory"
type: memory
status: active
tags: [memory, <project>, claude-code]
content: |
  # Title
  
  > block scalar text
  > more text
  ---
  another section
```
Note: the body's `---` is safe because it's inside the `|` scalar.

**Rule**: When a Markdown file needs both valid YAML frontmatter AND body content that may contain YAML-reserved characters (`>`, `---`, `#` at start of line), wrap the body in a `content: |` literal block scalar. Never use a trailing `---` after frontmatter — it starts a second YAML document. For Obsidian compatibility: the `content` key is ignored by Obsidian's markdown renderer (it only reads frontmatter + body), so the file renders identically.

**Tags:** `#yaml` `#frontmatter` `#obsidian` `#memory` `#validation` `#structural-debt`
