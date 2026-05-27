---
id: "DOCS-001-macos-claim-reconcile"
type: spec
status: draft # draft | implementing | verifying | archived
created: "2026-05-27"
tags: [spec, proposal]
template_version: "1.0"
---

# DOCS-001-macos-claim-reconcile

> **Naming**: file lives at `<repo>/specs/DOCS-001-macos-claim-reconcile/proposal.md`. `DOCS-001-macos-claim-reconcile` is `YYYY-MM-DD-<slug>` or `<TICKET-NN>`.

## Why

<!-- from 11-tasks.md: README claims Linux+macOS+Windows but no `setup-macos.sh`, no Brewfile, no macOS CI runner, no `.macos` defaults file. Decision-only ticket: drop the claim everywhere, or spawn a separate L spec to add real macOS support. Effort: S. Anti-scope: this ticket is the decision; if "add" wins, that spawns its own SDD spec. -->

`README.md:3` says "Works across Linux, macOS, and Windows" but the rest of the README only shows Linux+Windows install snippets. There is no `setup-macos.sh`, no Brewfile, no macOS-specific code path, no `macos-latest` CI runner, no `.macos` defaults file. AGENTS.md, CLAUDE.md, and several `ai/<agent>/*.md` files reference macOS as if supported (path resolution, SSH conventions). The claim drifts expectations the repo cannot meet — and for users this manifests as a broken install. Pick one stance and align everywhere.

## What

This is a **decision ticket**, not an implementation ticket. The PR records the chosen stance (in the PR body and in the vault `11-tasks.md`) and implements that choice:

- **(a) Drop**: edit `README.md`, `AGENTS.md`, `ai/claude/CLAUDE.md`, `ai/copilot/copilot-instructions.md`, `ai/agy/AGY.md` to remove "macOS" / "Linux/macOS" claims, replacing with "Linux + Windows".
- **(b) Add**: this spec closes with a link to a brand-new `SDD-XXX-macos-support` spec covering the full L-effort: `setup-macos.sh`, Brewfile, `.macos` defaults script, macOS-latest CI runner.

The PR does NOT branch — it commits to one path.

## Out of scope

- **Actually adding macOS support** (option b's work) — that's a separate L spec downstream.
- **Auditing other OS-support claims** (e.g., "WSL2" mentions) — surgical to the macOS claim only.
- **Translating Linux scripts to be macOS-compatible** without setup-macos.sh — partial support is worse than no support.

## Risks / open questions

- **R1**: Some Linux scripts use `case "$(uname -s)" in Darwin)` branches. If "drop" wins, those branches become dead code — delete them now or leave them harmless? Recommendation: leave (zero harm), let next REFACTOR remove dead.
- **R2**: `~/Projects/knowledge` path resolution in CLAUDE.md says "(Linux/macOS) or (Windows)" — purely a docs string, harmless if macOS is dropped (just becomes "Linux" + "Windows").
- **R3**: macOS user count is unknown. If the user actually runs macOS occasionally, dropping the claim is a self-foot-shot. Confirm before merging.

## Acceptance criteria

- [ ] PR body has a `## Decision` section that picks (a) or (b) with rationale.
- [ ] If (a): grep for `macOS` and `Mac OS` returns no hits in `README.md`, `AGENTS.md`, `ai/**/*.md` (excluding historical PR refs / ADR archive).
- [ ] If (b): this spec is closed with the URL of the new `SDD-XXX-macos-support` spec.
- [ ] Vault `11-tasks.md` DOCS-001 entry ticked with the decision.

## References

- Vault: `10_projects/dotfiles/11-tasks.md` § "Session 2026-05-27 — fresh-eyes audit" → DOCS-001.
- README claim site: `README.md:3,7,17,161`.
- AGENTS.md vault path resolution: `AGENTS.md:31, 228`.
