---
id: "REFACTOR-011-version-manifest"
status: verifying # draft | implementing | verifying | archived
created: "2026-06-07"
issue: "mlorentedev/dotfiles#282"   # repo#NNN — GitHub issue / Project item that tracks this spec
tags: [spec, proposal, versions, ssot]
template_version: "1.0"
---

# REFACTOR-011-version-manifest

## Why

`versions.conf` is the central version manifest and **is** sourced by `.zshrc`/`.bashrc`
(`. "$DOTFILES_DIR/versions.conf"` one line above the tool-home exports). But a version
bump is **not** "one line": every tool version is *also* hard-coded as a `${VAR:-X.Y.Z}`
fallback in **both** RC files, so a bump must be repeated in three places or the fallback
silently drifts from the manifest (a stale fallback only surfaces if `versions.conf` ever
goes missing — i.e. it masks failure rather than catching it). Separately, **opencode is
not pinned at all** (installed via `curl https://opencode.ai/install | bash` = always
latest), so the daily-driver agent can differ across machines. REFACTOR-011 makes
`versions.conf` the **sole** version source (true one-line bumps), pins opencode, and adds
a guard so no version literal can re-appear outside the manifest.

## What

- `OPENCODE_VERSION` is added to `versions.conf`; opencode install (Linux + Windows) pins
  to it (`curl … | bash -s -- --version "$OPENCODE_VERSION"` — the official installer
  honors `--version`/`VERSION`, verified).
- The hard-coded `${VAR:-X.Y.Z}` fallback literals in `.zshrc`/`.bashrc` are removed;
  `versions.conf` (sourced immediately above) becomes the only source of each version.
- A bats cross-check fails CI if any RC file re-introduces a hard-coded version literal, or
  if a `*_HOME` references a version var that `versions.conf` does not define (incident→guard).
- `healthcheck.{sh,ps1}` report/assert the installed opencode version against `OPENCODE_VERSION`.

## Out of scope

- Auto-bump / Dependabot-style version-update automation.
- Migrating tools that are not currently path-pinned (node, gh, etc.) into the manifest — only `opencode` is added now.
- Changing the versioned-install directory scheme (`~/Applications/tool-X.Y.Z`).
- Fixing `init-spec.sh`'s legacy vault `11-tasks.md` rooting (ADR-018 moved tasks to the bitácora) — noted, separate ticket.

## Risks / open questions

- **Removing RC fallbacks**: if `versions.conf` is ever absent (broken checkout), `*_HOME`
  paths resolve empty and those tools drop off PATH. Accepted as non-fatal; the source line
  `[[ -f versions.conf ]] && . versions.conf` stays, and the new guard + healthcheck surface it.
- **opencode pin contract**: the pin relies on the upstream installer's `--version` flag; if
  upstream drops it the install silently falls back to latest. Mitigation: healthcheck asserts
  installed == pinned (turns a silent drift into a visible warning).
- **Windows parity**: `setup-windows.ps1` opencode install must accept the same pin — verify the Windows installer path honors a version argument.
- **CI Docker build-arg**: `BATS_VERSION` is passed as a Docker build arg in `ci.yml`; adding `OPENCODE_VERSION` must not perturb that path.

## Acceptance criteria

- [ ] `versions.conf` contains `OPENCODE_VERSION` (semver); `versions-conf.bats` asserts it is set + semver.
- [ ] `setup-linux.sh` and `setup-windows.ps1` install opencode pinned to `$OPENCODE_VERSION`.
- [ ] `.zshrc` and `.bashrc` contain no hard-coded version fallback literal; every tool home resolves solely from `versions.conf`.
- [ ] A bats test fails if a hard-coded version literal is re-introduced into either RC file (incident→guard, both shells).
- [ ] `healthcheck.{sh,ps1}` report/assert opencode version against `OPENCODE_VERSION`.
- [ ] Full `bats tests/*.bats` green; `shellcheck` clean on changed scripts.

## References

- GitHub: `mlorentedev/dotfiles#282`
- Vault backlog: REFACTOR-011
- Related: `docs/adr/dotfiles-architecture-map.md` ("where does X live" → `versions.conf`)
- Patterns: `00_meta/patterns/pattern-spec-driven-development.md`; incident→guard (`90-lessons`)
