---
id: "CLI-024-secrets-retire-loadsecrets"
type: spec
status: archived # draft | implementing | verifying | archived
created: "2026-06-25"
issue: "mlorentedev/dotfiles#587"   # repo#NNN — GitHub issue / Project item that tracks this spec
tags: [spec, proposal, secrets, shell]
template_version: "1.0"
---

# CLI-024-secrets-retire-loadsecrets

## Why

ADR-028 Phase 1 killed the ambient `load-secrets` shell export (#581) and shipped
the `secrets/registry.yaml` mapping SSOT with `dotf secrets run/ls/show` (#579,
#584). But the **non-shell** consumers of the old `load-secrets.{sh,ps1}` twins and
`sensitive/env-mapping.conf` are still live — three `nan-*` scripts and both
`setup-*` bootstrappers source them directly. Until those are migrated, the twins
and `env-mapping.conf` cannot be deleted, so the two-SSOT (registry vs
env-mapping.conf) and the "decrypt-at-startup" code path both persist. The original
tracker (#493) is closed and its design — a Linux superset + `eval "$(dotf secrets
env)"` that mutated the parent shell — was explicitly rejected by ADR-028 (inject
into the child only). This spec is the ADR-028-aligned retirement.

## What

After this feature no runtime path sources `load-secrets.{sh,ps1}` or reads
`env-mapping.conf`. Secrets are resolved on demand through the `dotf secrets`
facade:

- The three `nan-*` scripts fetch `NAN_API_KEY` via `dotf secrets show nan-api-key`
  (and stay invocable under `dotf secrets run -- <script>` with no double-fetch).
- `setup-{linux,windows}` resolve each mid-setup secret (`OPENROUTER_API_KEY`, …)
  via scoped `dotf secrets show <id>` instead of a blanket eager source.
- The `load-secrets.{sh,ps1}` twins and `sensitive/env-mapping.conf` are deleted,
  along with their tests and the setup chmod/deploy blocks.

Delivered as three atomic PRs against this one spec/issue:

| PR | Scope | Risk |
|----|-------|------|
| **B1** (this branch) | migrate the 3 `nan-*` scripts to `dotf secrets show` | low |
| **B2** | migrate the `setup-{linux,windows}` eager-load | high (critical path, cross-OS) |
| **C** | `git rm` the twins + `env-mapping.conf` + tests; archive the PAT lesson | low (gated on a clean consumer sweep) |

## Out of scope

- The Bitwarden (`bw serve`) backend and the age→bw migration — Phase 3, **#585**.
- Curation: Bitwarden folders, per-purpose token split (#321), offline age key —
  Phase 4, **#586**.
- How agents materialize MCP configs (placeholder `{env:…}` injection stays as-is).
- Secret rotation.

## Risks / open questions

- **B1 depends on a deployed `dotf secrets show`** (ships in 0.19.0). Until the
  redeploy, a direct `nan-*` invocation degrades gracefully: empty key → a clear
  error pointing at `dotf secrets run`. Resolved: graceful degradation, no hard dep.
- **B2 is critical-path, cross-OS.** A missed mid-setup consumer silently receives
  an empty value. Mitigation: enumerate every consumer of the eager-loaded vars
  before editing; the existing age-key preflight warning still fires.
- **PR-C deletion must be gated on a clean consumer sweep** (grep over scripts +
  setup + rc files shows zero live readers of the twins / `env-mapping.conf`).

## Acceptance criteria

- [ ] **AC1 (B1)** — `nan-bench.sh`, `nan-debug.sh`, `nan-quality-bench.sh` resolve
  `NAN_API_KEY` via `dotf secrets show nan-api-key`; none source `load-secrets` or
  call `secrets_refresh`. *Verify:* `bats tests/nan-scripts-secrets.bats`.
- [ ] **AC2 (B2)** — `setup-linux.sh` / `setup-windows.ps1` resolve mid-setup
  secrets via `dotf secrets show`; neither eager-sources `load-secrets`. *Verify:*
  `bats tests/setup-windows.bats` + `setup-linux` grep.
- [ ] **AC3 (C)** — `load-secrets.{sh,ps1}` + `sensitive/env-mapping.conf` deleted;
  a repo-wide grep finds no runtime reference. *Verify:* grep sweep is clean.
- [ ] **AC4** — all bats suites green cross-OS after each PR.
- [ ] **AC5 (C)** — fine-grained-PAT lesson archived in
  vault `00_meta/runbooks/bitacora-project-setup.md` (migrated from `docs/runbooks/guide-bitacora-setup.md`, 2026-07-07).

## References

- ADR: `docs/adr/adr-028-secrets-two-tier-bitwarden-age.md`
- Issue: `mlorentedev/dotfiles#587` (supersedes the closed #493)
- Registry: `secrets/registry.yaml` (id `nan-api-key` → `NAN_API_KEY`)
- Related patterns: `00_meta/patterns/secrets-security.md`, `shell-standards.md`
