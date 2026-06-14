---
id: "CLI-007-dot-spec-init"
type: spec
status: archived # draft | implementing | verifying | archived
created: "2026-06-13"
issue: "dotfiles#358"   # repo#NNN — GitHub issue / Project item that tracks this spec
tags: [spec, proposal]
template_version: "1.0"
---

# CLI-007-dot-spec-init

> **Naming**: file lives at `<repo>/specs/CLI-007-dot-spec-init/proposal.md`. `CLI-007-dot-spec-init` is `AREA-NNN-slug`.

## Why

<!-- from issue #358: CLI-007: port init-spec scaffold to `dot spec init` (Go twin) -->

The ADR-020 CLI epic converges the `scripts/*.sh` tooling into one `dot` Go binary, one subcommand per twin port. CLI-002 landed the layout; this is the **first real port**. `scripts/init-spec.sh` is the natural first mover for two reasons: it is mechanical (id validation + work-gate + template copy), and it carries a live defect — it injects the `<!-- from issue #N -->` comment into `## Why` but **never sets the frontmatter `issue:` field**, so every scaffolded spec starts with `issue: ""` and has to be hand-fixed (it was, in this very spec, and in CLI-002 before it). Porting to Go lets us fix that on contact and gives the scaffold a self-contained binary that no longer needs the private vault checked out at the path the shell assumes.

## What

A new `dot spec init <feature-id> --issue <N>` subcommand that reproduces the observable behaviour of `init-spec.sh` and corrects the frontmatter bug:

- Validates `<feature-id>` against the same grammar (`AREA-NNN[letter][-slug]` or `YYYY-MM-DD-slug`).
- Enforces the ADR-018 work-gate: the issue must exist and be **OPEN** (verified by shelling out to `gh issue view`), unless `--force-no-gate` is given. Missing/closed/nonexistent issue → non-zero exit, no spec dir created, message naming the gate.
- Scaffolds `specs/<feature-id>/{proposal,tasks,verification}.md` from templates **embedded in the binary** (`//go:embed`), so it works on any machine with or without the vault.
- Injects `<!-- from issue #N: <title> -->` into `## Why`, **and** sets frontmatter `issue: "dotfiles#<N>"` (the fix).
- Refuses to clobber an existing `specs/<feature-id>/`; warns (does not fail) if the id exists under `specs/archive/`.

The shell `init-spec.sh` stays in place and unchanged; `dot spec init` is added alongside it.

## Out of scope

- Retiring `init-spec.sh` or collapsing bats↔`go test` for migrated logic — that is **CLI-005 #339**.
- Per-agent thin adapters that call `dot` instead of the shell — **CLI-006 #340**.
- Porting `archive-spec.sh` to `dot spec archive` — separate ticket on contact.
- The SELF-002 #249 shell-side repo-local fallback — this spec solves the vault-absent problem only for the **Go** path (by embedding), it does not touch `init-spec.sh`.
- Discovering the issue number from the feature-id — explicit `--issue` remains the contract.

## Risks / open questions

- **Drift guard vs. private vault (resolved decision, noted here).** The embedded templates duplicate the vault SSOT (`$VAULT_PATH/00_meta/templates/spec-*.md`). The dotfiles CI has **zero visibility into the private vault** (ADR-013), so a guard that diffs embedded-vs-vault can only assert on a machine where the vault is present. Decision: a Go test compares each embedded template to its vault file and **skips cleanly when `$VAULT_PATH` is absent** (caught locally + on dev machines where edits happen; a no-op in CI by design). The skip must not be failure-shaped — write it with `t.Skip`, the Go equivalent of the bats teardown trap fixed in #357.
- **`gh` availability/auth** (offline, fresh machines, CI): the gate fails closed with a clear message; `--force-no-gate` is the documented escape hatch. Tests stub `gh` on `PATH` — no network in the Go tests.
- **Template SSOT divergence over time.** If the vault template format changes, the embedded copy goes stale until someone re-vendors it. The drift guard surfaces this locally; re-vendoring is a manual `cp` step (a future `dot spec sync-templates` or a `compile-harness` hook could automate it, out of scope here).
- **Date source.** `init-spec.sh` stamps `created:` with `date -u +%Y-%m-%d`. The Go twin uses `time.Now().UTC()`; tests inject a fixed clock to stay deterministic.

## Acceptance criteria

Observable outcomes. Each must be testable.

- [ ] `dot spec init ID --issue N` with an OPEN issue scaffolds the three files, injects the Why comment, and sets frontmatter `issue: "dotfiles#N"` (exit 0).
- [ ] Missing `--issue`, nonexistent issue, or closed issue → non-zero exit, **no** `specs/ID/` directory created, message names the issue gate.
- [ ] `--force-no-gate` scaffolds without invoking `gh` at all.
- [ ] Invalid feature-id (fails the grammar) → non-zero exit with the expected-format message; valid sub-id forms (`SDD-012b-guard`) and date forms accepted.
- [ ] Existing `specs/ID/` → non-zero exit, no clobber; id under `specs/archive/` → warning, still scaffolds.
- [ ] Templates are embedded: `dot spec init` produces valid files with `$VAULT_PATH` unset / vault absent.
- [ ] Drift guard test fails if an embedded template diverges from its vault SSOT, and **skips** (not fails) when the vault is absent.
- [ ] `gofmt -l`, `go vet ./...`, `go test ./...`, `go build ./...` all green.

## References

- GitHub issue: `dotfiles#358` (work-gate per ADR-018)
- Epic: ADR-020 (CLI); `docs/architecture.md` ("where does X live")
- Twin source: `scripts/init-spec.sh`
- Related ADRs: `docs/adr/adr-018-de-vault-task-placement.md` (work-gate), ADR-013 (CI has no vault)
- Related patterns: `00_meta/patterns/pattern-spec-driven-development.md`
- Related tickets: CLI-005 #339, CLI-006 #340, SELF-002 #249

<!-- archived 2026-06-14 — PR: https://github.com/mlorentedev/dotfiles/pull/359 -->
