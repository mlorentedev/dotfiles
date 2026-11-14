---
id: "HARNESS-196-vendor-security-audit-web-perf"
type: spec
status: implementing # draft | implementing | verifying | archived
created: "2026-11-13"
issue: "mlorentedev/dotfiles#3456"   # repo#NNN — GitHub issue / Project item that tracks this spec
tags: [spec, proposal]
template_version: "1.0"
owner: manu
---

# HARNESS-196-vendor-security-audit-web-perf

> **Naming**: file lives at `<repo>/specs/HARNESS-196-vendor-security-audit-web-perf/proposal.md`. `HARNESS-196-vendor-security-audit-web-perf` is `AREA-NNN-slug` (e.g. `TOOL-001-secret-drift`).

## Why

<!-- from issue #3456: harness: committed records drifted from the vault (7 stale, security-audit and web-perf missing) -->

The vault adopted two skills on 2026-11-07, `security-audit` and `web-perf` (vault commit `6fbe06bc`), and dotfiles vendors the rendered copies of every vault skill under `harness/skills/`. Main never received them, so the committed harness lagged the vault and every unrelated `dotf harness refresh` dragged the drift in (#3456). Adopting them is more than a re-render: `security-audit` ships three third-party files (two validators and a JSON schema, all MIT, from Cloudflare) that land in the repository, and both skills need an attribution row and, for `security-audit`, a dependency entry. A change that brings third-party code into the repository is a decision, so it carries a spec.

## What

After this change:

- `harness/skills/security-audit/` (SKILL.md plus `scripts/{validate-findings.cjs,validate-coverage-ledger.cjs,report-schema.json}`) and `harness/skills/web-perf/SKILL.md` exist on main, rendered from the vault.
- The six records that had drifted (builder and reviewer agents; the adversarial-review, agent-lifecycle, handoff and using-git-worktrees skills) are re-rendered from the same vault commit. The reviewer persona now lists `security-audit` with `enforce: warn`, so `dotf harness gate` names it on stderr when a reviewer has not invoked it, and lets the call through.
- `DefaultSkillDependencies` carries `security-audit: [test]`, equal to the record's `requires:`.
- `harness/skills/ATTRIBUTION.md` names both skills, their upstream repositories and licenses: `security-audit` MIT (cloudflare/security-audit-skill), `web-perf` Apache-2.0 (cloudflare/skills). Both licenses were read from the upstream repositories' LICENSE files on 2026-11-13. The vault's web-perf record had said MIT; the vault was corrected first (vault commit `11bacfaf`, #3470) and re-rendered here.

## Out of scope

- A scheduled check or CI job that catches vault-to-repo drift. #3456 names it as the follow-up decision; it gets its own ticket and spec.
- Vendoring the upstream `*.test.cjs` files and the 13 reference documents of `security-audit`. The vault adopted `SKILL.md` and the three runtime files only.
- Running the validators anywhere in CI or in a hook. They are invoked only by an agent following the skill.
- The vault's own content (SKILL.md prose of either skill). Edits there go through the vault.

## Risks / open questions

- **Supply chain: third-party code in the repository.** The three files under `security-audit/scripts/` are byte-identical to upstream `cloudflare/security-audit-skill` at commit `c1c8a8c1471069fb0e188eeaff69b8e8db6564a8` (2026-09-14), checked with `cmp`. The record does not pin that commit anywhere; the pin lives in this spec and in `verification.md`. A later upstream change reaches the repository only through a vault edit and a refresh, each a reviewed PR.
- **What the scripts do.** Read in full for this spec: they use only `fs`, `path` and `util` (Node built-ins). Neither imports `child_process`, nor calls `eval`/`Function`, `fetch`/`http`, `process.env`, or any write, delete or rename API. Each reads one JSON file named by its single argument (and `validate-findings.cjs` also reads the sibling `report-schema.json`), refuses symlinks, non-regular files and inputs over 5 MiB, parses with nesting, array and error-count limits, prints PASS or FAIL lines on stdout/stderr, and exits 0 or 1. They write nothing.
- **Deploy path.** No Go code or shell script in this repository names or runs them (`git grep` for both file names outside the skill's own directory returns nothing). The deployer copies skill directories as files; the committed mode is 0644, so they are not executable. The only invocation is the one SKILL.md instructs an agent to make, `node scripts/validate-*.cjs <file>`, which needs Node (already a doctor-checked tool).
- **The reviewer persona gains a skill.** `security-audit` is declared `enforce: warn` in the reviewer record, so every reviewer dispatch now sees one more `[gate]` line until it invokes the skill. Warn-only, by the vault's choice; this PR does not raise it.
- **Local vault checkouts lag.** `tests/roster-consistency-real.bats` reads the vault it resolves. A checkout that has not pulled vault commit `11bacfaf` (ROSTER.md lists `security-audit` for the reviewer) fails that test until it does; CI has no vault and skips it.

## Acceptance criteria

- [ ] AC1: Every committed harness record, the two new skills included, carries a `generated_sha` equal to its own body, and `dotf harness check` reports no drift against the manifest.
- [ ] AC2: The compiled dependency map equals the records' `requires:`, including `security-audit: [test]`.
- [ ] AC3: Every vendored skill, `security-audit` and `web-perf` included, has an ATTRIBUTION.md row, and each of the two rows states the license the record declares.
- [ ] AC4: The three vendored `security-audit` scripts are byte-identical to upstream `cloudflare/security-audit-skill` at the commit pinned in this spec.
- [ ] AC5: The scripts' inspection (imports, side effects, deploy path) is recorded in `verification.md`, and no file of `harness/skills/security-audit/scripts/` is executable.

## References

- Bitácora board: dotfiles#3456 (this change), dotfiles#3470 (vault ROSTER and web-perf licence, fixed first)
- Vault commits: `6fbe06bc` (adopts both skills), `11bacfaf` (roster and licence fix)
- Upstream: https://github.com/cloudflare/security-audit-skill (MIT), https://github.com/cloudflare/skills (Apache-2.0)
- Related: `harness/skills/ATTRIBUTION.md`, `cli/internal/harness/triggers.go` (`DefaultSkillDependencies`)
