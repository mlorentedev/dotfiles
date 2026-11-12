---
id: "MEMORY-018-handoff-line-budget"
type: spec
status: implementing # draft | implementing | verifying | archived
created: "2026-11-12"
issue: "mlorentedev/dotfiles#3227"   # repo#NNN — GitHub issue / Project item that tracks this spec
tags: [spec, proposal]
template_version: "1.0"
owner: manu
---

# MEMORY-018: handoff-write holds MEMORY.md to the lines a session start loads

## Why

<!-- from issue #3227: handoff-write: hold MEMORY.md to the 200-line load cap and archive finished threads -->

Claude Code loads the first 200 lines of MEMORY.md at session start. `dotf mem handoff-write` refuses a write past 25,000 bytes (#1477), but a file can be well under that and still past 200 lines. The handoff section is the file's last, so the newest threads are the ones cut off. On 2026-11-05 the dotfiles MEMORY.md was 20.7 KB and 224 lines, and every session start lost its last 24 lines. On 2026-11-12 it was 239 lines. After three threads were archived it was 217 lines, still 17 past the cap.

## What

`dotf mem handoff-write` refuses a write that takes MEMORY.md past 200 lines, the same way it already refuses one past 25,000 bytes. The error says how many lines the write produces and the cap the harness loads. It names the least recently updated threads to archive and how to archive each, as the byte refusal does. A write that keeps the line count the same or lowers it still passes, so a file already past the cap can be brought back under it.

## Out of scope

- Archiving finished threads on write (#3227's second proposal). That rewrites MEMORY.md on a thread's behalf, which needs the owner's go; it is #3393.
- Any change to the byte budget, its message or the thread store's entry shape.
- Trimming the dotfiles MEMORY.md itself, which is vault work, not a code change.

## Risks / open questions

- A file already past the cap refuses every write that adds a line until threads are archived. That is the intended pressure, the same as the byte cap; the error names what to archive. Resolved: accept.
- "Line" means a newline-terminated line, plus a final line with no newline. That is what a 200-line read sees. Resolved.

## Acceptance criteria

- [ ] AC1: A write that takes a MEMORY.md under 25,000 bytes past 200 lines is refused. The error states the line count and the 200-line cap and names the least recently updated threads to archive.
- [ ] AC2: A write that does not grow the line count of a file already past 200 lines passes. A write that keeps the file at or under 200 lines passes.
- [ ] AC3: The byte budget still refuses on its own terms. A write past 25,000 bytes but under 200 lines is refused with the byte message.

## References

- #3227 (this spec's gate), #3393 (proposal 2), #1477 (byte budget)
- `cli/internal/mem/thread_store.go` `CheckMemoryBudget`; caller `cli/internal/cmd/mem_handoff.go`
