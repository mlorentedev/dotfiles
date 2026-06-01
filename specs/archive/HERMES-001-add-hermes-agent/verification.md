---
tags: [spec, verification]
created: "2026-05-31"
---

# Verification - HERMES-001-add-hermes-agent

## Evidence

Map every acceptance criterion from `proposal.md` to concrete proof (commit hash, test name, or observed behavior). Filled during implementation.

- [x] AC1 (shellcheck + bash -n + zsh -n clean) -> Track A, PR #195 / `tests/hermes-setup.bats`
- [x] AC2 (idempotent, second run no-op) -> Track A, PR #195 / test `hermes-setup.bats: idempotent`
- [x] AC3 (fail fast on missing uv / token) -> Track A, PR #195 / test `hermes-setup.bats: fail fast`
- [x] AC4 (AGENTS.md thin pointer) -> Track A, PR #195 / `ai/hermes/AGENTS.md`
- [x] AC5 (local-deploy surface untouched) -> Track A, PR #195 (setup.sh never wired into setup-linux)
- [x] AC6 (vault SSOT consistent, validate.sh green) -> Track B, vault master `8b3c299`: `validate.sh` exits 0 ("Vault SSOT consistent"). Curated `80_agents/hermes-nan/` against the reconciled provisioning reality.

## Test status

- Test suite: `tests/hermes-setup.bats` -> 11 green (Track A, PR #195).
- Manual smoke test (remote): `setup.sh` run twice on the NaN box (Track A) — idempotent, second run a no-op.
- Vault-side: `HERMES_VAULT_PATH=<checkout> bash 80_agents/hermes-nan/scripts/validate.sh` -> exit 0, "Vault SSOT consistent"; 10 SSOT files pass with frontmatter, constitution names its write zone, 3 warnings are box-runtime advisories (expected off-box).
- Follow-ups (this PR, #196): registered `ai/hermes/AGENTS.md` in the root overlay matrix; re-homed the `pattern-loader` skill off the `/tmp/hermes-vault` hardcode (vault source fixed on knowledge master; harness record regenerated). Opencode command-count test bumped 18 -> 19 for the new skill.
- No regressions in existing test suite: yes (after the count bump).

## Decisions made during implementation

Brief log of non-obvious trade-offs or course corrections. Routine choices belong in commit messages.

- Original ticket assumptions revised during design (apt relaxed, `/tmp` -> persistent path, `_index.md` -> `00-context.md`) — full rationale table in `proposal.md`.
- Box probe (2026-05-31) findings that shaped the implementation: (a) Hermes product config is at `/hermes-home/config.yaml`, not `~/.hermes/`, and exposes a native `hermes mcp add` CLI -> register Hive via CLI, not YAML patch; (b) Hermes commits via git CLI (pre-commit probe FIRED) -> write-zone/secret/no-force guardrails implemented as local git hooks; (c) the vault remote URL had the token embedded (`x-access-token:***@`) -> setup.sh strips it and relies on the credential helper; (d) the existing `/tmp/hermes-vault` inline pull cron is migrated to the robust `vault-pull.sh` wrapper.
- Open: `hermes mcp add` env-passing flag for `HIVE_VAULT_PATH` (non-blocking).

## Promotion candidates

Before archiving, flag what (if anything) should be promoted to the vault.

- [x] Lesson for `dotfiles/90-lessons.md`? Likely yes — "a remote agent's self-authored bootstrap docs drift; curate them against reality as part of onboarding it as code" (and the `/tmp`-as-durable-bridge anti-pattern).
- [x] ADR-worthy decision? Likely yes — remote self-provisioning agent: full-clone read + instruction-enforced write boundary + secrets divergence from the age model. Candidate `docs/adr/` or `30-architecture/adr-XXX`.
- [x] New pattern candidate for `00_meta/patterns/`? Only if it recurs — "remote agent provisioning via curl + vault SSOT" could generalize beyond Hermes.

## Archive checklist

- [x] `proposal.md` frontmatter set to `status: archived`
- [x] Folder moved: `specs/HERMES-001-add-hermes-agent/` -> `specs/archive/HERMES-001-add-hermes-agent/`
- [x] Backlog entry in vault `11-tasks.md` ticked with PR link
- [x] Promotions above executed (if any)
