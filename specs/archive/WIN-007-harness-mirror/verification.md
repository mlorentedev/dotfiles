---
tags: [spec, verification]
created: "2026-08-27"
---

# Verification - WIN-007-harness-mirror

## Evidence

All produced 2026-08-27 on the Windows work box (the box where the defect was measured), from the `feat/harness-mirror` worktree.

`features.json` verification commands (Git Bash, Go 1.26 on PATH):

```
WIN-007-harness-mirror-f1: exit=0  ok  github.com/mlorentedev/dotfiles/cli/internal/harness
WIN-007-harness-mirror-f2: exit=0  ok  github.com/mlorentedev/dotfiles/cli/internal/cmd
WIN-007-harness-mirror-f3: exit=0  ok  github.com/mlorentedev/dotfiles/cli/internal/harness
WIN-007-harness-mirror-f4: exit=0
WIN-007-harness-mirror-f5: exit=0  ok  github.com/mlorentedev/dotfiles/cli/internal/doctor
```

Whole tree: `go build ./... && go vet ./... && GOOS=linux go vet ./... && go test ./...` green; `golangci-lint run ./...` (pinned 2.12.2) 0 issues; `bash -n setup-linux.sh` + `shellcheck` (no new findings; pre-existing info-level notes only); `setup-windows.ps1` parses clean (PowerShell AST parser, 0 errors) and the inserted block is ASCII-only; `bats tests/setup-linux.bats tests/setup-windows.bats` — the rewritten and new guards pass (the one failing case, *valid zsh syntax*, is the pre-existing `zsh: command not found` on Windows).

**Real-box run** with `dotf` built from this branch:

```
$ dotf harness mirror
harness mirror: harness/ + 3 target(s) → C:\Users\<u>\.dotfiles (76 updated, 0 unchanged)
$ dotf harness mirror
harness mirror: harness/ + 3 target(s) → C:\Users\<u>\.dotfiles (0 updated, 76 unchanged)
$ dotf doctor | grep -E 'FAIL|Results'
  [FAIL] no DR escrow at ...                     ← WIN-011, fixed in #1304
  [FAIL] $HOME/.pi/agent/settings.json pi-deployed-default-model: "deepseek-v4-flash-0731" resolves to nothing the routing registry declares
  [FAIL] stale: .copilot/copilot-instructions.md ← WIN-008, comparator fixed in #1304
Results: 124 passed, 3 failed, 2 warned, 26 skipped   (was 122 / 4 / 2 / 25 before the mirror)
```

The two registry FAILs this spec exists for (`harness/model-map.json not found`, `read harness/model-pins.json`) are gone. The pin check, which had never been able to run on this box, now reports a genuine drift in the live pi settings (the "dead dated id" `model-pins.json` itself anticipates) — a finding, not a regression.

## Test status

| AC | Test | Status |
|---|---|---|
| AC1 | `TestMirror_CopiesTheTreeAndEveryDeclaredTarget`, `TestMirror_IsIdempotentAndDoesNotRewriteIdenticalFiles`, `TestHarnessMirrorCmd` | pass |
| AC2 | `TestMirror_NamesADeclaredTargetTheCheckoutLacks`, `TestHarnessMirrorCmd` (stderr + exit 1) | pass |
| AC3 | `TestMirror_DoesNotPrune` | pass |
| AC4 | `setup-linux.bats` (mirror via dotf; derivation in Go; ordering after `--refresh`), `setup-windows.bats` (call present; parity) | pass |
| AC5 | `TestCheckHarnessMirrorOrphans` (Windows subtest flipped), `TestPiPackagesManifest_ReadsTheCheckoutBeforeTheMirror` | pass |

Also: `TestMirror_RefusesWhenTheCheckoutIsTheDeployDir`, `TestMirror_FailsLoudWithoutAManifest`, `TestHarnessMirrorCmd_SaysSoWhenTheCheckoutIsTheDeployDir`.

## Decisions made during implementation

- **Narrow slice, not the engine.** `dotf harness mirror` only; `refresh/deploy/check` stay with CLI-026/CLI-035 (#495/#909). A second manifest-driven "deploy everything" design mid-drive was the rabbit hole named in review.
- **Never prune.** Kept #802's semantic: setup copies, `dotf doctor --fix` prunes. The Windows early-return in the orphan check was removed instead of extended, because the premise behind it ("Windows has no mirror") was false.
- **Missing target = exit 1 after mirroring the rest.** The bash block warned and continued; the Go command mirrors everything it can, names the gap on stderr and exits non-zero, and both setups keep the `|| log_warning` / `$LASTEXITCODE` degrade — loud, not fatal (C9 degrade-not-break).
- **Deleted, not ported, the jq-by-path guard (#1202).** jq is no longer involved; the test asserting the workaround would have asserted a workaround for a problem that no longer exists.
- **`env.RepoDir` prefers the cwd walk-up** (BUG-072), so the cmd test carries a `.git` marker and `t.Chdir`s into its fixture — the first draft resolved the real checkout and mirrored 76 files into a temp dir.

## Retroactive review (2026-09-25, W1.4 of #1625)

Reviewed from a detached worktree at the landing commit `acf78b3` (#1305), with that commit's own
`dotf` first on `PATH`, so the reviewer's scope was `acf78b3^...HEAD`. All five `features.json`
verifiers passed there first, running 15 tests. Because this tree predates SKILL-001 and the spec
is about the mirror, the deploy dir's `harness/` and the deployed skill lists were compared before
and after the review. Neither changed.

**Round 1, `nan/deepseek-v4-flash`, PASS WITH GAPS.** Transcript kept outside the repo
(`~/.local/state/dotf/review-transcripts/WIN-007-r1.jsonl`). Dispositions:

| Finding | Disposition |
|---|---|
| Major THEORETICAL: the mirror finds the checkout from the cwd, and `setup-windows.ps1` seeds `DOTFILES_REPO_DIR` only after the call | **Ticketed**: #1751 (WIN-014). Still true on `main`. The fix is an explicit `--repo` from both twins. HARNESS-139's new `dotf pi packages apply` takes that shape from the start. |
| Minor REAL: the mirror writes every file `0o644`, dropping the exec bit of `find-polluter.sh` | **Ticketed**: #1751 |
| Minor REAL: the installer and CI changes (`install-dotf` keeps a `dev` source build; the integration image builds dotf from the PR) are not in the proposal | **Applied**: recorded here. They were needed so the integration job certifies the PR's `dotf` and not the pinned release, which `install_dotf` would otherwise have put back. |
| Minor THEORETICAL: the Windows mirror block has no `else`, so a missing `dotf` is silent | **Ticketed**: #1751 |
| Evidence note: `TestMergeAgainstTheRealDeployedSettings` reads the live deployed settings and was red on the reviewer's box | **Declined**: untouched by this diff and environmental. It passes on `main` today. |

The one open `tasks.md` box ("PR opened referencing this spec folder") is ticked, and only the box:
#1305 is that PR.

## Promotion candidates

- Lesson (repo `docs/lessons/`): *a mirror that is partial on one OS while the code believes it is absent* — the early-return and its comment were the reason nobody looked; the check that could have flagged the gap was the one switched off. Written with the DOCS-015 (#1303) batch.
- No vault promotion: build/operate detail.

## Archive checklist

- [ ] `dotf spec review WIN-007-harness-mirror` — passing `review.md` from the reviewer pool (needs an unlocked Bitwarden vault on the box that runs it: the reviewer resolves its key through `dotf secrets run`)
- [ ] PR merged; issue #1288 closed
- [ ] `dotf spec archive WIN-007-harness-mirror`
