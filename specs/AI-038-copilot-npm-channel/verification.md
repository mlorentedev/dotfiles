---
tags: [spec, verification]
created: "2026-08-28"
---

# Verification - AI-038-copilot-npm-channel

## Evidence

Run on the Windows work box, 2026-08-28, worktree `dotfiles-wt-setup-cluster`,
branch `feat/copilot-npm-channel`, `dotf` built from the branch.

- [x] **AC1/AC2** → `bats tests/copilot-config.bats tests/setup-windows.bats tests/opencode.bats tests/setup-linux.bats`: all ok
  (the two `zsh` cases of setup-linux need a zsh binary this box lacks; CI carries them).
  `packages.json` declares `copilot` (npm, `@github/copilot`, 1.0.81); `setup-windows.ps1` has no
  `Id = "GitHub.Copilot"` row; `ai/copilot/config.json` → `autoUpdate: false`.
- [x] **AC3** → `TestCheckCopilot_PinMatchByStatus` (5 rows: at pin PASS, above WARN drift, below
  WARN drift, no semver WARN without a drift line, absent SKIP). Mutation: the `matchPinFrom`
  call replaced by `_ = catalogPin` → `--- FAIL: TestCheckCopilot_PinMatchByStatus`; restored, ok.
- [x] **AC4** → ADR-036 table row moved; "Amendment 2026-08-28 (AI-038, #1321)" section.
- [x] **AC5** → box, in order:
  1. Before: `copilot` resolved only from the winget path; the binary had self-updated to
     1.0.81 (winget's registry said 1.0.78). With the winget copy satisfying the floor,
     `dotf tools install` did NOT add the npm copy — ADR-036 §5 in practice: retire the other
     channel first.
  2. `winget uninstall --id GitHub.Copilot --force` (the portable package had self-modified, so
     winget refused without `--force`) → `copilot` gone from PATH.
  3. Catalog mirrored (`~/.dotfiles/packages.json` from the branch) → `dotf tools install`:
     `copilot 1.0.81 installed via npm (@github/copilot)`; a second run: `already installed; skipping`.
  4. `Get-Command copilot -All` → only the npm shims under scoop's node bin; `copilot --version` →
     `GitHub Copilot CLI 1.0.81.`; `dotf tools version copilot` → `1.0.81`.
  5. `dotf doctor` with the branch catalog: `[GitHub Copilot CLI] (2 checks, all ok)`, no
     shadowed-install WARN; overall `131 passed, 0 failed`.
  6. Smoke: `copilot -p "... Reply with exactly one word: PONG"` → `PONG` (seat auth intact,
     30.3k tokens of instructions + skills loaded).

## Test status

```
go build ./... && go vet ./... && GOOS=windows go vet ./... && GOOS=linux go vet ./... && go test ./internal/doctor/   -> ok
golangci-lint run ./internal/doctor/...   -> 0 issues
bats tests/copilot-config.bats tests/setup-windows.bats tests/opencode.bats   -> all ok
```

- No regressions in the existing suite: yes.
- Found on the way, filed: **#1358** — `env.RepoDir` and doctor's DOTFILES_REPO_DIR check do not
  recognise a git worktree (`.git` is a file there), so branch-local `packages.json`/`ai/` changes are
  invisible to doctor and `dotf tools install` until merged; verification used the mirror copy.

## Decisions made during implementation

- **SKIP, not FAIL, when absent.** A box may deliberately carry no Copilot and setup deploys its
  config only when the binary is present; the catalog install is the remedy the SKIP names.
- **Exact-match PASS / drift WARN, like opencode.** The floor semantic lives in `dotf tools install`
  (never downgrades); doctor reports whether the installed version is the pinned one.
- **The winget copy is removed by hand, reported by doctor.** ADR-036 §4: a second copy on PATH is a
  finding, not a state to converge silently — and while it satisfies the floor the npm copy will
  not even be installed.
- **`autoUpdate: false`.** The pin is the floor and `dotf tools install` the updater; a
  self-updating binary is exactly what made winget's registry lie.

## Promotion candidates

- [ ] Lesson: no (the ADR amendment carries the record).
- [x] ADR-worthy decision: recorded as an amendment to ADR-036.

## Archive checklist

- [ ] `dotf spec review AI-038-copilot-npm-channel` PASS
- [ ] `proposal.md` frontmatter set to `status: archived`
- [ ] Folder moved to `specs/archive/AI-038-copilot-npm-channel/`
- [ ] Bitácora #1321 closed with the PR link
