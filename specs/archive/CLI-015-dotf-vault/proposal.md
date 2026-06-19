---
id: "CLI-015-dotf-vault"
type: spec
status: archived # draft | implementing | verifying | archived
created: "2026-06-16"
issue: "mlorentedev/dotfiles#388"   # repo#NNN — GitHub issue / Project item that tracks this spec
tags: [spec, proposal]
template_version: "1.0"
---

# CLI-015-dotf-vault

> **Naming**: file lives at `<repo>/specs/CLI-015-dotf-vault/proposal.md`. `CLI-015-dotf-vault` is `AREA-NNN-slug` (e.g. `TOOL-001-secret-drift`).

## Why

<!-- from issue #388: CLI-014 follow-up: restore work-SDK vault-entry generation in dotf vault (ADR-021 step 3) -->

CLI-014 folded repo scaffolding into `dotf init` and deleted the `init-*.sh` twins, but it had to **drop** one mode that did not belong there: `init-project.sh --work-sdk <family> <component>`, which scaffolds a **vault-only** entry under `50_work/45-development/<family>/<component>/` (no repo). That capability is gone from both OSes and #388 tracks restoring it in its correct home — the `dotf vault` noun (ADR-021 step 3). Establishing that noun also exposes a latent duplication: `dotf init` *already* scaffolds a vault entry (the personal-project entry under `10_projects/<repo>/`, via `initrepo.WriteVaultEntry`), so "scaffold a vault entry" is a concern that already lives — half-formed and repo-coupled — inside the repo-init flagship. This spec makes `dotf vault` the single home for **vault-entry scaffolding**, with the **entry type** as the dimension.

## What

A `dotf vault` Cobra noun (parent runnable via `RunE: cmd.Help`, the `dotf init` idiom) whose subcommands scaffold a vault entry by type, from **templates embedded in the binary** (`//go:embed`, drift-tested against the vault SSOT — the `dotf spec` / `dotf init` precedent):

