---
tags: [spec, verification, templates]
created: "2026-06-16"
---

# Verification - MEMORY-002-agnostic-memory-symlink

## Evidence

- [ ] AC1 (resolve + link, 3 conventions) -> bats `ensure-memory-symlink.bats`
- [ ] AC2 (idempotent + safe no-op) -> bats `ensure-memory-symlink.bats`
- [ ] AC3 (Claude delegation parity) -> bats parity fixture
- [ ] AC4 (dotf init best-effort wiring) -> Go/bats on the init path
- [ ] AC5 (inline function removed, no regression) -> grep + session-start tests

## Test status

- Test suite: `<command> -> <output>` (filled during implementation)
- Manual smoke test: run the script against a real `10_projects/<name>/memory` source
- No regressions in existing session-start tests: yes / no

## Decisions made during implementation

- R1 (contract): flags `--cwd/--target/--project`, exit 0 always, stdout one-liner on link.
- R2: `encode_project_path` stays Claude-side; the script receives the computed target.
- R3 (dotf init role): **DESCOPED 2026-06-16.** Found `dotf init` already links memory at scaffold time (`initrepo.linkMemory`, Go, Claude-encoded). Unifying with this session-start resolver composes with #395 (moves `linkMemory` to `cli/internal/vault`); not touched in this PR to avoid colliding with that in-flight strangler.

## Promotion candidates

- [ ] Lesson for `docs/lessons.md`? Maybe — "agnostic plumbing extracted from a Claude-only hook; the per-agent detail is the caller's, the vault-source resolution is shared."
- [ ] ADR-worthy? No — extraction, not a new architecture decision (the single-sink architecture is decided elsewhere).
- [ ] New pattern? No.

## Archive checklist

- [ ] `proposal.md` frontmatter set to `status: archived`
- [ ] Folder moved to `specs/archive/MEMORY-002-agnostic-memory-symlink/`
- [ ] Issue #402 ticked / closed with PR link
- [ ] Promotions executed (if any)
