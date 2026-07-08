---
id: "HARNESS-040-doctor-fix-drift-repair"
type: spec
status: archived # draft | implementing | verifying | archived
created: "2026-06-25"
issue: "mlorentedev/dotfiles#551"   # repo#NNN — GitHub issue / Project item that tracks this spec
tags: [spec, proposal]
template_version: "1.0"
---

# HARNESS-040-doctor-fix-drift-repair

## Why

`/handoff` writes the session continuity block to the knowledge vault, and on Claude that vault dir is surfaced into the agent's per-project memory through a junction at `~/.claude/projects/<encoded-cwd>/memory`. Today that junction is created **only** by a silent SessionStart hook (now `memlink` via the Claude adapter); on a machine where it is missing, `/handoff` orphans the continuity block in `.claude` instead of landing it in the vault (observed 2026-06-19, and still broken on this machine). `dotf doctor` is the one place a user runs to find and repair post-setup drift, but it cannot see or fix this junction. Separately, the doctor's env-contract sweep is hardcoded to the `linux` OS key, so on Windows it checks the wrong PATH entries and per-OS defaults and reports a **false-positive drift banner every session** (e.g. `C:\Users\mlorente/.dotfiles/scripts not in PATH`, a Linux entry expanded with a Windows home). Both are per-machine setup gaps that the no-CI model relies on `doctor` to catch.

## What

After this PR:

1. `dotf doctor` reports the auto-memory↔vault junction as a first-class check: **PASS** when the link exists and resolves to the project's vault memory source, **FAIL** (with a `run --fix` hint) when it is missing while a vault source exists, **SKIP** when no vault source exists (a valid state). `dotf doctor --fix` recreates the junction idempotently via the shared `memlink` primitive — the same noun the SessionStart adapter uses — so `/handoff` lands the continuity block in the vault, not in `.claude`.
2. The env-contract sweep (`checkContractEnvVars`, `checkContractPath`) selects its OS key (`linux` / `windows` / `darwin`) from the injected `System.GOOS` instead of the hardcoded `"linux"`. On Linux behavior is unchanged; on Windows the doctor now checks the `windows` PATH entries and defaults, ending the false-positive drift that the `--quick` session-start banner surfaces.

## Out of scope

- **Hive MCP venv repair** (issue #551 part 3: `No module named 'rich.traceback'`). Repairing a Python venv from the Go doctor is a cross-language concern; tracked in its own issue (see References) per atomic-PR discipline.
- **`machine.json` scaffolding / reconciliation** for non-standard repo layouts (e.g. `~/Projects/Workspace/`). The OS-aware fix above removes the false drift; actively rewriting the per-machine override file is a separate config concern, deferred unless the OS-aware fix proves insufficient.
- Changing `memlink`'s link-creation semantics (junction vs symlink), the SessionStart hook, or the `--quick` mode's check *set* (only the OS key its existing checks read changes).

## Risks / open questions

- **Blast radius into session-start.** `checkContractEnvVars`/`checkContractPath` are also run by `dotf doctor --quick` (the SessionStart hook). Making them OS-aware changes the Windows banner output. This is the intended fix, but tests must cover both GOOS branches so the Linux/POSIX byte-equivalence expectations are preserved (Linux output is unchanged).
- **Junction target resolution must match the adapter exactly.** The check must compute the same `~/.claude/projects/<encoded-cwd>/memory` target and the same vault source precedence as `mem/session_start_detect.go`, or doctor and session-start would disagree. Mitigation: reuse `memlink` for both detection and repair; share the encode logic rather than re-deriving it.
- **memlink is best-effort silent by contract** (a failed link must never crash a session). Doctor needs a *non-mutating* status to report PASS/FAIL without side effects, and a loud repair under `--fix`. Resolution: add a thin `memlink.Status` (read-only) beside `Ensure`; do not weaken `Ensure`'s resilience contract.

## Acceptance criteria

- [ ] On a machine with a missing junction but a present vault memory source, `dotf doctor` reports the junction check as FAIL; `dotf doctor --fix` recreates it and a re-run reports PASS (idempotent: a second `--fix` neither errors nor duplicates state).
- [ ] When no vault memory source resolves for the project, the junction check is SKIP (not FAIL) under both `--fix` and plain modes.
- [ ] With `System.GOOS = "windows"`, `checkContractPath` checks the `windows` `required_path_entries` and `checkContractEnvVars` resolves `windows` defaults; with `GOOS = "linux"` the output is byte-for-byte unchanged from today.
- [ ] `cli` `go test ./...` passes; no unrelated changes in the diff.

## References

- Issue: `mlorentedev/dotfiles#551` (HARNESS-040)
- Deferred follow-ups: #574 (part 3 — Hive venv repair, cross-language), #575 (`memlink.createLink` robustness for path components with bare cmd delimiters)
- Code: `cli/internal/memlink/memlink.go` (the shared primitive), `cli/internal/mem/session_start_detect.go` (the adapter consumer), `cli/internal/doctor/checks_contract.go` (OS-key sites), `cli/internal/doctor/checks_vault_hooks.go` (the verify/repair check pattern this mirrors)
- ADR: `docs/adr/adr-025-cross-machine-paths.md` (the path seam), ADR-021 (the doctor consolidation)
