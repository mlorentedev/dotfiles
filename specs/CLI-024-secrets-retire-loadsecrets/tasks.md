---
tags: [spec, tasks, secrets, shell]
created: "2026-06-25"
---

# Tasks - CLI-024-secrets-retire-loadsecrets

> TDD order. One task = one focused commit. Three PRs (B1/B2/C) against this spec.

## Setup

- [x] Branch created from main: `feat/nan-scripts-via-dotf-secrets` (B1)
- [x] `proposal.md` complete; acceptance criteria testable
- [x] No open questions left in `proposal.md` "Risks / open questions"

## PR-B1 — nan-* scripts → dotf secrets show (this branch)

- [ ] **RED**: `tests/nan-scripts-secrets.bats` — assert the 3 scripts do NOT
  source `load-secrets` / call `secrets_refresh`, and DO self-fetch via
  `dotf secrets show nan-api-key` guarded by `command -v dotf`.
- [ ] **GREEN**: edit `scripts/nan-bench.sh`, `nan-debug.sh`, `nan-quality-bench.sh`
  — replace the `load-secrets` fallback with the `dotf secrets show` idiom; update
  the error hint to `dotf secrets run -- <script>`.
- [ ] **Verify**: `bats tests/nan-scripts-secrets.bats` green; `bash -n` clean.
- [ ] `verification.md` + `features.json` (`f1` = AC1) filled with evidence.
- [ ] PR-B1 opened referencing #587 (title `feat(secrets): …` → triggers 0.19.0).

## PR-B2 — setup eager-load → dotf secrets show (later branch)

- [ ] Enumerate every mid-setup consumer of the eager-loaded vars.
- [ ] Replace the eager source with scoped `dotf secrets show <id>`; update
  `tests/setup-windows.bats` parity assertions.

## PR-C — delete twins + env-mapping.conf (later branch)

- [ ] Consumer sweep clean → `git rm` twins + `env-mapping.conf` + `load-secrets.bats`.
- [ ] Drop setup chmod (`setup-linux:101`) + deploy block (`setup-windows:1565-1574`).
- [ ] Archive the fine-grained-PAT lesson in `docs/runbooks/guide-bitacora-setup.md`.
- [ ] Close #587.

## Machine-readable features

See sibling `features.json`. `f1` (AC1, B1) is verifiable now; `f2`/`f3` (B2/C)
land with their PRs.
