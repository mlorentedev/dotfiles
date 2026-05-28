---
tags: [spec, verification, sdd-009]
created: "2026-05-13"
updated: "2026-05-27"
---

# Verification - SDD-009-opencode-deploy-time-secrets

## Evidence

- [x] `utils.sh exports substitute_env_placeholders` -> `tests/sdd-009-deploy-time-secrets.bats:54` (`substitute_env_placeholders is defined in utils.sh`)
- [x] `utils.ps1 declares Substitute-EnvPlaceholders` -> `tests/sdd-009-deploy-time-secrets.bats:172` (parity assertion) + `tests/sdd-009-deploy-time-secrets.Tests.ps1` (runtime)
- [x] `setup-linux.sh calls helper on staged opencode.jsonc before deploy` -> `tests/opencode.bats:33` (`opencode deploy calls substitute_env_placeholders`)
- [x] `setup-windows.ps1 calls helper on staged opencode.jsonc before deploy` -> `tests/setup-windows.bats:334` (`opencode deploy calls Substitute-EnvPlaceholders`)
- [x] `Deployed file has zero {env: tokens for resolvable keys; unresolved left intact` -> bats tests 60-100 (substitute / leave-intact / warn / strict-when-all-resolved)
- [x] `Idempotent re-substitution on secret rotation` -> bats test 110-140
- [x] `Deployed file owner-only perms (chmod 600 equiv)` -> bats test 150-160
- [x] `Bats + Pester present` -> `tests/sdd-009-deploy-time-secrets.{bats,Tests.ps1}`
- [x] `Audit of {env:VAR} consumers in repo` -> PR body audit table (opencode.jsonc:40 NAN, opencode.jsonc:97 OLLAMA — only live consumers)
- [x] `load-secrets.{sh,ps1} unchanged` -> `git diff main...HEAD scripts/load-secrets.sh scripts/load-secrets.ps1` reports zero changes

## Test status

- Test suite: `~/.local/bin/bats tests/*.bats` -> 880/880 pass, 0 fail
- Manual smoke test (Linux): `setup-linux.sh` run on this machine deployed `~/.config/opencode/opencode.jsonc` with `NAN_API_KEY` literal value substituted and `{env:OLLAMA_API_KEY}` placeholder preserved (intentional - homelab secret not yet encrypted). One `[WARNING]` log line emitted naming OLLAMA_API_KEY as expected.
- Manual smoke test (Windows): **pending** — to be run by user on a Windows box. Pester suite at `tests/sdd-009-deploy-time-secrets.Tests.ps1` mirrors bats coverage. Tracked in vault for follow-up via WIN-004 CI runner.
- No regressions in existing test suite: yes — full bats run pre/post change both at 880 ok, 0 not ok.

## Decisions made during implementation

- **Unresolved-placeholder semantics (skip + warn)**: Helper substitutes only placeholders whose mapping is uncommented AND whose `.secret.age` file exists AND whose decryption succeeds. Other placeholders are left intact verbatim; one `log_warning` lists every unresolved NAME at end-of-run. opencode's runtime env resolver remains a fallback for the unresolved ones. Reason: `OLLAMA_API_KEY` ships with mapping commented (homelab secret not yet encrypted); fail-hard would break setup today.
- **AC5 relaxed accordingly**: original AC5 ("zero `{env:` tokens remaining") tightened to "zero `{env:` tokens for keys with a resolvable mapping". Original strict form holds in the all-resolvable case (asserted by bats test #5).
- **Idempotence pattern preserved via staged compare**: instead of `cmp -s "$SRC" "$DST"`, post-SDD-009 uses `cmp -s "$TMP" "$DST"` where TMP is the substituted artifact. That way the "already in sync" optimization survives substitution without false redeploys.
- **chmod 600 redundant on Linux** because `mktemp` creates at mode 600 and `mv` preserves it. Dropped the defensive `chmod 600 ... 2>/dev/null || true` from setup-linux.sh after `tests/opencode.bats:46` flagged it as a new silenced error.
- **PowerShell ACL owner derivation**: switched from `$env:USERDOMAIN\$env:USERNAME` to `WindowsIdentity::GetCurrent().Name` so the ACL hardening works under Microsoft / Azure AD accounts (`AzureAD\user@org` shape) where USERDOMAIN does not round-trip into a valid SDDL principal.
- **Existing test parity assertions updated, not deleted**: `tests/opencode.bats:25` and `tests/setup-windows.bats:324` were rewritten to match the new staged-source shape and a paired SDD-009-specific assertion was added next to each. Tests assert intent (call `substitute_env_placeholders`, then `cmp -s` against tmp), not specific line numbers, so they stay stable across future refactors.

## Promotion candidates

- [ ] Lesson for `10_projects/dotfiles/90-lessons.md`? **yes** — "Deploy-time substitution preserves idempotence when you compare the rendered artifact, not the raw source" (small but recurring pattern; applies to any future templated config).
- [ ] ADR-worthy decision for `30-architecture/`? **no** — no architectural shift; existing `adr-012-deploy-strategy-copy-with-drift-assertion` already covers the deploy-by-copy axiom this builds on.
- [ ] New pattern candidate for `00_meta/patterns/`? **no** — recurs only in this one config family today; promote if a second consumer materializes (e.g. agy or copilot ship runtime placeholder syntax).

## Archive checklist

- [ ] `proposal.md` frontmatter set to `status: archived`
- [ ] Folder moved: `specs/SDD-009-opencode-deploy-time-secrets/` -> `specs/archive/SDD-009-opencode-deploy-time-secrets/`
- [ ] Backlog entry in vault `11-tasks.md` ticked with PR link
- [ ] Promotions above executed (the one lesson candidate marked yes)
