---
id: "CLI-012-dotf-doctor"
type: spec
status: draft # draft | implementing | verifying | archived
created: "2026-06-14"
issue: "dotfiles#376"   # repo#NNN — GitHub issue / Project item that tracks this spec
tags: [spec, proposal]
template_version: "1.0"
---

# CLI-012-dotf-doctor

> **Naming**: file lives at `<repo>/specs/CLI-012-dotf-doctor/proposal.md`. `CLI-012-dotf-doctor` is `AREA-NNN-slug`.

## Why

<!-- from issue #376: CLI-012: Port healthcheck + doctor to `dotf doctor` (retire the diagnostics twins) -->

Post-setup diagnostics are split across two overlapping shell twin pairs: `healthcheck.{sh,ps1}` (585/660 LOC, a 12-section verification of core tools, versioned paths, version match vs `versions.conf`, symlinks, env vars, optional tools, vault) and `doctor.{sh,ps1}` (218/233 LOC, verifies env vars / PATH / binaries against `env-contract.json` and can apply safe defaults + invoke heals). They overlap on env/PATH/binary checks yet are maintained as four files (~1,500 LOC of dual-maintenance) with their own bats/Pester. This is the **first port of ADR-021** (the CLI orchestration roadmap): consolidate them into a single cross-compiled `dotf doctor` so there is one diagnostics surface, one `go test` suite, and no per-OS twin.

## What

A new `dotf doctor` subcommand that reproduces the observable post-setup checks of `healthcheck` + `doctor` from one binary:

- Reads `versions.conf` (pinned versions) and `env-contract.json` (structural env/PATH/binary contract).
- Runs the consolidated sweep: core tools on PATH, versioned tool dirs exist, installed version matches the pin, key symlinks resolve, required env vars + PATH entries are present, optional tools reported, contract satisfied.
- Exit 0 when everything passes, non-zero with a per-failure report otherwise (same contract as `healthcheck.sh` exit 0/1).
- Preserves `doctor`'s optional **apply-safe-defaults** path behind a flag (e.g. `dotf doctor --fix`), keeping the env-contract heal capability.
- Per ADR-021 *port = consolidate, not translate*: the overlapping checks fold into one coherent surface, not two transliterated scripts.

After this PR, `dotf doctor` is the single post-setup diagnostics entry on every platform; `setup` invokes it; the shell twins are gone.

## Out of scope

- `diff-check.{sh,ps1}` (repo ↔ `~/.dotfiles` deploy drift) — a related diagnostic, but a separate concern; may become a `dotf doctor` mode in a follow-up, not this PR.
- `vault-health.sh` — routed to `dotf vault` (knowledge domain), not `dotf doctor`.
- Any change to `env-contract.json` / `versions.conf` schemas — consumed as-is.
- Windows `setup-windows.ps1` wiring of `dotf doctor` — independent (CLI-009 follow-up territory); Linux setup is repointed here.

## Risks / open questions

- **Behavioural parity surface is large.** healthcheck has 12 sections; the port must reproduce each observable check + the exit-code contract. Mitigation: enumerate the sections as a checklist in `tasks.md`; golden-test the report output where stable.
- **`env-contract.json` parsing.** `doctor.sh` shells out to `jq`; the Go port parses JSON natively (removes the `jq` runtime dependency — a small win). Confirm the contract schema is fully covered.
- **`--fix` / heal invocation.** `doctor.sh` can apply safe defaults and invoke known heals (e.g. claude-mem-heal). Decide which heals are in-scope for `dotf doctor --fix` vs left to their own commands. Lean conservative: report by default, fix only behind the explicit flag.
- **Vault section coupling.** healthcheck's "Knowledge Vault" section overlaps with the future `dotf vault`. Keep the read-only vault *presence* check here; defer deep vault health to `dotf vault`.
- **Twin deletion gating.** As with CLI-005, the twins are on PATH via `scripts/`; they can only be deleted once `dotf` is the installed on-PATH replacement (v0.2.0 shipped, install path fixed by #373).

## Acceptance criteria

Observable outcomes. Each must be testable.

- [ ] `dotf doctor` reproduces the healthcheck + doctor checks (core tools, versioned paths, version match, symlinks, env vars, optional tools, env-contract); exit 0 on all-pass, non-zero on any failure with a per-failure report.
- [ ] `dotf doctor --fix` applies the env-contract safe defaults that `doctor.sh` applied (no behavioural regression in the heal path).
- [ ] `scripts/{healthcheck,doctor}.{sh,ps1}` and their bats/Pester no longer exist in the tree.
- [ ] No live reference to the retired scripts remains outside provenance: `grep -rE 'healthcheck|doctor\.(sh|ps1)'` returns only CHANGELOG / ADRs / `specs/`.
- [ ] `setup-linux.sh` post-setup step invokes `dotf doctor`; `CLAUDE.md` / `AGENTS.md` / docs point to it.
- [ ] `go test ./...` covers the check logic; `dotf doctor` smoke-tested end-to-end on a real checkout.

## References

- GitHub issue: `dotfiles#376` (work-gate)
- Roadmap: `docs/adr/adr-021-cli-orchestration-roadmap.md` (this is its first port)
- Pattern: ADR-020 §5 strangler-fig + the CLI-005 port playbook (delete twins on contact, guard-grep as completeness oracle)
- Twin sources: `scripts/healthcheck.sh` (12-section), `scripts/doctor.sh` (env-contract); `.ps1` siblings as parity reference
