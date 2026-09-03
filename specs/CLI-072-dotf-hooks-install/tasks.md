---
tags: [spec, tasks, templates]
created: "2026-09-02"
---

# Tasks - CLI-072-dotf-hooks-install

> TDD order. One task = one focused commit. Tick as you go.

## Setup

- [x] Branch created from `origin/main` @ `fcef14a`: `feat/cli-072-dotf-hooks-install`
- [x] `proposal.md` complete, acceptance criteria testable
- [x] Open questions resolved before implementation: R1 (seam shape), R6 (Windows
      ordering) and the embed-vs-source-dir fork were all settled by measurement
      first — see `proposal.md`

## Implementation

> Destination refusals first: they are what protects the `rm -rf`, and they are
> where the two suites disagree most. Port and repoint before deleting anything
> (the ordering finding from OPS-043).

- [ ] [AC3] The `gitRunner` seam, with a fake that records `config --global`
      writes and returns canned `--get` output
- [ ] [AC1][AC2] Failing tests: destination refusals — a non-`*/git-hooks` dest,
      a drive root (the Pester-only case), `$HOME`, and `/`
- [ ] [AC1] `deployHooks` with the destination guards
- [ ] [AC2] Failing test: the `#695` self-mirror case — `--source X` resolving to
      the destination — plus the first-install case where the destination does
      not exist yet and `os.SameFile` cannot stat it (R3)
- [ ] [AC2] Self-mirror short-circuit via `os.SameFile`
- [ ] [AC2] Failing tests: clean mirror prunes a hook removed upstream; a missing
      source dir and a source without `pre-commit` both refuse
- [ ] [AC1][AC4] Mirror + write-time CR stripping + `0755` on the entrypoints
- [ ] [AC2] Failing tests: `core.hooksPath` wiring — unset gets wired; already
      wired is a no-op; a trailing-slash variant counts as wired; an equivalent
      dispatcher elsewhere is reported active, not INACTIVE; an unrelated
      pre-existing value is preserved and warned about
- [ ] [AC1] `wireHooksPath` + the guard-dispatcher probe
- [ ] [AC3] The one integration test: real `git`, throwaway `GIT_CONFIG_GLOBAL`,
      proving the fake speaks the real binary's dialect
- [ ] [AC1] Wire `dotf hooks install` into the command tree, with a `Run()`-level
      test that the subcommand is reachable (the wiring gap CLI-071's reviewer
      found: every test proving the function, none proving it is called)
- [ ] [AC5] Repoint `setup-linux.sh`
- [ ] [AC5][AC5b] Repoint `setup-windows.ps1` **and move the step after
      `Install-Dotf`** (R6), plus a guard asserting that ordering
- [ ] [AC6] Guard: doctor's target and the installer's destination are the same
      path, asserted across both rather than two constants that happen to agree
- [ ] [AC5] Update `checks_guard.go`'s FAIL remedy to `dotf hooks install` and
      `checks_tools.go`'s stale `install-git-hooks.ps1` comment
- [ ] [AC5] Delete `scripts/install-git-hooks.{sh,ps1}` and
      `tests/install-git-hooks.{bats,Tests.ps1}`
- [ ] [AC5] Prose sweep for referents to the deleted files — comments, docs,
      README, `architecture.md` — not just callers (lesson 259)

## Closing

- [ ] Every acceptance criterion covered by ≥1 named test
- [ ] `features.json` entries non-vacuous — `-v` piped through a `--- PASS:` grep,
      since `go test -run <no-match>` exits 0
- [ ] Every guard mutation-tested, `#695` first
- [ ] `go build ./... && go vet ./... && go test ./... && go test -race`
- [ ] `GOOS=windows go vet ./...`
- [ ] `golangci-lint run` at the `versions.conf` pin
- [ ] `shellcheck` + `bats tests/*.bats` (the shell layer changed)
- [ ] [AC7] Twin pairs 6 → 5 and the setup LOC delta recorded in `verification.md`
- [ ] PR opened with `Refs #1460` (not `Closes`: the spec-gate refuses a closing
      keyword on a PR that does not archive)

## Machine-readable features

See the sibling `features.json`.
