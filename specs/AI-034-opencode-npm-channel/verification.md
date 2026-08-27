---
tags: [spec, verification]
created: "2026-08-27"
---

# Verification - AI-034-opencode-npm-channel

## Evidence

All produced 2026-08-27 on the Windows work box (where the defect was measured), from the `feat/opencode-npm-channel` worktree.

`features.json` verification commands (Git Bash, Go 1.26, jq, bats 1.13):

```text
AI-034-opencode-npm-channel-f1: exit=0
AI-034-opencode-npm-channel-f2: exit=0
AI-034-opencode-npm-channel-f3: exit=0
AI-034-opencode-npm-channel-f4: exit=0
AI-034-opencode-npm-channel-f5: exit=0
```

Whole tree: `go build ./... && go vet ./... && GOOS=linux go vet ./...` clean; `go test ./internal/tools/ ./internal/cmd/ ./internal/doctor/` green; `golangci-lint run ./...` (2.12.2) 0 issues; `bash -n` + `shellcheck -S warning setup-linux.sh` clean; `setup-windows.ps1` parses (0 errors) with the non-ASCII byte count unchanged from `main` (32); `bats tests/opencode.bats tests/setup-windows.bats tests/versions-conf.bats tests/hive-upgrade-timer.bats tests/setup-linux.bats` — no failures beyond the pre-existing `zsh: command not found` cases on Windows.

**Real-box run** with `dotf` built from this branch:

```text
$ dotf tools version hive
3.0.0                      ← setup used to report "hive <unknown> predates 'hive service'"
$ dotf tools version opencode
1.16.2                     ← setup used to accept "locked." here
$ dotf tools version nope
Error: nope: no version found on PATH   (exit 1)
$ DOTFILES_DIR=<checkout> dotf tools install opencode
opencode 1.16.2 already installed; skipping
$ DOTFILES_REPO_DIR=<checkout> dotf doctor        # [OpenCode + pi]
  [WARN] opencode resolves from 3 PATH directories (…\scoop\apps\nodejs-lts\current\bin, …\scoop\shims, …\WinGet\Packages\SST.opencode_…) — `dotf tools install` converges the npm copy only; remove the other channel's copy
```

The last line is the finding this spec makes visible: three copies on the box, which setup had summarised as *"converge that install instead"*.

## Test status

| AC | Test | Status |
|---|---|---|
| AC1 | f1 (jq over `packages.json`, `versions.conf` refute); `opencode.bats` "packages.json is the only opencode pin" | pass |
| AC2 | `opencode.bats` (no install block, retired PATH line, post-deploy assertion), `setup-windows.bats` (no winget entry / pin machinery), `versions-conf.bats` | pass |
| AC3 | `TestProbeVersion_*` (7 banners), `TestToolsVersionCmd`, `hive-upgrade-timer.bats` parity guard | pass |
| AC4 | `TestCatalogPin_ReadsTheCheckoutBeforeTheMirror`, `TestCheckShadowedCatalogTools_*` (two dirs, one dir, duplicated entry) | pass |
| AC5 | f5 (ADR-036 present, migration consequence recorded) | pass |

## Decisions made during implementation

- **`pi` stays out of the catalog for now.** Its Linux install uses `--prefix ~/.local --ignore-scripts` (#426); `installNpm` cannot express that yet. Recorded in ADR-036 and on #1294 as the follow-up.
- **`versions.conf` loses `OPENCODE_VERSION` instead of mirroring it.** Two pins for one tool is the #693 drift class; the catalog is the SSOT for catalog tools, and doctor reads it there (`matchPinFrom`, source named in every message).
- **The winget loop lost its version-pin machinery entirely** rather than gaining a better parser: with opencode gone, no winget tool carried a pin, so the code path — and the StrictMode guard test written for it — was dead.
- **`ProbeVersion` uses output even on a non-zero exit.** Several tools print the version and then complain about something unrelated; an absent tool is distinguished by having no output at all.
- **Shadowed copies are a WARN, not a FAIL.** The tool runs; what is wrong is which copy `dotf tools install` can converge, and that is the operator's removal to make (ADR-036 §4).

## Promotion candidates

- None beyond the ADR: the channel policy is a build/operate decision and lives in `docs/adr/`.

## Archive checklist

- [ ] `dotf spec review AI-034-opencode-npm-channel` — passing `review.md` from the reviewer pool (needs an unlocked Bitwarden vault on the box that runs it)
- [ ] PR merged; issue #1294 closed
- [ ] `dotf spec archive AI-034-opencode-npm-channel`
