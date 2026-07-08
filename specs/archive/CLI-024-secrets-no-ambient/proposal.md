---
id: "CLI-024-secrets-no-ambient"
type: spec
status: archived 
created: "2026-06-25"
issue: "dotfiles#493"
tags: [spec, proposal, secrets, shell, adr-028]
template_version: "1.0"
---

# CLI-024-secrets-no-ambient

> Phase 1b of ADR-028: retire the login-time ambient secrets export from the interactive shells, migrating the consumers that read keys from the environment to launch via `dotf secrets run` (Phase 1a, #579). **Stacked on `feat/secrets-run-jit` — must merge after #579 and after a `dotf` redeploy that ships `dotf secrets run`.**

## Why

`.bashrc`/`.zshrc`/`profile.ps1` source `load-secrets` at every login, exporting ~30 decrypted secrets into the ambient environment of every shell. That is the exposure ADR-028's "not always exposed" objective targets. Phase 1a added the on-demand primitive (`dotf secrets run`); this phase removes the ambient export and routes the consumers through it.

## What

- Remove the `source load-secrets.sh` line from `.bashrc` and `.zshrc`, and the `load-secrets.ps1` dot-source block from `profile.ps1`.
- Replace it with **launcher wrappers** for the AI CLIs that read their keys from the environment (audited: opencode `{env:NAN_API_KEY}`, pi `{env:NAN_API_KEY}`, agy `${OPENROUTER_API_KEY}` + its MCP children):
  - POSIX: `opencode() { dotf secrets run -- opencode "$@"; }` (likewise `pi`, `agy`), guarded by `command -v dotf`.
  - PowerShell: `function opencode { dotf secrets run -- opencode @args }` (likewise), guarded by `Get-Command dotf`.
- **Parity injection** (no `--only`): each wrapped tool sees the same secret set it saw from the ambient export, but now scoped to its own process — zero risk of omitting a key.
- **Recursion-safe**: `dotf` is a child process that resolves the real binary on PATH via `exec.Command`, blind to the shell function — so the wrapper never calls itself.
- Update `tests/powershell-profile.bats` (the two assertions that encoded the old sourcing parity) to assert the new contract.

## Out of scope (deferred follow-ups)

- **`setup-{linux,windows}` eager-load** of load-secrets (a provisioning-time, non-interactive context) is left intact — its consumers are setup-internal and it is not the daily interactive exposure. Migrating it (and any setup block that reads ambient keys) is a separate ticket.
- **Deleting `load-secrets.{sh,ps1}`** — the later full-convergence step of #493; kept for now (manual fallback + the setup eager-load still uses it).
- **`--only` scoping per tool** — a later optimisation once the registry (Phase 2) maps consumers to keys.

## Risks / open questions

- **R1 — hard dependency on #579 + a `dotf` redeploy.** The wrappers call `dotf secrets run`; until the new `dotf` is on PATH the wrappers (guarded by `command -v dotf`) still launch the tool, but keyless. *Mitigation:* merge after #579; the `command -v dotf` guard avoids breaking the shell when dotf is absent.
- **R2 — a consumer not in the audit loses its key.** *Mitigation:* parity injection (full set) covers any key the tool previously saw; the audit (opencode/pi/agy + their MCP children; root MCP servers consume no secrets; CI uses Actions secrets) found no other interactive ambient consumer.
- **R3 — `dotf secrets run` failing (no age key) now blocks the tool launch** (fail-loud) where the ambient export would have launched it keyless. *Mitigation:* acceptable and more honest; `dotf doctor` (Phase 0) surfaces a missing age key.

## Acceptance criteria

- [ ] **AC1** — `.bashrc`/`.zshrc` no longer `source load-secrets.sh`; `profile.ps1` no longer dot-sources `load-secrets.ps1`. *Verify:* grep + the updated bats.
- [ ] **AC2** — opencode/pi/agy are wrapped to launch via `dotf secrets run` in all three RC files, guarded by a dotf-presence check. *Verify:* bats + grep.
- [ ] **AC3** — the RC files parse cleanly and are ASCII (`bash -n`, pwsh parser, no non-ASCII in shell files). *Verify:* `bash -n`, PS parser, grep.
- [ ] **AC4** — `tests/powershell-profile.bats` passes with the new contract; no other test asserts the removed sourcing. *Verify:* `bats tests/powershell-profile.bats`.

## References

- Issue: dotfiles#493; ADR: adr-028 (Phase 1b)
- Builds on: #579 (Phase 1a `dotf secrets run`), #578 (Phase 0 provisioning + doctor)
- Audit: opencode `ai/opencode/opencode.jsonc`, pi `ai/pi/models.json`, agy `ai/agy/mcp_servers.json`; root `mcp-servers.json` (no secret env)
