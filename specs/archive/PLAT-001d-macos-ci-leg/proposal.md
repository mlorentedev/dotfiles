---
id: "PLAT-001d-macos-ci-leg"
type: spec
status: archived # draft | implementing | verifying | archived
created: "2026-10-07"
issue: "mlorentedev/dotfiles#2013"   # repo#NNN — GitHub issue / Project item that tracks this spec
tags: [spec, proposal]
template_version: "1.0"
wip_override: "PLAT-001 track W10: the macOS CI leg is the evidence for every other PLAT-001 row, and the other stalled specs are outside this change's scope (19 active, limit 10, 2026-10-07)"
---

# PLAT-001d: the macOS CI leg, and the bats failures it exposed (track W10)

## Why

<!-- from issue #2013: PLAT-001: [EPIC] OS-agnostic from-zero bootstrap — macOS bring-up as the forcing function -->

Every CI job runs on Linux or Windows, so nothing could say that the suite is wrong on a Mac. Run there for the first time on 2026-10-07, 88 of 1784 bats tests failed, and four of the causes were silent: a lint that passed every file because `grep -P` errored into `/dev/null`, `wc` output padded with spaces, guards that compared a logical path with a physical one, and a stray-process detector that matched nothing. W10 of #2013 (#1843 E10, pulled forward) asks for a macOS leg; a leg that runs a suite known to be red on that OS, or one that selects nothing and passes, would be worse than none.

## What

Three layers, one common suite. There are never per-OS copies of a test.

1. **OS behaviour is tested by injection where it can be.** Go: doctor's `System.GOOS` seam and deploy's `goos` parameter run the darwin and windows branches on the one Linux runner. Nothing to build here; one gap found (see Risks).
2. **A real macOS runs the OS-sensitive tier.** Tests whose answer depends on the OS (BSD sed, stat, wc, symlinked tmp paths, hooks, shell startup, installers) carry the bats tag `os-sensitive`. A non-required `test-macos` job in `ci.yml` runs `bats --filter-tags os-sensitive` under `/bin/bash` 3.2, plus `go build`, `go vet` (native, so it is the darwin vet) and `go test ./...`, only when the existing path filter says code changed. On a push to main it also runs the full suite. The Linux job keeps the full suite. The same tag is usable by `test-windows` later.
3. **End to end from zero is X1**, not this change; it is gated on the root installers.

Every bats failure on this Mac is classified (script bug, non-hermetic test, Linux-only) and resolved, so the leg starts green. The classification is in `verification.md`.

`scripts/run-bats.sh` is the one place that runs bats in parallel on every OS: it counts CPUs with `getconf` (the old `nproc` does not exist on macOS), checks for GNU parallel with the same `::error` on both OSes, asserts the bash major version, and fails a tag filter that selects nothing.

## Out of scope

- X1, the from-zero run, and making `test-macos` required (it becomes required after it is green on main, a deliberate later edit of `forge/branch-protection.json`).
- Declaring GNU parallel in `packages.json`: the catalog has no system-package source yet (#2013 P5b). The macOS job installs it in a workflow step and points at that row.
- Python 3.12 and PyYAML on a developer's Mac: toolchains arrive with the mise toolchain rows; CI installs them with `actions/setup-python`.
- Moving `test-windows` to tag selection (noted on #2059).

## Risks / open questions

- The runner image is not this Mac: macOS 15 on the runner against 26 here (`sha256sum` is absent on 15, present in `/sbin` on 26). Mitigated by running the tier locally with `/sbin` off PATH; the first CI run is the real evidence.
- Tag selection is a silent-failure surface: bats ignores a near-miss tag comment, and an empty selection passes. Guarded by `tests/guard-bats-tags.bats` and by `run-bats.sh` failing on zero matches.
- Gap in layer 1: `checks_catalog.go` and `checks_repodir.go` read `runtime.GOOS` directly instead of `System.GOOS`, so their windows branches are not injectable. Ticketed as #2061, not fixed here.

## Acceptance criteria

- [ ] AC1: the Go layer 1 seams still run the darwin and windows branches on Linux (no work; the check is the existing tests).
- [ ] AC2: the OS-sensitive tier is tagged, non-empty, justified, and passes on this Mac under bash 3.2 with `/sbin` off PATH.
- [ ] AC3: a near-miss tag comment, an unknown tag and an empty selection each fail loudly.
- [ ] AC4: every bats failure on macOS is classified and resolved, and the full suite is green on this Mac.
- [ ] AC5: `ci.yml` has a non-required `test-macos` job on the existing path filter, running the tier under asserted bash 3.2 and zsh, Go build, native (darwin) vet and test, and the full suite on main.
- [ ] AC6: CPU count and the GNU parallel preflight come from one script, and `ci.yml` does not call `nproc`.

## References

- Bitácora board: #2013 (row W10), #1843 (E10)
- Lesson: `docs/lessons/lesson-342-linux-green-says-nothing-about-the-bsd-half-of-every-unix-tool.md`
- Related spec: `specs/PLAT-001c-tools-via-mise/`
