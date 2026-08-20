---
id: "IDEAS-006-ai-topical-discovery"
type: spec
status: abandoned
created: "2026-05-25"
tags: [spec, proposal, ideas-006, ai, setup, refactor, dotfiles-survey, tier-3]
template_version: "1.0"
review: waived
review_waived_reason: "Abandoned in owner triage 2026-08-20. Never started, and the spec declines itself: its own Why section records that research/dotfiles-survey.md rated this Tier-3 (filosofico) and recommended NOT pursuing the full refactor. Cold since 2026-05-26."
---

# IDEAS-006: Topical discovery for `ai/<agent>/`

> **Naming**: file lives at `<repo>/specs/IDEAS-006-ai-topical-discovery/proposal.md`.
> Origin: dotfiles-survey research, Tier-3 idea (#6) — holman's topical pattern, applied ONLY to `ai/`.

## Why

<!-- from research/dotfiles-survey.md §"Top 6 ideas a aplicar" #6: scoped topical discovery for ai/<agent>/, Tier-3 (filosófico, partial-only) -->

`ai/<agent>/` is already topical de facto: one folder per agent (`claude/`, `opencode/`, `aider/`, `agy/`, `nan/`, `gemini/`...). Adding a new agent today requires editing `setup-linux.sh` to wire its config deployment + any per-agent install step. holman's full topical pattern (`*.symlink` + `install.sh` glob-discovery for ALL top-level folders) is overkill here — the user's repo has SDD discipline, drift detector, and audit-005's 9-category classification that DON'T benefit from filename-suffix convention. But applying the pattern *narrowly* to `ai/<agent>/install.sh` would let any future agent self-register without touching `setup-linux.sh`.

**Important caveat from research**: the dotfiles-survey doc rated this Tier-3 ("filosófico") and recommended NOT pursuing the full holman refactor. This spec is the scoped-incremental version — only `ai/<agent>/install.sh` discovery, leaving everything else hardcoded. If the user reviews this spec and decides even the scoped version is over-engineering for the current `ai/` agent count (~6-8 agents, low churn), abandoning is a valid outcome.

## What

In `setup-linux.sh`, replace the hardcoded `ai/<agent>` deployment blocks with a discovery loop:

```bash
for installer in "$REPO_ROOT"/ai/*/install.sh; do
    [ -x "$installer" ] || continue
    agent_name=$(basename "$(dirname "$installer")")
    log_info "Installing AI agent: $agent_name"
    # Failure isolation: don't let one agent break setup
    bash "$installer" || log_warn "Agent $agent_name install failed (continuing)"
done
```

Per-agent convention (documented in `ai/README.md`):

- `ai/<agent>/install.sh` is OPTIONAL. If absent, agent has no install step (config-only via symlinks elsewhere).
- If present: must be idempotent (re-runnable), must exit 0 on success, must shellcheck clean.
- It MUST NOT exit-on-error in a way that aborts the parent setup-linux.sh (failure isolation pattern shown above).

Migration approach: **incremental**. The PR migrates AT LEAST 2 existing agents as proof-of-pattern (e.g., `claude` + `opencode`). Other agents stay in their hardcoded `setup-linux.sh` blocks until separately migrated. Drift detector + audit-005 classification capture which agents still need migrating.

## Out of scope

- **Topical pattern for `scripts/`, `home/`, top-level `.symlink` glob** — full holman pattern. Not applying it. The research doc explicitly recommended against this scope.
- **PowerShell mirror in `setup-windows.ps1`** — tracked as IDEAS-006b for the Windows VM session.
- **Removing or merging existing agent configs** — pure mechanism change; configs stay.
- **Migrating ALL existing agents in this PR** — incremental over multiple PRs (or further specs).
- **`ai/<agent>/uninstall.sh` discovery** — not yet needed; defer.
- **Order dependencies between agents** — current agents are independent; if a dependency emerges later, switch to an explicit manifest (out of scope here).

## Risks / open questions

- **R1 (BLOCKER)**: incremental migration mid-flight. After this PR, `setup-linux.sh` has BOTH the new `for installer in ai/*/install.sh` loop AND the remaining hardcoded blocks for un-migrated agents. If a hardcoded block AND a per-agent `install.sh` BOTH exist for the same agent, double-install happens. Mitigation: per-migrated-agent commits MUST delete the hardcoded block in the same diff. Verify via grep in CI: for each `ai/*/install.sh`, no matching `# Setup <agent>` block in setup-linux.sh.
- **R2 (BLOCKER for failure isolation)**: `set -e` in setup-linux.sh + bash `installer` invocation. If the installer exits nonzero, `set -e` aborts the parent unless we use `bash "$installer" || log_warn`. Confirm this works as intended; bats test the failure path.
- **R3**: order. Alphabetical iteration: `agy → aider → claude → gemini → nan → opencode`. If a future agent depends on `claude` being installed first, alphabetical fails. Defer to future-spec via explicit manifest if it ever happens. Document the order in `ai/README.md`.
- **R4**: drift detector interaction. Adding/removing files in `ai/<agent>/` changes what the drift detector sees on subsequent runs. Confirm baseline updates atomically with the migration commits.
- **R5 (open question, this might invalidate the spec)**: **is this worth doing?** Current agent count is ~6-8, churn is low (1-2 new agents per year). The maintenance cost of adding an `install.sh` to each agent folder + migrating existing logic might exceed the cost of occasionally editing setup-linux.sh. **Recommendation: validate via cost-benefit BEFORE starting implementation. Sample-implement ONE agent migration end-to-end (claude or opencode), measure LOC delta + the felt simplicity gain. If marginal, archive this spec as `abandoned` and reclaim the LOC budget for higher-ROI work.**

## Acceptance criteria

- [ ] `setup-linux.sh` contains a discovery loop over `ai/*/install.sh` with failure isolation (per R2).
- [ ] At least 2 existing agents migrated to `ai/<agent>/install.sh` files (matching pattern from `setup-linux.sh` removed in same commit).
- [ ] Bats test: dummy `ai/_dummy/install.sh` that touches `/tmp/ideas006-marker` → setup-linux.sh run → marker file exists.
- [ ] Bats test: dummy `ai/_failer/install.sh` that exits 1 → setup-linux.sh completes without aborting, log shows warning.
- [ ] Un-migrated agents continue to work via remaining hardcoded blocks (no regressions in current `setup-linux.sh` behavior for them).
- [ ] `ai/README.md` documents the convention (when to add `install.sh`, idempotency requirement, failure-isolation contract).
- [ ] CI grep-check: no agent has BOTH `ai/<name>/install.sh` AND a `# Setup <name>` block in setup-linux.sh (double-install guard).
- [ ] Shellcheck clean on `setup-linux.sh` + each new `install.sh`.

## Completeness review

Standard items considered:

- **Rate limit / cost guard** — N/A.
- **Idempotency** — each per-agent `install.sh` MUST be idempotent (convention enforced in `ai/README.md`).
- **Regression test** — covered via "un-migrated agents continue to work" criterion + existing setup-linux integration tests.
- **Cert provisioning** — N/A.
- **Rollback** — single-commit revert restores hardcoded blocks. Cleaner: each agent migration in its own commit so individual rollback is granular.

Adding (not in template, load-bearing here):

- **Validate-before-implement gate** (per R5): the FIRST task in tasks.md is "sample-implement migration of one agent and measure". If the measured win is unclear, the spec is archived as `abandoned` BEFORE committing any code.
- **Cross-OS gap**: this is Linux-only. IDEAS-006b tracks the PowerShell mirror.

## References

- Research source: `research/dotfiles-survey.md` § "Top 6 ideas a aplicar" #6 (Tier 3, partial-only).
- Upstream: holman/dotfiles `script/bootstrap` + `script/install` glob-discovery pattern.
- Related: audit-005 (scripts classification) — `ai/` is the most topical-shaped subtree already; this spec formalizes it.
- Related: ADR-009 (multi-agent runtime), ADR-010 (agent harness parity) — context for why `ai/<agent>/` matters.
- Conditional dependency on: SDD discipline (spec-gate CI). This spec is itself ≥50 LOC → full SDD applies.

## LOC estimate

~30 LOC discovery loop + ~50 LOC per migrated agent's install.sh × 2 = ~100 LOC + ~80 LOC bats + ~30 LOC ai/README.md = **~210 LOC total**. Above the 50-LOC threshold; full SDD discipline applies.

**Or, if abandoned per R5 validation**: ~0 LOC, archived as `_abandoned/IDEAS-006-...` with the validation finding documented in verification.md.
