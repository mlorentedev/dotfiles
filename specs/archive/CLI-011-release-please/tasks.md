---
tags: [spec, tasks, release, ci]
created: "2026-06-17"
---

# Tasks - CLI-011-release-please

> TDD order where testable; release automation is verified by config assertions + a post-merge end-to-end observation. One task = one focused commit.

## Setup

- [x] Branch created from main: `feat/release-please-adoption` (rebased on main @ DOTF_VERSION=0.3.0)
- [x] `proposal.md` complete; acceptance criteria testable
- [x] Open question R1 (token) resolved as a documented dependency: a `RELEASE_TOKEN` PAT secret (the default `GITHUB_TOKEN` would not trigger goreleaser)

## Implementation

- [x] Seed `.release-please-manifest.json` at `0.3.0` (the current published version) so the first release PR proposes `0.4.0`, not a duplicate `0.3.0`
- [x] `release-please-config.json`: `release-type: simple`, `include-component-in-tag: false` (→ `v*` tags goreleaser consumes), 0.x bump policy (`bump-minor-pre-major: true`, `bump-patch-for-minor-pre-major: false` → `feat`→minor, matching 0.1→0.2→0.3 history), `extra-files: [versions.conf]`
- [x] Annotate `versions.conf` `DOTF_VERSION` with `x-release-please-start-version` / `x-release-please-end` block markers on their **own lines** (keeps `versions-conf.bats` semver test green — an inline marker would fail it)
- [x] `.github/workflows/release-please.yml` on `push: main`, running `googleapis/release-please-action@v4` with `secrets.RELEASE_TOKEN` (R1)
- [x] Retire `scripts/changelog-gen.sh` + `tests/changelog-gen.bats` (release-please owns CHANGELOG.md)
- [x] Update `CHANGELOG.md` header to name release-please

## Blocked on the user / post-merge (cannot be done from this PR)

- [ ] **USER:** create a fine-grained PAT (`contents: write` + `pull-requests: write`) and add it as the repo Actions secret `RELEASE_TOKEN` (ties to OPS-007/#321). Until then the release PR + tag work but goreleaser does not auto-fire.
- [ ] Observe the first release PR after merge (proposes `0.4.0`; updates CHANGELOG.md + `DOTF_VERSION`)
- [ ] Observe merge → `v0.4.0` tag → `cli.yml` `release` job publishes binaries (AC4 end-to-end) → record in verification.md

## Closing

- [x] Every acceptance criterion mapped in `features.json` with an executable verification
- [x] `versions-conf.bats` green with the block markers; full bats suite unaffected
- [x] No `changelog-gen` references remain (outside this spec)
- [ ] PR opened referencing this spec folder + issue #369
