---
tags: [spec, verification]
created: "2026-06-07"
---

# Verification - REFACTOR-011-version-manifest

> Implementation deferred — this records the **planned** verification command per acceptance
> criterion so the next session has an executable target. Fill `Evidence` with commit hashes /
> test names once implemented.

## Planned verification per acceptance criterion

- [ ] **OPENCODE_VERSION in manifest** → `grep -qE '^OPENCODE_VERSION=[0-9]+\.[0-9]+\.[0-9]+$' versions.conf` + new bats `versions.conf sets OPENCODE_VERSION`.
- [ ] **opencode install pinned (Linux)** → `grep -qF 'bash -s -- --version "$OPENCODE_VERSION"' setup-linux.sh`; setup sources the manifest (`grep -qE '\. .*versions\.conf' setup-linux.sh`).
- [ ] **opencode install pinned (Windows)** → `setup-windows.ps1` passes `--version` for `SST.opencode` (Windows-empirical: confirm on a Windows box).
- [ ] **No hardcoded fallbacks in RC** → `! grep -qE '\$\{[A-Z_]+_VERSION:-[0-9]' .zshrc .bashrc`.
- [ ] **Guard test exists + fails on re-introduction** → `bats tests/versions-no-hardcode.bats` green; manually add a `:-1.2.3` to a temp RC fixture → test goes red.
- [ ] **healthcheck asserts opencode version** → `bats tests/healthcheck.bats` (+ `.ps1` mirror); `grep -q OPENCODE_VERSION scripts/healthcheck.sh`.
- [ ] **No regressions** → `bats tests/*.bats` green (baseline this session: 1063 pass; 3 pre-existing `shell-profile.bats` failures are sandbox-environmental, unrelated).

## Evidence

- [ ] (pending implementation)

## Decisions made

- **Remove RC fallbacks rather than keep-and-cross-check** (user decision 2026-06-06): the explicit goal is a one-line bump; the lost "missing versions.conf" safety net is non-fatal (tools drop off PATH) and is replaced by the guard test + healthcheck.
- **opencode pin is feasible** — official installer honors `--version`/`VERSION` (verified).
- **Windows pin deferred** — winget path is Windows-empirical (batch-windows rule).

## Promotion candidates

- [ ] Lesson for `docs/lessons.md`? Maybe — "a sourced manifest + a hardcoded fallback is two sources of truth; the fallback masks failure instead of catching it." Decide at archive.
- [ ] ADR-worthy? No.
- [ ] New pattern for `00_meta/patterns/`? No (single-repo concern).

## Archive checklist

- [ ] `proposal.md` frontmatter `status: archived`
- [ ] Folder moved to `specs/archive/REFACTOR-011-version-manifest/`
- [ ] Backlog entry ticked with PR link
- [ ] Promotions executed (if any)
