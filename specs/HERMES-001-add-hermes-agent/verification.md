---
tags: [spec, verification]
created: "2026-05-31"
---

# Verification - HERMES-001-add-hermes-agent

## Evidence

Map every acceptance criterion from `proposal.md` to concrete proof (commit hash, test name, or observed behavior). Filled during implementation.

- [ ] AC1 (shellcheck + bash -n + zsh -n clean) -> commit `<hash>` / `features.json` f1
- [ ] AC2 (idempotent, second run no-op) -> commit `<hash>` / test `hermes-setup.bats: idempotent`
- [ ] AC3 (fail fast on missing uv / token) -> commit `<hash>` / test `hermes-setup.bats: fail fast`
- [ ] AC4 (AGENTS.md thin pointer) -> commit `<hash>` / `features.json` f4
- [ ] AC5 (local-deploy surface untouched) -> commit `<hash>` / `features.json` f5
- [ ] AC6 (vault SSOT consistent, validate.sh green) -> vault-side run of `validate.sh`

## Test status

- Test suite: `<command> -> <output / coverage %>`
- Manual smoke test (remote): `setup.sh` run twice on the NaN box — what was exercised, what was observed
- Vault-side: `bash 80_agents/hermes-nan/scripts/validate.sh -> <output>`
- No regressions in existing test suite: yes / no (if no, document)

## Decisions made during implementation

Brief log of non-obvious trade-offs or course corrections. Routine choices belong in commit messages.

- Original ticket assumptions revised during design (apt relaxed, `/tmp` -> persistent path, `_index.md` -> `00-context.md`) — full rationale table in `proposal.md`.
- Box probe (2026-05-31) findings that shaped the implementation: (a) Hermes product config is at `/hermes-home/config.yaml`, not `~/.hermes/`, and exposes a native `hermes mcp add` CLI -> register Hive via CLI, not YAML patch; (b) Hermes commits via git CLI (pre-commit probe FIRED) -> write-zone/secret/no-force guardrails implemented as local git hooks; (c) the vault remote URL had the token embedded (`x-access-token:***@`) -> setup.sh strips it and relies on the credential helper; (d) the existing `/tmp/hermes-vault` inline pull cron is migrated to the robust `vault-pull.sh` wrapper.
- Open: `hermes mcp add` env-passing flag for `HIVE_VAULT_PATH` (non-blocking).

## Promotion candidates

Before archiving, flag what (if anything) should be promoted to the vault.

- [ ] Lesson for `dotfiles/90-lessons.md`? Likely yes — "a remote agent's self-authored bootstrap docs drift; curate them against reality as part of onboarding it as code" (and the `/tmp`-as-durable-bridge anti-pattern).
- [ ] ADR-worthy decision? Likely yes — remote self-provisioning agent: full-clone read + instruction-enforced write boundary + secrets divergence from the age model. Candidate `docs/adr/` or `30-architecture/adr-XXX`.
- [ ] New pattern candidate for `00_meta/patterns/`? Only if it recurs — "remote agent provisioning via curl + vault SSOT" could generalize beyond Hermes.

## Archive checklist

- [ ] `proposal.md` frontmatter set to `status: archived`
- [ ] Folder moved: `specs/HERMES-001-add-hermes-agent/` -> `specs/archive/HERMES-001-add-hermes-agent/`
- [ ] Backlog entry in vault `11-tasks.md` ticked with PR link
- [ ] Promotions above executed (if any)
