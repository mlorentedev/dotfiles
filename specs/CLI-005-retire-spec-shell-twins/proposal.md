---
id: "CLI-005-retire-spec-shell-twins"
type: spec
status: draft # draft | implementing | verifying | archived
created: "2026-06-13"
issue: "dotfiles#339"   # repo#NNN — GitHub issue / Project item that tracks this spec
tags: [spec, proposal]
template_version: "1.0"
---

# CLI-005-retire-spec-shell-twins

> **Naming**: file lives at `<repo>/specs/CLI-005-retire-spec-shell-twins/proposal.md`. `CLI-005-retire-spec-shell-twins` is `AREA-NNN-slug` (e.g. `TOOL-001-secret-drift`).

## Why

<!-- from issue #339: CLI-005: Retire bats + Pester for migrated logic (single `go test` suite) -->

The ADR-020 CLI epic ported `init-spec.sh` → `dot spec init` (CLI-007) and `archive-spec.sh` → `dot spec archive` (CLI-008), then wired `dot` into setup so it installs on PATH (CLI-009 #365). The two shell twins (`init-spec`, `archive-spec`) and their `.ps1` siblings now duplicate logic that lives — and is tested — in Go, yet they remain on PATH and `tests/init-spec.bats` keeps a parallel test suite alive. That is the exact "three coexisting" state ADR-020 §5 forbids and the dual-maintenance debt #339 targets. Retire the twins so the spec lifecycle has a single Go implementation, one `go test` suite, and a smaller Windows/Linux surface.

## What

Remove the migrated shell spec-tooling and repoint every reference to the Go CLI:

- **Delete** `scripts/init-spec.sh`, `scripts/init-spec.ps1`, `scripts/archive-spec.sh`, `scripts/archive-spec.ps1`.
- **Delete** `tests/init-spec.bats` (the only migrated-logic bats; no `archive-spec.bats` and no Pester for these ever existed — `dot spec init/archive` `go test` is the replacement).
- **Repoint references** from the shells to `dot spec init` / `dot spec archive`:
  - `AGENTS.md` §389 (shell-fallback line) and §406 (the Discipline Gate step 2).
  - `tests/agents-md.bats:34` — the `grep -qF 'init-spec'` assertion → assert `dot spec init` instead.
  - `harness/skills/spec/SKILL.md` — the Windows/non-interactive invocations of `init-spec.ps1` / `archive-spec.ps1`.
  - `scripts/check-spec-gate.sh:193` — the "Create a spec folder: ./scripts/init-spec.sh" hint string.
  - `docs/adr/dotfiles-architecture-map.md` — the "where does X live" rows naming the scripts.

After this PR, `dot spec` is the single, on-PATH entry for the spec lifecycle on every platform; the shells are gone.

## Out of scope

- Any change to the Go CLI behaviour — `dot spec init/archive` already shipped (CLI-007/008) and is unchanged here.
- `check-spec-gate.sh` itself (the CI spec-gate) stays — it is Linux-only bootstrap/CI tooling, not migrated logic; only its hint string updates.
- Historical artifacts left as-is: `CHANGELOG.md`, the `docs/adr/audit-*` point-in-time audits, and archived/active spec docs that mention the scripts (provenance records, not live references).
- `setup-windows.ps1` install of `dot` (CLI-009 follow-up, Windows-empirical) — independent.

## Risks / open questions

- **Hard dependency on CLI-009 #365 (resolved sequencing).** `scripts/` is on PATH via the RC files, so `init-spec`/`archive-spec` are live commands; they can only be deleted once `dot` is their installed on-PATH replacement. **Implementation is gated on #365 merging** and is the reason this is a separate PR rather than folded into CLI-007/008. The spec is written now; deletions land after #365.
- **`agents-md.bats` coupling.** That test greps `AGENTS.md` for `init-spec`; it must be updated in lockstep with the AGENTS.md edit or it fails. Covered as a task.
- **Stale machines.** A machine that has not re-run setup post-#365 will have neither the shells nor `dot`. Acceptable for a personal dotfiles repo (standard "re-run setup after pulling"); noted in the AGENTS.md edit.
- **No behavioural regression risk in the Go path** — the twins are byte-verified equivalent (CLI-007/008 parity smokes); deleting them removes only the duplicate.

## Acceptance criteria

Observable outcomes. Each must be testable.

- [ ] The 4 shell files and `tests/init-spec.bats` no longer exist in the tree.
- [ ] No live reference to `init-spec`/`archive-spec` remains outside historical artifacts: `grep -rE 'init-spec|archive-spec'` returns only CHANGELOG, audits, and spec-provenance docs.
- [ ] `AGENTS.md`, `check-spec-gate.sh`, and the spec `SKILL.md` direct users to `dot spec init` / `dot spec archive`.
- [ ] `tests/agents-md.bats` asserts the `dot spec` guidance and passes.
- [ ] Full bats suite green (minus the deleted `init-spec.bats`); `shellcheck` clean; `dot spec init/archive` still work end-to-end.

## References

- GitHub issue: `dotfiles#339` (work-gate)
- Epic: ADR-020 §5 (strangler-fig: delete the pair on contact) — `docs/adr/adr-020-tooling-cli-go-convergence.md`
- Predecessors: CLI-007 (`dot spec init`), CLI-008 (`dot spec archive`), CLI-009 #365 (setup installs `dot` — the unblocker)
- Testing strategy: `docs/adr/adr-004-bats-testing.md` (bats stays for bootstrap/profile, not migrated logic)