- **`dotf vault work <family> <component>`** *(the #388 deliverable)* — scaffolds the work-SDK entry `50_work/45-development/<family>/<component>/`: `00-context.md` (with `source_path` to the real repo), `memory/MEMORY.md`, and the parent `<family>/00-context.md` (created only if absent). Restores the deleted `--work-sdk` capability, cross-platform with no `.ps1` twin.
- **`dotf vault project <repo>`** — the standalone personal-project entry `10_projects/<repo>/` (`00-context.md` + `10-roadmap.md` + `memory/MEMORY.md`), **extracted** from `dotf init`'s inlined `WriteVaultEntry`. `dotf init` is rewired to **call** `dotf vault project` instead of carrying its own copy — one renderer, two entry points.

Shared mechanics across both subcommands:

- **Skip-if-present + `--force`.** Never clobber a `00-context.md` / `MEMORY.md` that may have accumulated real content; `--force` regenerates. (Matches today's `WriteVaultEntry`.)
- **Vault absent = error**, not auto-skip. `dotf vault` is a vault-only command — a missing vault is a usage error naming `$VAULT_PATH`, unlike `dotf init` where the vault entry is one optional step that degrades to `[skip]`.
- **Drift-tested templates.** New SSOT files `00_meta/templates/work-sdk-{context,memory,family}.md`, embedded and `bytes.Equal`-guarded against the vault (skip-when-absent, per ADR-013), exactly like `cli/internal/spec` and `cli/internal/initrepo`.

**Delivered as a 2-PR sequence** (atomic-PR discipline; the whole change is a multi-PR refactor touching a public contract):

| PR | Scope | Work-gate | Ships |
|----|-------|-----------|-------|
| **PR1** | `dotf vault` noun + `vault work` + work-SDK SSOT templates + drift-test + shared `writeEntry` core | #388 | the restored capability — closes #388 |
| **PR2** | extract `WriteVaultEntry` → `vault project`; rewire `dotf init` to call it; delete the `initrepo` copy (strangler, no coexistence) | #395 | the unification — behavior-preserving |

## Out of scope

- The other `dotf vault` responsibilities named in ADR-021 step 3 — `knowledge-crystallize`, `vault-maintenance-weekly`, `obs-cli` / `vault-health`. Those are separate verbs/PRs under the same noun, later.
- The **content/schema** of the entries themselves (front-matter fields, section layout). This ports the existing templates faithfully; redesigning them is a vault concern, not this port.
- Any `.ps1` — Go is cross-platform by construction; there is no Windows twin to write or delete here.
- `dotf init`'s repo-scaffolding behavior. PR2 changes only *where the vault-entry code lives*, not what `dotf init` produces (guarded by its existing golden/orchestrator tests).

## Risks / open questions

- **PR2 is a behavior-preserving refactor of `dotf init`, just shipped (CLI-014).** Risk: drift in the generated `10_projects/` entry. Mitigation: PR2 reuses the *same* templates and token-replacement; the existing `initrepo` orchestrator/vault tests are the parity oracle; golden-compare before/after.
- **Drift-test value vs. ceremony.** The work-SDK templates have no independent consumer *today* (the session hook only *discovers* existing entries). Counter: promoting them to vault SSOT gives the interim manual process a real template to copy from, and keeps `dotf vault` uniform with every other scaffolder. Accepted: consistency wins (Standing Order #6).
- **New vault SSOT templates require a vault commit** under the isolated-worktree discipline (master, `--ff-only`, never a branch). Sequenced inside PR1's work, not deferred.
- **Follow-up issue for PR2** opened as #395 (the unification), so the multi-PR sequence is visible in the bitácora from the start.
- **`linkMemory` (the Claude auto-memory symlink) is `10_projects`-specific** (it encodes a *repo* path). `vault work` entries map to a work repo via `source_path`, not a local checkout — so `vault work` does **not** create that symlink (the session hook handles work-SDK discovery). Confirmed: symlink logic stays on the `project` path only.

## Acceptance criteria

Observable outcomes. Each must be testable.

- [ ] `dotf vault work <family> <component>` scaffolds `50_work/45-development/<family>/<component>/` with `00-context.md` + `memory/MEMORY.md`, and creates `<family>/00-context.md` only when absent — restoring the deleted `--work-sdk` output (byte-comparable to the old script's, modulo date).
- [ ] `dotf vault project <repo>` scaffolds `10_projects/<repo>/` identically to today's `dotf init` vault step; `dotf init` produces a byte-identical entry after being rewired to call it (parity test).
- [ ] Both subcommands skip-if-present and regenerate under `--force`; a missing vault is a non-zero error naming `$VAULT_PATH` (not a silent skip).
- [ ] `00_meta/templates/work-sdk-*.md` are embedded and drift-tested (`go test` fails on divergence where the vault is present; skips where absent).
- [ ] `dotf vault` parent is a first-class "Available Command" (runnable via `cmd.Help`), listed by `dotf --help`.
- [ ] The `initrepo.WriteVaultEntry` copy is deleted in PR2 (guard-grep leaves only provenance); `go test ./...` green across both PRs.

## References

- ADR: `docs/adr/adr-021-cli-orchestration-roadmap.md` (step 3 — `dotf vault`) · `docs/adr/adr-022-dotf-init-flagship.md` (records `--work-sdk` as a `dotf vault` concern)
- Work-gate: `mlorentedev/dotfiles#388` (CLI-014 follow-up) · origin: CLI-014 (#389, removed there) · epic #131
- Precedent: `cli/internal/initrepo/vault.go` (`WriteVaultEntry`, the extraction target) + `cli/internal/initrepo/drift_test.go` + `cli/internal/spec/drift_test.go` (embed + vault-drift guard)
- Removed source: `scripts/init-project.sh --work-sdk` at `git show e861b41~1:scripts/init-project.sh` (the template heredocs to port)
