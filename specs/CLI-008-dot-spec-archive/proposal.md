---
id: "CLI-008-dot-spec-archive"
type: spec
status: verifying # draft | implementing | verifying | archived
created: "2026-06-13"
issue: "dotfiles#361"   # repo#NNN — GitHub issue / Project item that tracks this spec
tags: [spec, proposal]
template_version: "1.0"
---

# CLI-008-dot-spec-archive

> **Naming**: file lives at `<repo>/specs/CLI-008-dot-spec-archive/proposal.md`. `CLI-008-dot-spec-archive` is `AREA-NNN-slug` (e.g. `TOOL-001-secret-drift`).

## Why

<!-- from issue #361: CLI-008: port archive-spec to `dot spec archive` (Go twin) -->

The ADR-020 CLI epic converges the `scripts/*.sh` tooling into one `dot` Go binary, one subcommand per twin port. CLI-007 landed `dot spec init` — the harder half of the spec lifecycle (embedded templates + `gh` work-gate + drift guard). This ports the other half: `scripts/archive-spec.sh` → `dot spec archive`. It is the simplest port in the epic so far (no templates, no gate, no network), which makes it the right place to exercise the established pattern — `internal/cmd/<verb>.go` wiring delegating to `internal/<domain>/` logic — on pure filesystem mechanics. It also lets the `dot spec` command own the **full** init→archive lifecycle in Go, so agents and CI no longer reach for the shell for either end.

## What

A new `dot spec archive <feature-id> [--pr <url>] [--abandoned] [--force-with-drafts]` subcommand that reproduces the observable behaviour of `archive-spec.sh`:

- Resolves the repo root (reusing `spec.RepoRoot`) and requires `specs/<feature-id>/` to exist (non-zero exit, clear message otherwise).
- **Pre-flight tag check:** scans the spec folder for unresolved `[AGENT-DRAFT]` / `[AGENT-SUGGESTION]` markers and refuses to archive while any remain, listing `file:line: text` for each — unless `--force-with-drafts` is given.
- Moves `specs/<id>/` → `specs/archive/<id>/`, or → `specs/archive/_abandoned/<id>/` under `--abandoned`. No-clobber: if the target already exists, it errors without moving.
- Rewrites `status:` in `proposal.md` frontmatter to `archived` (or `abandoned`), **scoped to the first frontmatter block only** — mirroring the `awk` state machine in the `.sh`, deliberately *not* the looser `(?m)^(status:)` regex in the `.ps1` which would also rewrite a `status:` appearing in the body.
- When `--pr <url>` is given, appends `<!-- archived <YYYY-MM-DD> — PR: <url> -->` to `proposal.md`.

The shell `archive-spec.{sh,ps1}` stay in place and unchanged; `dot spec archive` is added alongside them (same coexistence stance CLI-007 took for `init-spec`).

## Out of scope

- Retiring `archive-spec.{sh,ps1}` or collapsing bats/Pester ↔ `go test` for migrated logic — that is **CLI-005 #339**.
- Per-agent thin adapters that call `dot` instead of the shell — **CLI-006 #340**.
- Vault promotion (lessons/ADR/pattern) and any backlog/board tick — those stay **interactive** via `/spec archive` in an agent; this twin is mechanical only, matching the `.sh` which explicitly defers promotion.
- Un-archiving / reviving a spec — not a behaviour the `.sh` has.

## Risks / open questions

- **Exit-code fidelity (resolved decision).** Both shells use `exit 4` for the unresolved-tags case. No caller in the repo branches on it (grep confirms: no bats/CI dependency). To keep the Go twin family consistent — `dot spec init` returns a plain Cobra error (exit 1) for every failure — `dot spec archive` does the same: a descriptive error, exit 1, no bespoke exit-4 plumbing. Documented here so a reviewer can object; trivially reversible if fidelity is later required.
- **Frontmatter rewrite — combine the best of both shells (resolved via repo evidence).** The two twins diverge twice: the `.sh` (awk) **scopes** the substitution to the first `---`…`---` block but **replaces the whole line** (dropping any trailing comment); the `.ps1` (regex) **preserves the trailing comment** (replaces only the value token) but does **not** scope. Inspecting archived specs shows why both matter: `specs/archive/_abandoned/DX-002-dot-umbrella-command/` carries `status: abandoned # superseded by ADR-020 …` — a *meaningful* trailing comment the awk would have destroyed. So `dot spec archive` ports the **union**: scope to the first frontmatter block (from awk) **and** replace only the value after `status:`, preserving the rest of the line (from the regex). Covered by a dedicated test with (a) a decoy `status:` in the body and (b) a frontmatter `status:` carrying a trailing comment.
- **Cross-filesystem move.** `os.Rename` fails across filesystems; `specs/<id>` → `specs/archive/<id>` is always same-tree, so `os.Rename` is correct and atomic here (no need for a copy+remove fallback).
- **Feature-id validation (revised after review).** An earlier draft of this spec chose *not* to validate the id grammar in `archive` "for faithfulness to the `.sh`". A review (CodeRabbit on #362) flagged the real consequence: an unconstrained id flows straight into `filepath.Join`, so `../../foo` could move a directory outside `specs/`. The decision is reversed: `Archive` calls `ValidateID(id)` first. The grammar admits no `/`, `\`, or `..`, so validation *is* the traversal guard, and it makes the init/archive pair symmetric. In practice no real spec dir is excluded — `init` validates on creation, so every existing `specs/<id>` already conforms. Guarded by `TestArchiveRejectsTraversalID`.
- **Trailing-newline / byte fidelity.** The `.sh` appends the PR comment with a leading `\n`; the rewrite must not corrupt the file otherwise. Tests assert the surrounding content is preserved.

## Acceptance criteria

Observable outcomes. Each must be testable.

- [ ] `dot spec archive ID` moves `specs/ID/` → `specs/archive/ID/` and sets `status: archived` in `proposal.md` (exit 0).
- [ ] `--abandoned` routes to `specs/archive/_abandoned/ID/` and sets `status: abandoned`.
- [ ] Unresolved `[AGENT-DRAFT]` / `[AGENT-SUGGESTION]` tags block the archive (non-zero exit), the message lists each `file:line: text`, and **no move occurs**; `--force-with-drafts` overrides and archives.
- [ ] Missing `specs/ID/` → non-zero exit, clear message, no move.
- [ ] Target already present in archive → non-zero exit, no clobber, source left in place.
- [ ] `--pr <url>` appends `<!-- archived <date> — PR: <url> -->` to `proposal.md`.
- [ ] The `status:` rewrite changes only the first frontmatter block; a `status:` token elsewhere in the body is left untouched.
- [ ] `gofmt -l`, `go vet ./...`, `go test ./...`, `go build ./...` all green.

## References

- GitHub issue: `dotfiles#361` (work-gate per ADR-018)
- Epic: ADR-020 (CLI convergence); `docs/adr/dotfiles-architecture-map.md` ("where does X live")
- Twin source: `scripts/archive-spec.sh` (canonical); `scripts/archive-spec.ps1` (parity reference — note the regex-vs-awk divergence above)
- Predecessor: `specs/CLI-007-dot-spec-init/` (established the cmd/domain split + coexistence stance)
- Related tickets: CLI-005 #339 (retire shells), CLI-006 #340 (per-agent adapters)
- Related patterns: `00_meta/patterns/pattern-spec-driven-development.md`
