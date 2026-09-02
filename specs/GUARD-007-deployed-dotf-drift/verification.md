---
tags: [spec, verification]
created: "2026-09-01"
---

# Verification — GUARD-007-deployed-dotf-drift

All evidence below was produced on 2026-09-01 in worktree `dotfiles-wt-dotfdrift`, branch `feat/detect-deployed-dotf-drift`, base `6e0d180`.

## The defect, reproduced before the fix

The installed binary and the pin agree, and the binary is two feature merges stale:

```
$ dotf version                     →  dotf version 0.52.0
$ grep DOTF_VERSION versions.conf  →  DOTF_VERSION=0.52.0
$ ls -l $(command -v dotf)         →  9.4M  29 Aug 21:59
```

The consequence, which version-equality cannot see — the same command, two binaries:

```
$ dotf doctor | grep -A2 'Persona skill enforcement'
(nothing — the section does not exist)

$ go build -o /tmp/dotf-main ./cmd/dotf && /tmp/dotf-main doctor
[Persona skill enforcement]
  [WARN] 31 of 35 persona skills carry no `enforce:` severity ...
```

`dotf doctor` reported `Results: 158 passed, 2 failed, 7 warned, 4 skipped` with a whole check section absent. That report is indistinguishable from one where the check ran and passed.

## AC-by-AC

### AC1 — the build stamps the commit; `--commit` prints it bare

Built with the ldflags goreleaser now emits, then asked:

```
$ go build -ldflags "-X main.version=0.52.0 -X main.commit=$(git rev-parse 11a68b1)" -o /tmp/fakebin/dotf ./cmd/dotf
$ /tmp/fakebin/dotf version
dotf version 0.52.0
$ /tmp/fakebin/dotf version --commit
11a68b1af73ce3bb2e0b427d5f31afc520c3285a
```

`go test ./internal/cmd/ -run TestVersionCommitFlagPrintsBareValue` — PASS.

### AC2 — the default output is unchanged, and the installers still parse it

`go test ./internal/cmd/ -run TestVersionDefaultOutputStaysSingleLineForTheInstallers` — PASS.

`~/.local/bin/bats tests/install-dotf.bats` — **9/9 ok**, including *"idempotent: a no-op when the pinned version is already on PATH"*, which is the assertion that would break if the default output grew a line.

### AC3 — behind HEAD on `cli/` WARNs, end to end against a real stamped binary

Not a fake: a real binary stamped with a real commit 7 commits back, placed on PATH, read by the real check.

```
$ PATH=/tmp/fakebin:$PATH /tmp/dotf-prov doctor | grep -A2 'dotf provenance'
[dotf provenance (deployed binary vs checkout)]
  [WARN] deployed dotf is 4 cli/ commit(s) behind HEAD (built 11a68b1af73c, HEAD 6e0d180ae57c)
         — it may run gates this tree no longer defines, or miss checks it does; reinstall to converge
```

7 commits back, 4 of which touch `cli/` — the scoping in AC4 working.

### AC4 — the pathspec is root-relative, proven fail-first

Measured while verifying AC3, and the reason this criterion exists at all:

```
from repo ROOT:              git rev-list --count 11a68b1..HEAD -- cli          → 4
from cli/ subdir:            git rev-list --count 11a68b1..HEAD -- cli          → 0
from cli/ with :(top) magic: git rev-list --count 11a68b1..HEAD -- ':(top)cli'  → 4
```

A bare pathspec resolves against git's CWD, so from `cli/` it means `cli/cli` and counts **0 — silently**. That is the failure mode this spec exists to end, reproduced inside its own fix.

Fail-first, by reintroducing the bug:

```
$ sed -i 's/":(top)cli"/"cli"/' internal/doctor/checks_dotf_provenance.go
$ go test ./internal/doctor/ -run 'RootRelativePathspec'
--- FAIL: TestCheckDotfProvenanceUsesRootRelativePathspec (0.00s)
    rev-list must scope with the root-relative pathspec, got:
    "rev-list --count 11a68b1c0ffee...aa..HEAD -- cli"
FAIL

$ (restored)
$ go test ./internal/doctor/ -run 'RootRelativePathspec'
ok  	github.com/mlorentedev/dotfiles/cli/internal/doctor	0.006s
```

### AC5 — each non-current state is reported distinctly

`go test ./internal/doctor/ -run 'TestCheckDotfProvenance$' -v` — **8/8 subtests PASS**, one per row of the proposal's state table.

The pre-stamp row is not only fake-tested — it is the live state of this machine, and the check reports it:

```
$ /tmp/dotf-prov doctor | grep -A2 'dotf provenance'
[dotf provenance (deployed binary vs checkout)]
  [WARN] deployed dotf (/home/manu/.local/bin/dotf) does not understand `version --commit`,
         so it predates the build stamp and its provenance CANNOT be established
         — reinstall from the current release to make this answerable
```

### AC6 — SKIPs where it cannot answer, never FAILs

Both skip cases pass (`outside a checkout the check skips rather than passing`, `TestCheckDotfProvenanceSkipsWhenDotfIsAbsent`). Every subtest asserts `rep.Failures() != 0` → failure, so "never FAILs" is checked in all 10 cases, not stated.

## Repo verification loop

```
cd cli
go build ./...              OK
go vet ./...                OK
GOOS=windows go vet ./...   OK      # the leg that broke on #1075
go test ./...               22 packages ok, 0 FAIL
golangci-lint run           0 issues   (v2.12.2 = the versions.conf pin)
```

`bats tests/install-dotf.bats` 9/9 ok. No `.sh`/`.ps1` files changed, so the shell layer is untouched by construction.

## features.json

All six verification commands executed; all six exit 0. `state` stays `pending` in the committed file — only the harness may write `passing`, and only with captured evidence.

## Not verified, and disclosed

- **The stamp on a real goreleaser build.** It lands only in a release build, and v0.53.0 has not published. Verified against a locally stamped binary using the same ldflags string instead. `goreleaser snapshot` runs on every PR touching `cli/**` and will exercise the template before merge.
- **Windows at runtime.** Compilation is checked (`GOOS=windows go vet`); no Windows box was available this session.
