---
id: "REFACTOR-013-version-minimum-enforcement"
type: spec
status: active
created: "2026-06-20"
issue: 479
tags: [spec, proposal, setup, versions, cross-os-parity, refactor]
template_version: "1.0"
---

# REFACTOR-013 — version pins as minimums (not presence, not exact)

> Extends [[REFACTOR-011]] (version-manifest). REFACTOR-011 made opencode
> *converge* to the pin; this spec generalizes the contract to **minimum**
> semantics across the pinned-install gates and removes the two ways the
> installers still violate it.

## Why

`versions.conf` pins are documented as **minimum** versions, but two install
patterns break that contract:

**Defect A — presence-only gates never upgrade.** A tool gated solely on
`command -v X` / `-x $BIN` is reported "already installed" even when the
installed copy is *older* than the pin, so it is never converged. This is the
user-reported symptom ("if it finds an old one it keeps it"). Worst case: `pi`
on Linux (`setup-linux.sh`), whose drift was only ever *warned* by healthcheck
(#474), never fixed.

**Defect B — exact-match reconcile downgrades.** The opencode/yarn/pi reconcile
blocks compare with `!=` (shell) / `-ne` (PowerShell). A version *newer* than
the pin reads as drift and is **downgraded** to the pin — the opposite of a
minimum.

Both directions contradict "pins are minimums". The fix is a single semver
comparator used by every gate: upgrade iff `installed < pin`, no-op iff
`installed >= pin`.

## What

A shared minimum-version comparator + rewired gates:

- `scripts/utils.sh`: new `version_gte "$installed" "$pin"` (pure `sort -V`;
  empty pin → satisfied, empty installed → needs install).
- `setup-windows.ps1`: new `Test-VersionAtLeast` helper (uses the native
  `[version]` type, already used for the hive `-ge` check; falls back to string
  equality for non-semver tags so it never throws).
- `setup-linux.sh`: `opencode` and `yarn` switch `!=` → `! version_gte`; `pi`
  gains a below-minimum upgrade branch (was presence-only).
- `setup-windows.ps1`: `opencode` (tools loop), `yarn`, and `pi` switch
  `-ne` → `-not (Test-VersionAtLeast …)`.

### Out of scope (deliberate)

- `bats`, `obsidian` on Linux install *latest*, not the pinned version — the
  pin is not wired into their install command, so they are a different (weaker)
  case. Converting them means choosing whether to pin the install too; tracked
  as a follow-up, not bundled here.
- `AGE_VERSION` / `EZA_VERSION` / `ZOXIDE_VERSION` are explicitly
  Windows-CI-only download pins (see the `versions.conf` comment); Linux
  installs these by other means and ignores the pins. Not minimum-enforced.

## Acceptance criteria

- **AC1** `version_gte` returns 0 for equal, 0 for newer, 1 for older; numeric
  not lexical (`1.10.0 >= 1.9.0`); empty-pin → 0; empty-installed → 1.
- **AC2** `Test-VersionAtLeast` mirrors AC1 and never throws on a non-semver tag.
- **AC3** No install gate compares a pinned tool's version with `!=` / `-ne`;
  the downgrade path is gone (regression-asserted).
- **AC4** `pi` on Linux upgrades when installed < pin (no longer presence-only).
- **AC5** A newer-than-pin install is left untouched on both OSes (no downgrade).
- **AC6** `setup-linux.sh` passes `bash -n`; `setup-windows.ps1` passes
  PSScriptAnalyzer (Error+Warning) and parses clean.
