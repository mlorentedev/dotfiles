---
tags: [spec, verification]
created: "2026-06-07"
verified: "2026-06-08"
---

# Verification - REFACTOR-011-version-manifest

Implemented and verified on branch `refactor-011-version-manifest` (worktree off
`origin/main`). Windows parity was authored in the same change (mechanical mirror of the
Linux side); the single runtime check that needs a Windows box is flagged below.

## Evidence per acceptance criterion

- [x] **OPENCODE_VERSION in manifest** -- `versions.conf` adds `OPENCODE_VERSION=1.16.2`
      (matches the installed `opencode --version` = 1.16.2). bats `versions.conf sets
      OPENCODE_VERSION` passes; the existing "all values match semver" test covers format.
- [x] **opencode install pinned (Linux)** -- `setup-linux.sh` now sources the manifest
      (`. "$CURRENT_DIR/versions.conf"` near the top) and installs with
      `bash -s -- --version "$OPENCODE_VERSION"`, falling back to latest only when the var is
      empty (broken checkout). Ghostty fallback literal dropped (`${GHOSTTY_VERSION}`).
- [x] **opencode install pinned (Windows)** -- `setup-windows.ps1` parses `OPENCODE_VERSION`
      from `versions.conf` and passes `--version` to `winget install SST.opencode` via array
      splatting (no orphan `--version` when unset). Authored; runtime check pending a Windows
      box -- see Windows-empirical note.
- [x] **No hardcoded fallbacks in RC** -- `.zshrc`/`.bashrc` `${VAR:-X.Y.Z}` -> `${VAR}` for
      JAVA/MAVEN/PYTHON/MINIKUBE/GO (10 lines). `grep -E '\$\{[A-Z_]+_VERSION:-[0-9]'` on both
      returns nothing.
- [x] **Guard test (incident->guard)** -- new `tests/versions-no-hardcode.bats`: 4 tests
      (no-fallback-literal for each RC; every `*_VERSION` referenced in each RC is defined in
      the manifest). All green.
- [x] **healthcheck asserts opencode version** -- `scripts/healthcheck.sh` compares
      `opencode --version` to `OPENCODE_VERSION` (pass/fail/skip mirroring ghostty).
      `scripts/healthcheck.ps1` mirrors it via `$script:Versions['OPENCODE_VERSION']`.
- [x] **No regressions** -- see test runs below.

## Test runs (worktree, 2026-06-08)

- `shellcheck --severity=error setup-linux.sh scripts/healthcheck.sh` -> clean. CI threshold
  is `severity: error` for both the `scripts/` and root-script steps; these changes are clean
  even at `--severity=warning`.
- `bash -n` / `zsh -n` on `setup-linux.sh scripts/healthcheck.sh .zshrc .bashrc versions.conf`
  -> all OK.
- `bats tests/versions-conf.bats tests/versions-no-hardcode.bats` -> 17/17 pass.
- `bats tests/*.bats` (full suite) -> all version + deploy tests pass. 9 failures are
  pre-existing and environmental, reproduced identically on pristine `main` (0 introduced by
  this change):
    - 6x `tests/claude-mem-heal-ps1.bats` (PowerShell repair fns) -- pwsh not installed.
    - 3x `tests/shell-profile.bats` (4-6) -- `shell-profile.sh` exits non-zero in this sandbox.
- `.ps1` changes are ASCII-only: `grep -P '[^\x00-\x7F]'` flags only a pre-existing em-dash at
  `setup-windows.ps1` (~line 1035 on main), untouched by this change.

## Windows-empirical (one runtime check, not Linux-verifiable)

The Windows code is complete. The only thing not verifiable on Linux is whether
`winget install --version 1.16.2 SST.opencode` succeeds (i.e. the version is published in the
winget manifest) and that `opencode --version` on Windows prints a parseable trailing token.
Run `setup-windows.ps1` + `scripts/healthcheck.ps1` on a Windows box to confirm.

## Decisions made

- Remove RC fallbacks rather than keep-and-cross-check (user, 2026-06-06).
- opencode pin via installer `--version` (Linux) / winget `--version` (Windows).
- Windows parity authored now (Linux-authorable) rather than deferred -- avoids accumulated
  debt; only the winget runtime check remains environment-bound.

## Follow-ups (tracked, NOT in this PR)

- Renovate/Dependabot watching `versions.conf` -> auto-PR on new tool releases (the
  "self-manages" goal; explicitly out of scope here). Source stays the git repo, never the
  vault. Open as a new spec.
- Self-deploy timer (systemd/cron) over `scripts/dotfiles-sync.sh` (git pull + idempotent
  setup re-run).
- Pre-existing: `tests/shell-profile.bats` 4-6 fail in this sandbox -- investigate or gate.
- Pre-existing: non-ASCII em-dash at `setup-windows.ps1` ~line 1035 (latent PSScriptAnalyzer risk).
- pi code agent integration (separate spec; reuse `{env:NAN_API_KEY}` placeholder, rotate the
  plaintext key currently in `ai/pi/models.json`).

## Promotion candidates

- Lesson for `docs/lessons.md`: "a sourced manifest plus a hardcoded fallback are two sources
  of truth; the fallback masks failure instead of catching it." Promote at archive.
- ADR-worthy? No. New cross-project pattern? No (single-repo concern).

## Archive checklist

- [ ] `proposal.md` frontmatter `status: archived`
- [ ] Folder moved to `specs/archive/REFACTOR-011-version-manifest/`
- [ ] Backlog entry (#282) ticked with PR link
- [ ] Promotions executed (lesson)
