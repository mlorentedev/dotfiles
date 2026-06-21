---
id: "CLI-026-dotf-harness-engine"
type: spec
status: draft # draft | implementing | verifying | archived
created: "2026-06-21"
issue: "mlorentedev/dotfiles#495"   # repo#NNN — GitHub issue / Project item that tracks this spec
tags: [spec, proposal]
template_version: "1.0"
---

# CLI-026: dotf harness engine

> AUDIT-007 Phase B / PR10. Parent ADR: `docs/adr/audit-007-cli-convergence-state.md`. Supersedes the bash engine `scripts/compile-harness.sh` (spec: `specs/ENGINE-001-deploy-engine-core/`).

## Why

<!-- from issue #495: CLI-026: dotf harness — port compile-harness to harness {refresh,deploy,check} -->

The harness deploy engine exists twice: `scripts/compile-harness.sh` (Linux/macOS) and a hand-maintained twin inside `setup-windows.ps1` (the `Deploy-SkillRecord` block). They drift, and the bash half is fragile on Windows — proven on 2026-06-20: the winget `jq` build emits CRLF, shell word-splitting carried a stray `\r` into section slugs, and `compile-harness.sh --refresh` aborted on Windows. Because Windows `setup` only runs the deploy half, the committed skill records silently drifted from the vault SSOT for ~16 skills with nothing catching it (PR #511 is the interim bash fix). A single Go `dotf harness {refresh,deploy,check}` collapses the `.sh`/`.ps1` split into one cross-OS-by-construction binary — the AUDIT-007 strangler end-state (ADR-021) — so this whole class of platform-skew bug becomes impossible.

## What

A new `dotf harness` noun (Go, `cli/internal/harness/`, wired in `cli/internal/cmd/`, modeled on the existing `spec` noun which already ports shell scripts with embedded assets) with three subcommands that reproduce the current engine's contract:

- **`dotf harness refresh`** — vault patterns/skills → committed records (`harness/enforced/`, `harness/skills/`) and inject enforced blocks into each target's managed region. Requires the vault. Asserts per-file line caps. No `$HOME` write.
- **`dotf harness deploy`** — offline: render committed records to their per-agent `$HOME` paths and inject the copilot skill catalog. No vault. This replaces the `setup-windows.ps1` `Deploy-SkillRecord` block.
- **`dotf harness check`** — offline drift check: enforced regions diffed against `harness/enforced/`; skill records validated to render cleanly. No vault (CI / healthcheck).
- **`dotf harness check --against-vault`** (vault-present) — flags committed records that are **stale vs the vault SSOT** (`sha(vault skill) != committed record`) — the exact gap that silently drifted ~16 skill records before this spec (PR #511). This is the **upstream** half of the chain; it is a distinct axis from CLI-019/#488, which checks records ↔ deployed `~/.dotfiles` (the downstream half). Together they make `vault → records → deploy` fully observable.

End state (after PR-B): one engine, invoked identically on every OS; `setup-windows.ps1` calls `dotf harness deploy`; `scripts/compile-harness.sh` is deleted.

## Out of scope

- **Records ↔ deployed-dir drift** (repo records vs deployed `~/.dotfiles`/`$HOME`) — that is **CLI-019 / #488** (`dotf doctor`), the *downstream* half of the chain. This spec owns the *upstream* half only (vault ↔ records, via `check --against-vault`); the two are complementary, not duplicative.
- Changing the `harness/manifest.json` schema, the slugify rules, or the deployed `BEGIN/END HARNESS` block format — byte-for-byte parity is the goal, not a redesign.
- The vault SSOT content itself (patterns/skills).
- Porting any other AUDIT-007 noun (vault, secrets, mem, sync) — those are their own PRs.

## Risks / open questions

- **Behavioral parity is the whole game.** The Go engine must reproduce, exactly: the marker `sha256` (first 16 hex), the GitHub-style slugify, section extraction (body ends at the next same-or-higher-level heading), line-cap assertions, copilot catalog rendering, and skill-frontmatter validation. Any deviation makes `harness check` flag spurious drift on every machine. **Mitigation (MUST resolve before cutover):** a golden-file test that runs both engines against the current vault + records and diffs outputs byte-for-byte; delete `compile-harness.sh` only once that diff is empty.
- **Line endings → resolved: normalize in Go.** Go normalizes CRLF→LF on read for vault inputs (the vault has no `.gitattributes`, so its `.md` are CRLF on Windows) and emits LF, independent of platform — the structural fix for the bug PR #511 patched in bash. Chosen over adding a vault `.gitattributes`, which would externalize the fix to another repo and reintroduce the bug if the vault ever drops it.
- **Cutover → resolved: phased (PR-A → PR-B).** Ship the Go `dotf harness` alongside the bash engine first (**PR-A**: add `cli/internal/harness/` + golden-file parity test; no caller rewired, nothing deleted). Once the parity diff is proven empty, a follow-up (**PR-B**) rewires `setup-windows.ps1` to `dotf harness deploy`, removes the `Deploy-SkillRecord` twin, and deletes `compile-harness.sh`. Chosen over a single atomic PR to shrink blast radius. The cost is two engines co-existing between PR-A and PR-B; the parity test gates PR-B so bash is never trusted past proven divergence, and CLI-019/#488 + `check --against-vault` catch any interim drift.
- **Sequencing → resolved: hold for roadmap order (PR10).** Build after vault/spec-gate/secrets/mem land, not pulled forward. The interim bash fix (PR #511) already stops the active CRLF drift, so there is no urgency that justifies building on `cli/` conventions while CLI-019 PR-B and CLI-029 PR-B are still churning that surface. This spec is resolved and ready to implement when its slot arrives.

## Acceptance criteria

Observable outcomes. Each must be testable. Tagged by phase per the resolved cutover: **(PR-A)** ships Go alongside bash; **(PR-B)** removes bash and rewires callers.

- [ ] **(PR-A)** `dotf harness {refresh,deploy,check}` exist; each reproduces its `compile-harness.sh` counterpart's output **byte-for-byte** over the current vault + committed records (golden-file test).
- [ ] **(PR-A)** For identical inputs, output is byte-identical (LF, correct marker sha) on Windows and Linux — the winget-jq CRLF class of failure is structurally impossible (no shell word-splitting on tool output).
- [ ] **(PR-A)** `dotf harness check` runs offline (no vault) and is green in CI on a fresh checkout.
- [ ] **(PR-A)** `dotf harness check --against-vault` flags a record whose vault source changed but was never refreshed (regression test: mutate a vault skill, assert non-zero exit) — closing the silent-drift gap that motivated PR #511.
- [ ] **(PR-B)** `setup-windows.ps1` invokes `dotf harness deploy`; the `Deploy-SkillRecord` block is removed; a guard-grep for the old block is clean.
- [ ] **(PR-B)** `scripts/compile-harness.sh` is deleted; no remaining references in `setup-*.sh`/`setup-windows.ps1`/`ci.yml`/profile (guard-grep clean).

## References

- Parent ADR: `docs/adr/audit-007-cli-convergence-state.md` (CLI convergence roadmap; ADR-021 end-state)
- Superseded engine spec: `specs/ENGINE-001-deploy-engine-core/proposal.md`
- Interim fix this supersedes: PR #511 (CRLF-robust bash refresh + record reconciliation)
- Follow-up guard: `#488` / CLI-019 (`dotf doctor` record-vs-vault drift)
- Sister noun (porting model): `cli/internal/spec/` (ports `init-spec.sh`/`archive-spec.sh`, embedded templates)
- Pattern: `00_meta/patterns/pattern-spec-driven-development.md`
