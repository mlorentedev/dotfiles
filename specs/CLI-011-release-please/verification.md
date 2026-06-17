---
tags: [spec, verification, release, ci]
created: "2026-06-17"
---

# Verification - CLI-011-release-please

> Status: **implementing** — config/workflow/retirement landed and statically verified; the end-to-end release behaviour (AC4) is observable only after merge + the `RELEASE_PLEASE_TOKEN` secret exists.

## Evidence

- [x] AC1 (config + manifest valid, seeded 0.3.0) -> `jq` parses both; `.release-please-manifest.json` = `{".": "0.3.0"}`
- [x] AC2 (v<version> tags) -> `release-please-config.json` `include-component-in-tag: false`, single root package `.`
- [x] AC3 (versions.conf bumped) -> `extra-files: [versions.conf]` + `# x-release-please-start-version` / `# x-release-please-end` block around `DOTF_VERSION`
- [~] AC4 (tag triggers goreleaser) -> workflow wires `secrets.RELEASE_PLEASE_TOKEN` (static check passes); **end-to-end pending** the secret + first real release
- [x] AC5 (changelog-gen retired) -> `scripts/changelog-gen.sh` + `tests/changelog-gen.bats` removed; `grep -r changelog-gen` empty outside this spec; CHANGELOG.md header updated
- [x] AC6 (versions.conf single source, still parses) -> see test status

## Test status

- `bash -n versions.conf && zsh -n versions.conf` -> clean (block markers are comment lines)
- `bats tests/versions-conf.bats` -> green (the semver-pattern test skips the `#` marker lines; `DOTF_VERSION=0.3.0` stays unquoted/clean)
- `jq . release-please-config.json .release-please-manifest.json` -> valid JSON
- Full suite: unaffected by this change (only removes changelog-gen.bats and adds release-please config)
- **Not verifiable pre-merge:** the release PR creation, the `0.4.0` proposal, and the tag->goreleaser chain — release-please only acts on `push: main`. Record observations here after the first release.

## Decisions made during implementation

- **R1 token:** the default `GITHUB_TOKEN` will not trigger `cli.yml` on the tag it pushes (GitHub recursion guard) -> goreleaser would silently never run. Resolved with a dedicated PAT `RELEASE_PLEASE_TOKEN` (on-brand with OPS-007/#321). Hard dependency on the user provisioning the secret.
- **Marker placement:** block markers on their own comment lines, NOT inline — an inline `# x-release-please-version` on the `DOTF_VERSION=` line would make `versions-conf.bats`'s semver-pattern assertion read `0.3.0 # ...` and fail.
- **Seed at 0.3.0:** so the first release PR proposes `0.4.0`; required #412 (the last manual bump) to merge first so main's `versions.conf` and the manifest agree.
- **`release-type: simple`:** the version has no language-manifest home (the binary stamps it from the git tag via goreleaser ldflags); `simple` + `extra-files` + the manifest is the right fit. Open watch: confirm `simple` does not insist on a `version.txt` on the first run; if it does, it is a release-please-maintained mirror, not a second source of truth.

## Promotion candidates

- [ ] Lesson for `docs/lessons.md`: "a CI-pushed tag with the default GITHUB_TOKEN won't trigger downstream workflows — release automation needs a PAT" (generalizes beyond release-please).
- [ ] Cross-project pattern already exists (`pattern-release-please-ci`); update it with the goreleaser-composition + PAT-trigger note if novel.

## Archive checklist

- [ ] `proposal.md` frontmatter -> `status: archived`
- [ ] Folder moved to `specs/archive/CLI-011-release-please/`
- [ ] Issue #369 closed by the PR
- [ ] First release observed (AC4 evidence recorded above) before archiving
