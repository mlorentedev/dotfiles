---
id: "SDD-009-opencode-deploy-time-secrets"
type: spec
status: draft # draft | implementing | verifying | archived
created: "2026-05-27"
tags: [spec, proposal]
template_version: "1.0"
---

# SDD-009-opencode-deploy-time-secrets

> **Naming**: file lives at `<repo>/specs/SDD-009-opencode-deploy-time-secrets/proposal.md`. `SDD-009-opencode-deploy-time-secrets` is `YYYY-MM-DD-<slug>` or `<TICKET-NN>`.

## Why

<!-- from 11-tasks.md: *(P1, queued 2026-05-26, cross-OS, SDD-required, spec: `specs/SDD-009-opencode-deploy-time-secrets/`)*: `ai/opencode/opencode.jsonc:40,97` uses opencode's runtime `{env:NAN_API_KEY}` / `{env:OLLAMA_API_KEY}` placeholder syntax. At runtime opencode resolves it from the launching shell's environment. On Linux this works because `~/.bashrc` sources `load-secrets.sh` and the value is in the parent shell. On Windows `load-secrets.ps1` sets vars in `Process` scope only — fragile when opencode launches from a different process or when the profile load order changed. **Proposal:** substitute `{env:VAR_NAME}` placeholders with the literal age-decrypted value at *deploy time* in both `setup-linux.sh` (line ~600) and `setup-windows.ps1` (line ~716). The deployed `~/.config/opencode/opencode.jsonc` becomes fully self-contained — no env-var propagation needed. **Tradeoff:** secret at rest in plaintext in the deployed config. **Mitigation:** that file is local-only, never committed, never synced; same trust model as `.git-credentials` and `~/.netrc`. **Done when:** (1) helper functions `substitute_env_placeholders` (sh) + `Substitute-EnvPlaceholders` (ps1) read `{env:NAME}` tokens, look up `sensitive/<key>.secret.age` via `env-mapping.conf`, decrypt with `age -d`, and inject the value; (2) both setup scripts call the helper before deploying opencode.jsonc; (3) bats + Pester tests with a fake age secret asserting placeholder is substituted byte-equivalently on both OSes; (4) audit any *other* file with `{env:VAR}` syntax (probably only opencode.jsonc — verify in proposal); (5) spec folder `specs/SDD-009-opencode-deploy-time-secrets/` with proposal + tasks + verification. **Cross-OS parity goal:** identical placeholder→value substitution on both setup scripts, tested. **Anti-scope:** do NOT remove `load-secrets.{sh,ps1}` — other tools (agy, claude, ssh) still need env-var injection. **TDD order:** bats/Pester failing test (placeholder present in deployed file) → helper implementation → green; repeat per OS. -->

`ai/opencode/opencode.jsonc:40,97` uses opencode's `{env:NAN_API_KEY}` / `{env:OLLAMA_API_KEY}` runtime placeholder syntax. opencode resolves these from the launching shell's environment. On Linux this works because `~/.bashrc` sources `load-secrets.sh` and the value propagates. On Windows `load-secrets.ps1` sets vars in `Process` scope only — fragile when opencode launches from a different parent process or when the profile load order shifts. Result: opencode silently 401s with no clear cause. The runtime-env propagation pattern is the bug class; deploy-time substitution makes the deployed config fully self-contained.

## What

Two new helper functions (`substitute_env_placeholders` in `utils.sh`, `Substitute-EnvPlaceholders` in `utils.ps1`) read `{env:NAME}` tokens from a file, look up `sensitive/<key>.secret.age` via `env-mapping.conf`, decrypt with `age -d`, and inject the literal value. Both setup scripts call the helper before deploying `opencode.jsonc`. The deployed `~/.config/opencode/opencode.jsonc` (Linux) and `%USERPROFILE%\.config\opencode\opencode.jsonc` (Windows) become self-contained — no env-var propagation needed at opencode launch.

## Out of scope

- **Removing `load-secrets.{sh,ps1}`** — other tools (agy, claude, ssh agent integration) still need env-var injection. The secrets loader stays.
- **Other files with `{env:VAR}` syntax** — audit confirms (or denies) opencode.jsonc is the only consumer. If others exist, surfaced in proposal; not folded in here.
- **Auto-detection of placeholder consumers** — explicit per-file substitution invocation; no scan-and-rewrite.
- **Re-encryption of deployed config** — plaintext at rest in `~/.config/opencode/` is accepted (same trust model as `.git-credentials`, `~/.netrc`).

## Risks / open questions

- **R1**: Plaintext secret at rest in deployed config. Mitigation: never committed (gitignore'd), never synced via dotfiles-sync, file mode 600 on Linux. Trust model documented.
- **R2**: Placeholder regex must distinguish `{env:VAR}` from genuine string content that happens to look similar (low risk in JSON/JSONC but worth a test).
- **R3**: Idempotency. Re-running setup must re-substitute fresh values (in case secret rotated). Helper rewrites every invocation, not just on file-missing.
- **R4**: Audit step. Find other `{env:VAR}` consumers in the repo. Likely opencode.jsonc only; proposal confirms via grep.
- **R5**: Cross-OS parity. The bats/Pester tests must verify byte-equivalent substitution given the same input + same `env-mapping.conf` entry.

## Acceptance criteria

- [ ] `utils.sh` exports `substitute_env_placeholders <file>` function.
- [ ] `utils.ps1` exports `Substitute-EnvPlaceholders -Path <file>` function.
- [ ] `setup-linux.sh` calls helper before deploying opencode.jsonc (line ~600).
- [ ] `setup-windows.ps1` calls helper before deploying opencode.jsonc (line ~716).
- [ ] Deployed `opencode.jsonc` has zero `{env:` tokens remaining (asserted by bats grep).
- [ ] Bats + Pester: fake age secret in fixture → asserts substituted value byte-equivalent on both OSes.
- [ ] Audit list in PR body: every other `{env:VAR}` consumer in the repo (or "opencode.jsonc only" if grep confirms).
- [ ] `load-secrets.{sh,ps1}` UNCHANGED.

## References

- Vault: `10_projects/dotfiles/11-tasks.md` → SDD-009 (GH #114).
- GH: <https://github.com/mlorentedev/dotfiles/issues/114>.
- Bug class precedent: BUG-006 (load-secrets cross-OS completeness).
- Related: `env-contract.json` (REFACTOR-002, now archived) defines the secret name space.
