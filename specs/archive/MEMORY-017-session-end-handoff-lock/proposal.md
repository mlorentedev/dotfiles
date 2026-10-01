---
id: "MEMORY-017-session-end-handoff-lock"
type: spec
status: archived # draft | implementing | verifying | archived
created: "2026-09-30"
issue: "mlorentedev/dotfiles#1931"   # repo#NNN — GitHub issue / Project item that tracks this spec
tags: [spec, proposal]
template_version: "1.0"
wip_override: "P1 Windows race discovered after HARNESS-088 merged; this closes shipped concurrency debt before starting unrelated work (22 active, limit 10, 2026-09-30)"
---

# MEMORY-017: Lock SessionEnd reads against handoff replacements

## Why

<!-- from issue #1931: MEMORY-017: Lock SessionEnd reads against handoff replacements -->

HARNESS-088 (#1886) serializes concurrent `dotf mem handoff-write` writers, but
`mem.SessionEnd` still opens the same `MEMORY.md` outside that lock. On Windows,
that read handle can make the writer's atomic rename fail with `Access is
denied`, so a valid handoff can fail during session shutdown.

## What

The canonical MEMORY.md lock becomes part of the `mem` domain instead of the
command package. Both `handoff-write` and `SessionEnd` acquire the same lock key;
the archive reader waits for an active writer and resumes after release.

## Out of scope

- Changing handoff parsing, thread identity, or fallback journal attribution
  (tracked by #1920, #1921, #1928, #1929, and #1930).
- Resolving the Windows junction measurement gap already recorded by HARNESS-088.

## Risks / open questions

- The helper move must preserve HARNESS-088's lock directory, canonical-path
  hashing, timeout, and Windows case normalization byte-for-byte.
- A lock timeout returns an error, but the `mem session-end` command already
  swallows hook errors by contract so session shutdown remains non-fatal.
- No open design question remains.

## Acceptance criteria

Observable outcomes. Each must be testable.

- [x] **AC1:** `handoff-write` and `SessionEnd` use one canonical-path lock
      helper and therefore one lock key for the same `MEMORY.md`.
- [x] **AC2:** a `SessionEnd` invocation started while the writer lock is held
      does not read or archive until the lock is released, then completes.
- [x] **AC3:** HARNESS-088's concurrent-writer and symlink-path lock regressions
      remain green, and both Windows and Unix lock implementations compile.

## References

- Bitácora: #1931.
- Originating change: #1886; round-2 review finding reproduced in the prior
  worktree before #1886 merged.
- Related debt: #1920, #1921, #1928, #1929, #1930.
