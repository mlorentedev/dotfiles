---
tags: [spec, verification]
created: "2026-06-10"
---

# Verification - TERM-002-remove-ghostty

## Evidence

- [x] Zero ghostty refs outside archive/CHANGELOG -> `grep -rni ghostty . | grep -v "specs/archive\|CHANGELOG"` empty
- [x] 12 sections both healthchecks -> bats "healthcheck.ps1 and .sh both have 12 sections" + renumbered per-section asserts
- [x] yarn pin + install + checks -> bats "yarn version match" tests (sh + ps1); versions.conf YARN_VERSION=1.22.22
- [x] Suite green -> full bats run, only pre-existing environmental failures (identical on main)

## Test status

- `bats tests/healthcheck.bats tests/healthcheck-ps1.bats tests/verify-setup.bats tests/tmux.bats tests/powershell-profile.bats tests/versions-conf.bats` -> green locally (verify-setup container asserts run in CI integration)
- shellcheck: no new findings (pre-existing info/style only)

## Decisions made during implementation

- Healthcheck renumber done in lockstep with the parity bats (sections are assertion-coupled cross-OS).
- Historical records keep ghostty mentions (CHANGELOG, archived specs, ADR/audit snapshots, lessons) — rewriting history is the anti-pattern; living docs are clean.
- DX-003 abandoned (subject removed); the `--pure` oc workaround stays, comments repointed to the abandoned spec id.
- yarn via npm-global classic pin (not corepack): uniform cross-OS, independent of Node version bundling.

## Promotion candidates

- [ ] Lesson? no
- [ ] ADR? no
- [ ] Pattern? no

## Archive checklist

- [ ] proposal.md -> status: archived
- [ ] Folder -> specs/archive/
- [ ] Issue #281 closed by PR
