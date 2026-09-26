---
spec: "WIN-007-harness-mirror"
verdict: "PASS-WITH-GAPS"
reviewed_sha: "acf78b3cc37633aeac0c9548ac32944fbe7459e5"
reviewer: "nan/deepseek-v4-flash"
date: "2026-09-25"
---

## Adversarial review

**Scope**: `WIN-007-harness-mirror` — one Go mirror for both OSes, replacing `setup-linux.sh`'s bash+jq block and filling the block `setup-windows.ps1` never had.
**Sources**: `specs/WIN-007-harness-mirror/{proposal,tasks,verification,features}.md`; `git diff 8680e568b45c38f6287454933c55ff6202cb35f9...HEAD` (single commit `acf78b3`, the base the launcher resolved); `cli/internal/{harness,cmd,doctor,env}`, `setup-{linux.sh,windows.ps1}`, `scripts/install-dotf.{sh,ps1}`, `tests/*.bats`, `tests/Dockerfile.integration`, `.github/workflows/ci.yml`.

Every claim below was produced by running something in this session: the five `features.json` commands re-run with `-count=1`, a real-repo `dotf harness mirror` (76 updated → 0 updated), a fixture reproducing AC2, four mutation edits reverted afterwards, `go build/vet` (linux + `GOOS=windows`), `golangci-lint run ./...` (0 issues), `bats tests/setup-linux.bats tests/setup-windows.bats tests/integration-builds-dotf.bats`. Working tree left clean apart from this file.

### Spec and task alignment

| AC | Verified how | Result |
|---|---|---|
| AC1 mirror tree + manifest targets, idempotent | `TestMirror_CopiesTheTreeAndEveryDeclaredTarget`, `TestMirror_IsIdempotentAndDoesNotRewriteIdenticalFiles` (PASS, `-count=1`); real repo: run 1 `76 updated, 0 unchanged`, run 2 `0 updated, 76 unchanged`; `harness/` + all three declared targets present in a temp deploy dir | **met** |
| AC2 missing declared target named on stderr, exit 1, rest mirrored | `TestMirror_NamesADeclaredTargetTheCheckoutLacks`, `TestHarnessMirrorCmd` (PASS); fixture run printed `… declares a target the checkout does not have: ai/orca/ORCA.md`, `Error: …`, `3 updated, 0 unchanged`, exit 1 | **met** |
| AC3 no pruning | `TestMirror_DoesNotPrune` (PASS); `mirror.go` has no delete path on any mirror route (only temp-file cleanup on write failure) | **met** |
| AC4 both setups call it; bash+jq block deleted | `features.json` f4 exit 0; `bats setup-linux.bats` 60–64 PASS, `setup-windows.bats` 111–112 PASS; `grep` shows a single call site each (`setup-linux.sh:576`, `setup-windows.ps1:672`) and no `cp -rf … harness/.` | **met on Linux; Windows call site is source-asserted only** (see F1) |
| AC5 doctor orphans on Windows; pi-packages checkout-first | `TestCheckHarnessMirrorOrphans/windows:…` and `TestPiPackagesManifest_ReadsTheCheckoutBeforeTheMirror` PASS; both are binding, proven by mutation (restoring `if sys.GOOS == "windows" { return }` → subtest FAILs; reverting checkout-first → test FAILs) | **met** |

Task list: all implementation and closing boxes ticked; only "PR opened referencing this spec folder" is open, which is correct for the verification window. `features.json` has all five entries `pending` with empty `evidence` — consistent with the rule that only the harness may write `passing`; not a defect. No `[AGENT-DRAFT]`/`[AGENT-SUGGESTION]` tags remain.

Full-suite context (honest evidence, not a finding): `go test ./...` is *not* green on this box — `TestMergeAgainstTheRealDeployedSettings` fails (`bind_test.go:247: foreign hooks 17 -> 16 on the REAL file`). It asserts against the live deployed settings on this machine, and `git diff --name-only <base>...HEAD -- cli/internal/harness/` is exactly `mirror.go` + `mirror_test.go`, so nothing in the diff can reach it. Every `harness`, `cmd` and `doctor` test that the diff touches passes fresh.

### Findings

