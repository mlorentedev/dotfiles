---
tags: [spec, tasks, secrets, shell]
created: "2026-06-25"
---

# Tasks - CLI-024-secrets-retire-loadsecrets

> TDD order. Delivered as a sequence of atomic PRs against this one spec / #587.

## Setup

- [x] Branch(es) created from main
- [x] `proposal.md` complete; acceptance criteria testable

## PR-B1 — nan-* scripts → dotf secrets show  (#588, MERGED)

- [x] **RED**: `tests/nan-scripts-secrets.bats` asserts no load-secrets source + `dotf secrets show` fallback.
- [x] **GREEN**: `scripts/nan-{bench,debug,quality-bench}.sh` → `dotf secrets show nan-api-key`.
- [x] Verify: bats 7/7; shellcheck clean. (Carried the `feat(secrets):` that cut 0.19.0.)

## PR-B2 — deploy secrets/registry.yaml in setup  (#591, MERGED)

- [x] Fix the 0.19.0 regression: setup-{linux,windows} deploy `secrets/registry.yaml`
  to `$DOTFILES_DIR/secrets/` (deployed `dotf secrets` reads it, but it was never deployed).
- [x] bats: both setups deploy it (structural) + the verify-setup integration check.

## PR-B3 — retire the load-secrets eager-source from setup  (#593, THIS PR)

- [x] Linux: remove the eager `load-secrets` source; agy's `OPENROUTER_API_KEY`
  now via `dotf secrets show openrouter-api-key` after install_dotf.
- [x] Windows: move the secrets deploy block before the opencode/agy consumers
  (parity with linux's early deploy); remove the (after-the-fact, dead) eager dot-source.
- [x] Finding: opencode/pi self-resolve via `substitute_env_placeholders` (age-decrypt
  from env-mapping.conf), NOT the ambient env — so only agy needed migrating.
- [x] Verify: bash -n + shellcheck; PowerShell AST parse 0 errors; bats `#587` green;
  live smoke `dotf secrets show openrouter-api-key` (len 73).

## PR-C — delete the load-secrets twins  (next)

- [x] Consumer sweep done (no rc/profile sourcing; only `scripts/test.sh` + tests reference the twin).
- [ ] `git rm scripts/load-secrets.{sh,ps1}` + `tests/load-secrets.bats`.
- [ ] Drop setup chmod (`setup-linux`) + the `load-secrets.ps1` deploy block (`setup-windows`).
- [ ] Retire the `scripts/test.sh` "Secrets Loading" section (tests the twin) + the integration Dockerfile ref.
- [ ] Archive the fine-grained-PAT lesson in `docs/runbooks/guide-bitacora-setup.md`.

## Deferred — `env-mapping.conf` + the substitute twins

- [ ] `substitute_env_placeholders` (utils.sh) + `Substitute-EnvPlaceholders` (utils.ps1)
  age-decrypt `{env:VAR}` from `env-mapping.conf`. Migrate them to read
  `secrets/registry.yaml` (or converge to a `dotf secrets render` command, ADR-020)
  BEFORE deleting `env-mapping.conf`. Then close #587.

## Machine-readable features

See sibling `features.json`. f1 (B1) verifiable now; f2 (B2/B3 setup) and f3 (deletion) land with their PRs.
