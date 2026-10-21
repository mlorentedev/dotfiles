---
id: "CLI-026-dotf-harness-engine"
type: spec
status: implementing # draft | implementing | verifying | archived
created: "2026-06-21"
issue: "mlorentedev/dotfiles#495"   # repo#NNN — GitHub issue / Project item that tracks this spec
tags: [spec, proposal]
template_version: "1.0"
---

# CLI-026: dotf harness engine

> AUDIT-007 Phase B / PR10. Parent ADR: `docs/adr/audit-007-cli-convergence-state.md`. Supersedes the bash engine `scripts/compile-harness.sh` (spec: `specs/ENGINE-001-deploy-engine-core/`). Absorbs #909 (CLI-035), which duplicated this issue.
>
> **Re-planned 2026-10-20** against `origin/main` `856f49fc` (owner decision on #495, 2026-10-10 and 2026-10-20). The original plan predates the Go verbs that have since landed; this version targets what remains.

## Why

The harness deploy engine exists twice: `scripts/compile-harness.sh` (Linux/macOS, 1563 lines, 1991 lines of bats) and a hand-maintained twin inside `setup-windows.ps1` (`Deploy-SkillRecord` and its helpers). They drift, and they already have: the shell matches a record's `targets:` as a substring and the PowerShell twin by word boundary (#1733, audit C21); the shell's prompt render drops every `---` line in a prompt's body and the twin keeps them; the catalog separator differs; `agents.deploy` is not ported to Windows at all (#1387, audit C23); and a twin missed HARNESS-069's provenance strip for one PR cycle (#909). A single Go engine collapses the split into one cross-OS binary — the AUDIT-007 strangler end-state (ADR-020, ADR-021) — so this class of platform-skew bug cannot recur.

### What already landed in Go (not in scope)

- `dotf harness instructions` (#2044): the agent instruction files. `deploy_instructions` delegates to it.
- `dotf harness presence` (#1368): the persona roster, on every OS.
- `dotf harness refresh` (#2175): a vault-currency gate **wrapping** `compile-harness.sh --refresh`; the render itself is still bash.
- `dotf harness mirror` and `dotf harness bind`.

### Callers of the bash engine today

`setup-linux.sh:615` (`--deploy`), `.github/workflows/ci.yml:171` (`--check`), `cli/internal/converge/harness.go:153` (`--deploy`, the `records-harness` step, #2025), `cli/internal/cmd/harness_refresh.go` (`--refresh`), and `dotf doctor`'s harness drift check. `setup-windows.ps1:1501` calls the `Deploy-SkillRecord` twin.

## What

Per-surface `dotf harness` verbs, the shape `instructions` and `presence` already set, then one composing verb:

- **`dotf harness skills`** — render committed skill records to every `skills.deploy[]` target (skill / command / prompt), exact `targets:` matching (inline and block list), prune generated outputs whose record or target is gone, report an unmarked copy that shadows a record. Then the copilot skill catalog.
- **`dotf harness agents`** — render agent records (`agents.deploy[]`) with their resolved model and capability lines.
- **`dotf harness doctrine`** — the compact doctrine payload for capped targets (agy, codex).
- **`dotf harness deploy`** — instructions, skills, agents, presence, doctrine, in the order the shell runs them. Replaces `compile-harness.sh --deploy` everywhere.
- **`dotf harness check`** — the offline drift check (`--check`), with `--against-vault` flagging a committed record stale against its vault source (the gap #1545 describes).
- **`dotf harness refresh`** — the render half of `--refresh` in Go, behind the existing vault-currency gate.

End state: one engine, invoked identically on every OS; `scripts/compile-harness.sh`, `tests/compile-harness*.bats` and `Deploy-SkillRecord` (with `Test-SkillTargetsAgent`, `Get-SkillField`, `Test-SkillFrontmatterKeyKept`, `Convert-SkillRecord`) are deleted.

## Out of scope

- **Records ↔ deployed-dir drift** — CLI-019 / #488 (`dotf doctor`), the downstream half.
- Changing the `harness/manifest.json` schema, the slugify rules or the `BEGIN/END HARNESS` block format.
- The vault SSOT content itself.
- Removing auxiliary files a record no longer carries from a deployed skill directory: neither twin does it today (#2851, after slice 3).

## Decisions

- **Per-surface verbs, then `deploy`** (2026-10-20). The original spec promised `refresh/deploy/check` only. `instructions` and `presence` already exist as per-surface verbs that the shell delegates to; the remaining surfaces follow that shape so each slice lands under the ~300-line cap and is callable on its own, and `deploy` composes them.
- **Parity oracle is a bats test, not a committed golden corpus.** `tests/harness-skills-parity.bats` runs the bash engine and the Go verb over the committed records into two throwaway HOMEs and diffs them, in CI's `test` job with the `DOTF_BIN` it already builds (CI-012). The oracle reads the live records, so it cannot go stale, and it is deleted with the script.
- **Declared divergences from the oracle** — each a defect in one twin, fixed in Go, not carried over:
  - prompt render keeps the body's `---` lines (the shell dropped all of them; the twin kept them);
  - `targets:` is matched by exact name, inline or block (#1733, C21; no committed record declares `targets:` today, so no output changes);
  - outputs are written only when they differ, so a converged run reports `0 deployed`;
  - records are walked in byte order of their names, not the locale's collation (glibc and macOS order hyphenated names differently, which matters for the catalog) nor the shell glob's (`*/` sorts `a-b/` before `a/`, lesson 505);
  - an END marker line with anything after the marker is refused by every GENERATED-region writer: the catalog skips the file as unmarked, doctrine fails that target (the shell validated it, then dropped the rest of the file).
- **Line endings: normalize in Go.** CRLF is read as LF and every output is LF, so a CRLF checkout renders the same bytes and sha as an LF one.
- **Cutover: Go first, then rewire.** A verb lands alongside the bash function it replaces; the rewire (shell delegates, PowerShell twin deleted) follows once a release carries the verb, since setup calls the pinned `dotf` (#1814 class).

## Acceptance criteria

- [ ] **AC1** `dotf harness skills` writes what `compile-harness.sh --deploy` writes for the skills block, byte-for-byte except the declared divergences, over the committed records (`tests/harness-skills-parity.bats`).
- [ ] **AC2** For identical inputs, output is byte-identical (LF, same sha) on Windows and Linux: a CRLF record renders the LF bytes.
- [ ] **AC3** The copilot skill catalog is written by Go, and the parity oracle covers it (seeded instructions file on both sides).
- [ ] **AC4** `dotf harness agents`, `dotf harness doctrine` and `dotf harness deploy` exist, with parity against the oracle for each surface.
- [ ] **AC5** `dotf harness check` runs offline and is green in CI on a fresh checkout; `dotf doctor`'s harness drift check calls it on every OS, so the Windows SKIP from #804 is retired.
- [ ] **AC6** `dotf harness check --against-vault` flags a record whose vault source changed but was never refreshed (regression test: mutate a vault skill, assert a non-zero exit).
- [ ] **AC7** `dotf harness refresh` renders in Go; no caller runs `compile-harness.sh --refresh`.
- [ ] **AC8** `setup-windows.ps1`, `setup-linux.sh` and the `records-harness` converge step invoke the Go deploy; `Deploy-SkillRecord` and its helpers are removed; `install-dotf` → deploy ordering is preserved.
- [ ] **AC9** `scripts/compile-harness.sh` and `tests/compile-harness*.bats` are deleted; a guard-grep finds no reference in `setup-*`, `ci.yml`, `cli/` or the profile.

## References

- Parent ADR: `docs/adr/audit-007-cli-convergence-state.md` (row 10); ADR-020 §5 (strangler-fig), C7 (bootstrap stays shell)
- Superseded engine spec: `specs/ENGINE-001-deploy-engine-core/proposal.md`
- Duplicate absorbed: #909 (CLI-035)
- Defects this closes or touches: #1733 (C21), #1387 (C23), #1493 and #1545 (C45, `--against-vault`), #1502 (`--check` blind to a reverting refresh), #1876 (`deploy_doctrine` length)
- Porting model: `cli/internal/harness/instructions.go`, `presence.go`
