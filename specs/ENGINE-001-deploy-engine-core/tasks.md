---
tags: [spec, tasks, engine-001, harness-001]
created: "2026-05-28"
---

# Tasks - ENGINE-001-deploy-engine-core

> TDD order. One task = one focused commit. Freeze once `implementing` starts.

## Setup

- [x] Branch `feat/engine-001-deploy-core` off `main`
- [x] `proposal.md` complete; acceptance criteria testable
- [x] Manifest format locked: JSON (`jq` Linux / `ConvertFrom-Json` Windows)
- [x] Decisions recorded: id=ENGINE-001 · D2=source-of-record · D3=section-anchor · guard=healthcheck

## Implementation (TDD)

- [ ] **(red)** `tests/compile-harness.bats`: `--check` exits non-zero when no record/blocks exist yet
- [ ] **(green)** `harness/manifest.json` + `compile-harness.sh` skeleton (arg parse: `--refresh` | `--check` | `--help`)
- [ ] **(red)** `--check` exits 0 on a refreshed fixture, non-zero after a deployed block is hand-edited (AC1)
- [ ] **(green)** implement `--check`: render block from `harness/enforced/<id>.md` → diff vs marker region in each target file
- [ ] **(red)** `--refresh` is idempotent; blocks carry BEGIN/END markers + source + sha256 (AC2, AC7)
- [ ] **(green)** implement `--refresh`: extract vault section by anchor → write `harness/enforced/<id>.md` → inject marker region (stable sort by id, FM3)
- [ ] **(red)** AC3 — `--check` works with `VAULT_PATH` at an empty dir (offline render from record)
- [ ] **(green)** ensure `--check` never touches the vault; `--refresh` preflights vault presence (actionable abort)
- [ ] **(red)** AC4 line-cap breach fails; AC5 missing END marker fails loudly (fixtures)
- [ ] **(green)** post-inject 80-line assertion + marker-integrity validation (no silent append)
- [ ] **(red)** AC6 — `healthcheck.sh` flags a tampered deployed block (offline)
- [ ] **(green)** add `check_harness` to `scripts/healthcheck.sh` (calls `compile-harness.sh --check`)
- [ ] wire `setup-linux.sh` to run `compile-harness.sh --refresh` in the deploy phase
- [ ] **(refactor)** extract helpers, shellcheck clean, dedupe

## Closing

- [ ] Every AC covered by ≥1 test in `tests/compile-harness.bats`
- [ ] `features.json` emitted; each AC → executable `verification` command
- [ ] `shellcheck scripts/compile-harness.sh` clean; `bash -n` + `zsh -n` pass
- [ ] `bats tests/compile-harness.bats` green (targeted — NOT `tests/*.bats`, hangs per #167)
- [ ] `verification.md` filled with evidence
- [ ] PR opened referencing this spec + the umbrella

## Machine-readable features

Emit `features.json` (sibling) per [[pattern-feature-list-as-primitive]]: one feature per AC with an executable `verification`. Only the harness may set `"state": "passing"` after capturing exit 0.