| Severity | Reality | Area | Finding | Evidence | Test (named, or UNTESTED) | Fix location (code / tests / spec / vault) |
|----------|---------|------|---------|----------|---------------------------|---------------------------------------------|
| Major | THEORETICAL | path-resolution (Windows) | `dotf harness mirror` resolves the checkout from the **process cwd** (`env.RepoDir()` walks up for `.git`, then falls back to `DOTFILES_REPO_DIR`), while `setup-windows.ps1` is written to be cwd-independent (`$DotfilesDir = $PSScriptRoot`) and exports `$env:DOTFILES_REPO_DIR` only at line ~2142 — **after** the mirror call at line 672. Launched from any directory that is not the checkout (e.g. right-click → Run with PowerShell, cwd `system32`; or from inside an unrelated git repo, where the walk-up finds the wrong root), the command cannot locate the checkout, mirrors nothing, and the `model-map`/`model-pins` doctor FAILs this spec exists to remove stay in place — against a printed remedy ("re-run setup") that still cannot clear them. | `cli/internal/env/env.go:161` (`RepoDir`); `setup-windows.ps1:75, 672, 2142`. Mechanism reproduced on Linux: `cd /tmp && env -u DOTFILES_REPO_DIR DOTFILES_DIR=/tmp/d /tmp/dotf-win007 harness mirror` → `Error: cannot locate the dotfiles checkout — set DOTFILES_REPO_DIR or run from inside it`, exit 1, 0 files mirrored. Documented invocations `cd` first (`README.md:43-45`, `docs/runbooks/ai-tools-setup.md:50`), which is why this is not REAL. | UNTESTED — the Windows guards (`tests/setup-windows.bats` 111–112) are source-text greps and cannot see the resolution; no test runs the command with cwd outside the checkout | code (`setup-windows.ps1`: set `$env:DOTFILES_REPO_DIR = $DotfilesDir` before the call, or `Push-Location $DotfilesDir` around it — the same value the script seeds into `machine.json` at line 1784) + tests |
| Minor | REAL | fidelity of the copy (mode bits) | The mirror writes every file `0o644`, so the exec bit is lost; the block it replaces preserved it. Measured: `harness/skills/systematic-debugging/find-polluter.sh` is `775` in the checkout, `644` in the mirror, and `775` under the old `cp -rf harness/. <deploy>/`. `harness/skills/systematic-debugging/root-cause-tracing.md:110-113` instructs the reader to run `./find-polluter.sh`, so a mirror-sourced execution would fail with permission denied. No consumer executes it from the mirror *today* (skills deploy from the checkout at `setup-linux.sh:1614`), hence Minor. | `stat -c %a` on source, mirrored copy and a `cp -rf` control; `mirror.go:180` (`os.Chmod(tmpName, 0o644)`) | UNTESTED — `TestMirror_IsIdempotentAndDoesNotRewriteIdenticalFiles` asserts bytes/mtime, never mode | code (derive the mode from the source file) + tests |
| Minor | REAL | scope | The diff carries changes no acceptance criterion or "What" bullet declares: `scripts/install-dotf.sh`, `scripts/install-dotf.ps1` (a source build reporting `dev` is never replaced by the pinned release — a user-visible installer behaviour change), `tests/Dockerfile.integration`, `.github/workflows/ci.yml`, `tests/integration-builds-dotf.bats`. The installer change is *needed* (otherwise `install_dotf` clobbers the container's source build and the integration job certifies the release again), and `tasks.md` ticks "No unrelated changes in the diff", but neither `proposal.md` nor `verification.md`'s "Decisions made during implementation" mentions it. | `git diff --stat <base>...HEAD`; `verification.md` decisions list; `install-dotf.sh` dev short-circuit (verified working: `dotf version` → `dotf version dev` → `grep -oE '…|dev'` selects `dev`) | named test exists but is static: `tests/integration-builds-dotf.bats` test 4 greps the installer text; no test executes the dev-gate | spec-as-record (`verification.md` disposition, or a follow-up ticket) — *not* the contract set |
| Minor | THEORETICAL | silent skip (Windows) | On Windows the mirror block is gated by `if (Get-Command dotf …)` with no `else`, so a machine where the freshly installed `dotf` is not yet visible to this process gets no mirror and no message naming the consequence; the Linux path logs exactly that warning. Same idiom as the adjacent `dotf tools install` gate, hence Minor. | `setup-windows.ps1:664-676` vs `setup-linux.sh:571-578` | UNTESTED | code + tests |

Not a finding, recorded so the next reader does not re-derive it: the `--refresh` ordering guard is *not* satisfied by prose. The comment above the call contains the string `dotf harness mirror` at `setup-linux.sh:555`, earlier than the call at 576, but moving the call above `--refresh` in a copy makes the guard fail (`bats tests/setup-linux.bats` case 64 uses the first match, which is then the call itself) — the guard is sound.

### Evaluator rubric

| Dimension | Grade (A-D) | Rationale (one line) |
|-----------|-------------|-----------------------|
| Correctness        | B | AC1–AC3 and AC5 verified end-to-end with red-green mutations; AC4's Windows half rests on a source grep and carries the cwd gap (F1). |
| Verification       | B | Evidence is reproducible and was re-run fresh here (5/5 `features.json` commands, 76→0 real run, 3 bats files, lint); Windows execution remains unproven on a box, which the spec itself accepts. |
| Scope              | C | Undeclared installer/CI/Dockerfile changes ride along, one of them user-visible; they are justified, but `tasks.md`'s "no scope creep" tick overstates the diff. |
| Reliability        | B | Error paths handled (no manifest, missing target, checkout == deploy, dotf absent on Linux), atomic temp+rename, no prune; interrupted runs can leave `.mirror-*` temps the "never prune" rule will not collect. |
| Maintainability    | A | `golangci-lint run ./...` 0 issues; longest function 37 lines (`mirrorFile`), `newHarnessMirrorCmd` 40 but largely a literal; flat control flow; comments carry the WHY (#1200/#1202/#802) and no dead code. |
| Handoff-readiness  | B | `verification.md` complete with commands and outputs and explicit decisions; the lesson is deferred to the DOCS-015 (#1303) batch as declared, and checksum/archive boxes are open as expected at this gate. |

### Verdict

PASS WITH GAPS

The severity axis: no Blocker, no REAL Major; the one Major is THEORETICAL, and the three Minor findings are dispositioned below. The rubric axis independently agrees (one C, no D). `dotf spec archive` is **advisable** once the gap this verdict tolerates is dispositioned; the contract set (`proposal.md`, `tasks.md`, `features.json`) must not be edited now — the verdict is about this state of it.

### Recommended next steps

Contract set is closed by this verdict: act on the items below in `verification.md` or in follow-up tickets, never by editing `proposal.md`/`tasks.md`/`features.json`.

- **F1 (Major/THEORETICAL, code+tests)** — Make the Windows call site cwd-independent: set `$env:DOTFILES_REPO_DIR = $DotfilesDir` (the checkout, as line 1784 already seeds) or `Push-Location $DotfilesDir` before `dotf harness mirror`. Add a guard that can fail: a bats case asserting the assignment precedes the call, or a `cmd` test running `newHarnessMirrorCmd` with cwd outside a checkout and the repo found via the env var. Worth doing because the fix is one line and the failure it prevents is the exact unfixed loop #1288 describes.
- **F2 (Minor/REAL, code+tests)** — Preserve the source file's mode in `mirrorFile` (e.g. `os.Chmod(tmpName, srcMode.Perm())`) and add `TestMirror_PreservesTheSourceMode` using a `0o755` fixture. `find-polluter.sh` is the concrete case; a future executable under `harness/` would otherwise arrive non-executable on every OS.
- **F3 (Minor/REAL, record)** — Record the installer/CI decisions in `verification.md` ("Decisions made during implementation") and, if the dev-build-frozen-on-a-release line should be visible to users, a follow-up ticket referencing the `install-dotf` dev short-circuit. Do not add it to the contract set in this round.
- **F4 (Minor/THEORETICAL, code+tests)** — Give the Windows mirror block an `else` that warns `dotf not found — harness not mirrored`, mirroring `setup-linux.sh:577`, so the two OSes fail the same way when the binary is invisible.
- **Evidence debt to carry, not to hide**: `go test ./...` is red on this box in `TestMergeAgainstTheRealDeployedSettings` (live deployed settings, untouched by the diff). If the archive evidence needs a green whole-suite run, produce it on a clean box or state the environmental cause in `verification.md`.
