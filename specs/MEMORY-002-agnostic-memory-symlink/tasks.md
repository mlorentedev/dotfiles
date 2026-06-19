---
tags: [spec, tasks, templates]
created: "2026-06-16"
---

# Tasks - MEMORY-002-agnostic-memory-symlink

> TDD order. One task = one focused commit. Behaviour-preserving extraction of proven logic — low design risk.

## Setup

- [x] Branch created from main: `feat/memory-symlink-extract`
- [x] `proposal.md` complete; acceptance criteria testable
- [x] R1 (contract) + R2 (no `encode_project_path` dependency) resolved; R3 (dotf init role) recommended, confirm at the wiring task

## Implementation (TDD)

- [ ] Failing bats `tests/ensure-memory-symlink.bats`: `--cwd <repo> --target <path>` with a `10_projects/<name>/memory` fixture → expect symlink created (AC1) — **RED**
- [ ] Implement `scripts/ensure-memory-symlink.sh`: arg parse (`--cwd/--target/--project`), resolve vault source (3 conventions, extracted verbatim), idempotent symlink, exit 0 always → **GREEN** (AC1)
- [ ] bats: `CWD/memory` (in-vault) + `50_work/45-development/<family>/<comp>/memory` (nested) fixtures resolve (AC1, all conventions)
- [ ] bats: idempotency + safety — already-linked no-op, non-empty real dir no-op (no clobber), no source no-op exit 0 (AC2) — **RED → GREEN**
- [ ] Refactor `claude-session-start.sh`: replace the inline `ensure_memory_symlink` body with a call to the script (compute encoded target, pass `--target`); remove the duplicated logic (AC3, AC5)
- [ ] bats parity: Claude-shaped fixture via the delegating path produces the same symlink (AC3)
- [x] ~~Wire `dotf init`~~ **DESCOPED (R3)**: `dotf init` already links memory (`initrepo.linkMemory`); unification composes with #395's move of that code to `cli/internal/vault`. Not touched here (AC4)
- [ ] shellcheck clean on the new script; cross-OS note (R4) recorded in the script header

## Closing

- [ ] Every acceptance criterion covered by a test
- [ ] `features.json` entries non-vacuous
- [ ] Lint passes (shellcheck), `go build ./...` if dotf init touched
- [ ] No unrelated changes (no scope creep — per-agent hooks are MEMORY-001-mirror, NOT this PR)
- [ ] `verification.md` filled
- [ ] PR opened referencing this spec folder + issue #402

## Machine-readable features

`features.json` is the harness-facing contract: each AC → ≥1 feature with `id`, `behavior`, `verification`, `state`, `evidence`. The agent cannot write `"state": "passing"` — only the harness, after running `verification` with exit 0, may.
