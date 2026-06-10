---
id: "OPS-002-bitacora-rollout"
type: spec
status: implementing # draft | implementing | verifying | archived
created: "2026-06-09"
issue: "mlorentedev/dotfiles#258"   # repo#NNN — GitHub issue / Project item that tracks this spec
tags: [spec, proposal]
template_version: "1.0"
---

# OPS-002-bitacora-rollout

## Why

Only `dotfiles` and `knowledge` are wired to the bitácora (GitHub Project #1), and partially: `knowledge` lacks `bitacora-status.yml` (verified 2026-06-09 — assigning an issue there does not flip Status), and the deployed workflow copies have drifted from the canonical templates. Every other repo needs the same three manual steps from the runbook, by hand, per repo. The #258 scope reaffirmation makes this the reusable bitácora baseline for ALL repos — manual per-repo setup does not scale and drifts.

## What

After this PR:

1. **Decision recorded** — board-add mechanism = **per-repo `add-to-project.yml` Action**, not the built-in project Auto-add: git-native (IaC — no UI snowflake config), uniform across N repos, portable via `init-project`, covers PRs (OPS-003 #266), and immune to the plan-dependent auto-add workflow limit.
2. **`scripts/bitacora-rollout.sh`** — one idempotent command that converges every non-archived, non-fork repo: link to the project, upload `BITACORA_PAT`, deploy BOTH canonical workflows via the contents API (create or update-on-diff), backfill open issues + PRs onto the board. `--check` = dry-run; second run reports 0 changes.
3. **Runbook** — §7 registers the script as the rollout mechanism (single repos may still follow the manual steps).

## Out of scope

- The PR review dashboard view — **OPS-003 (#266)**; this PR only guarantees PRs land on the board (auto-add trigger + backfill).
- `init-project` integration for NEW repos (SELF-001) — the script covers existing repos; new repos keep getting wired at creation.
- Field backfill (Priority/Type triage values) — human judgement per item, stays manual.

## Risks / open questions

- **PAT coverage.** `BITACORA_PAT` must reach every repo. Verified 2026-06-09: the decrypted PAT returns HTTP 200 on repos outside dotfiles/knowledge (pollex, kubelab, hive, mail2markdown, resume). (Resolved.)
- **Token scopes for API-pushing workflows.** Requires `workflow` scope on the local `gh` token — verified present. (Resolved.)
- **Forks.** Workflows in fork mirrors never run our CI usefully — discovery excludes `isFork`. (Resolved.)
- **Backfill idempotency.** `gh project item-add` returns the existing item when already on board (verified during HARNESS-010). (Resolved.)

## Acceptance criteria

- [ ] AC1 — `scripts/bitacora-rollout.sh` exists, is executable, passes shellcheck, and supports `--check` (read-only dry-run) plus explicit repo args.
- [ ] AC2 — per repo it converges: project link, `BITACORA_PAT` secret, `add-to-project.yml` + `bitacora-status.yml` (deployed from the canonical `.github/workflows/` copies — no embedded duplicates), and open issue/PR backfill.
- [ ] AC3 — repo discovery covers every non-archived, non-fork repo of the owner when no args are given.
- [ ] AC4 — runbook §7 registers the script as the multi-repo rollout mechanism and records the board-add decision.
- [ ] AC5 — live validation: full rollout executed; a re-run reports 0 changes (idempotence).

## References

- Issue: `mlorentedev/dotfiles#258` (OPS-002); unblocks #266 (OPS-003); decision thread in #258 comments (2026-06-09).
- Runbook: `docs/runbooks/guide-bitacora-setup.md` §7.
- Spec: `specs/HARNESS-010-bitacora-status-lifecycle/` (shipped the canonical workflows this script rolls out).
- ADR: `docs/adr/adr-018-de-vault-task-placement.md` (task state lives in the bitácora).
