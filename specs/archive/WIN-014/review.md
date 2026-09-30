---
spec: "WIN-014"
verdict: "PASS"
reviewed_sha: "b776e9da2b18dce2587adce65ad5f18bf4218978"
reviewer: "nan/deepseek-v4-flash"
date: "2026-09-30"
---

## Adversarial review

**Scope**: WIN-014 — Reliable Windows harness mirroring (`dotf harness mirror --repo`, mode preservation, setup-twin wiring and warnings)
**Sources**: `specs/WIN-014/{proposal,tasks,verification,features.json}`; `git diff b15ad970f98c1ec6de1a2e51defc944cbbb0280a...HEAD` (the launcher-stated base). The WIN-014-attributable change in that range is commit `79c89b43` (`fix(harness): make mirror checkout explicit (#1806)`); `git rev-parse 79c89b43^` == the stated base. The remaining 32 commits in the range are other merged PRs — see the scope note below.

### Scope note (read before the findings)

`b15ad970…HEAD` is 33 commits (`git rev-list --count`). The stated base is exactly the parent of the spec's landing commit, so only `79c89b43` is WIN-014's work; the other 32 commits are unrelated merged PRs that landed afterwards. `git log b15ad97..HEAD -- cli/internal/harness/mirror.go cli/internal/cmd/harness_mirror.go setup-linux.sh setup-windows.ps1 tests/setup-linux.bats tests/setup-windows.bats` returns `79c89b43` and one doc-link commit (`20aa19ec`, 2 lines, no behavior). I reviewed the WIN-014-attributable diff exhaustively and the WIN-014 surface's current state at HEAD; I did **not** adversarially review the other 32 merges (**UNVERIFIED**, and not this spec's contract). The 6-line `setup-windows.ps1` delta in the range (4 lines here + 2 doc lines) was checked line by line.

### Spec and task alignment

- All three acceptance criteria are implemented and I reproduced their evidence (commands and outputs under "Evidence produced this session").
- `tasks.md` checkboxes are all `[x]`; every claim maps to diff evidence (see alignment table):
  - `[AC1]` explicit checkout command coverage → `cli/internal/cmd/harness_mirror.go` + two new cmd tests.
  - `[AC2]` mode preservation → `cli/internal/harness/mirror.go` `mirrorFile` + `TestMirror_PreservesTheSourceMode`.
  - `[AC3]` setup twins → `setup-linux.sh:567`, `setup-windows.ps1:725` + bats cases.
- `verification.md` names the right tests for all three criteria. Two of its statements are not reproducible in this environment (Windows Go run, WSL BATS full run) — I reproduced the Linux-side equivalents instead and say so below.
- No `[AGENT-DRAFT]`/`[AGENT-SUGGESTION]` tags in `proposal.md`, `tasks.md` or `verification.md` (a grep over the folder also matches the launcher's own `review-transcript.jsonl`, which is not a spec artifact).
- Correct call sites: `$CURRENT_DIR` (`setup-linux.sh:22`, `$(pwd)`, required to be the checkout — the script sources `./scripts/utils.sh` relative to cwd and refuses `CURRENT_DIR == DOTFILES_DIR`) and `$DotfilesDir` (`setup-windows.ps1:75`, `= $PSScriptRoot`) are both the checkout, not the deploy dir. The Windows twin therefore passes the checkout, not the deploy target.

### Evidence produced this session

| # | Command | Result |
|---|---------|--------|
| E1 | `go build ./...` (in `cli/`) | exit 0 |
| E2 | `go test ./internal/harness/... ./internal/cmd/...` | both `ok` (0.270s / 1.832s) |
| E3 | `go test ./...` (in `cli/`) | no `FAIL` line in output |
| E4 | mutation A — `os.Chmod(tmpName, info.Mode().Perm())` → `0o644` | `TestMirror_PreservesTheSourceMode` FAIL: `destination mode = 0644, want 0775` (red → green) |
| E5 | mutation B — drop the perm-equality branch (bytes-equal ⇒ `Unchanged`) | `TestMirror_PreservesTheSourceMode` FAIL; `TestMirror_IsIdempotentAndDoesNotRewriteIdenticalFiles` still PASS |
| E6 | mutation C — ignore `--repo`, use `env.RepoDir()` only | both `TestHarnessMirrorCmd_*ExplicitRepo*` FAIL with the "cannot locate the dotfiles checkout" error (red → green) |
| E7 | `bats --filter WIN-014 tests/setup-linux.bats tests/setup-windows.bats` | `1..2`, both `ok` |
| E8 | `gofmt -l` on the four touched Go files | no output (exit 0) |
| E9 | `go vet ./internal/harness/ ./internal/cmd/` | exit 0 |
| E10 | `go run ./cmd/dotf harness mirror --help` | `--repo string   dotfiles checkout to mirror from` present |
| E11 | crafted `harness/manifest.json` target `"../escaped.txt"`, then `HOME=… DOTFILES_DIR=…/out go run ./cmd/dotf harness mirror --repo …/repo` | `harness mirror: … (3 updated, 0 unchanged)`; the file landed at `…/home/escaped.txt`, one level **above** the deploy dir `…/home/out` — confirms the traversal finding below |

All mutation edits were reverted; `git diff --stat` is empty (working tree clean apart from this review).

### Findings

| Severity | Reality | Area | Finding | Evidence | Test (named, or UNTESTED) | Fix location (code / tests / spec / vault) |
|----------|---------|------|---------|----------|---------------------------|---------------------------------------------|
| Minor | THEORETICAL | test coverage | AC1 says the explicit checkout "copies the named checkout's harness **and manifest targets**", but both new explicit-repo tests use a fixture manifest of `{"targets":[]}`, so the manifest-target half of AC1 is never exercised through `--repo`. An explicit-repo regression that only breaks manifest targets would not be caught. Risk is low because `Mirror` takes one `repoRoot` for both the tree and the targets. | `cli/internal/cmd/harness_mirror_test.go` fixtures (`{"targets":[]}`); `mirror.go` `Mirror` uses one `repoRoot` for `mirrorTree` and the targets loop | UNTESTED | tests |
| Minor | THEORETICAL | negative path | `proposal.md` "Risks" requires that an explicit repo lacking the manifest "must still fail clearly". The failure exists in `Mirror` (`reading harness/manifest.json: …`), but no test drives it through the command with `--repo`; `TestMirror_FailsLoudWithoutAManifest` exercises `Mirror`, not the flag path. | `mirror.go` `manifestTargets` error wrap; `harness_mirror.go` returns the error unwrapped; no `--repo` negative test | UNTESTED | tests |
| Minor | REAL | test strength | The Linux half of AC3 asserts `grep -qF 'dotf not found (PATH or ~/.local/bin)'`, a string that already existed before this change and is not part of the diff; that assertion passes on the pre-change script and pins nothing new. The Windows half is meaningful (the `else { Write-Warn … }` branch is new). Both bats cases are static source inspections — they never execute setup — so "passes its checkout" is proven by string match, not by running the twin. | `git show 79c89b43 -- tests/setup-linux.bats` adds only the assert; the warning line is unchanged; `verification.md` says the cases "passed" without distinguishing static from executed | `@test "setup-linux.sh passes its checkout explicitly and warns when dotf is unavailable (WIN-014)"` (runs, but its second `grep` is vacuous) | tests |
| Minor | REAL | handoff-readiness | `features.json` is still `state: "pending"` with empty `evidence` for all three features, 33 commits after the change merged. Per the file's own gating rule the agent may not write `passing`, so this is the harness's job — but no consumer has run the three `verification` commands and captured exit 0, so the machine-readable contract under-describes a change that is already in `main`. | `specs/WIN-014/features.json` (all `pending`, `"evidence": ""`); `tasks.md` box `[x] Every acceptance criterion has a matching entry in features.json` | UNTESTED (no named harness capture) | spec (`verification.md` disposition; do **not** hand-edit `features.json` — it is in the contract set) |
| Minor | REAL | path traversal (pre-existing) | A manifest target containing `..` escapes the deploy dir: `mirrorFile` writes `filepath.Join(deployDir, rel)` with no containment check, so a crafted `harness/manifest.json` writes outside `$DOTFILES_DIR`. Reproduced this session (E11). **Not introduced by `79c89b43`** — the target loop and `manifestTargets` are unchanged — but `--repo` is a new public flag that names an explicit source, so the input is now user-facing. Not escalated: whoever supplies the manifest also supplies `harness/` (and, on the setup path, `scripts/utils.sh`, which setup sources), so this crosses no boundary that running that checkout does not already cross. Escalate to Major if this repo ever mirrors a checkout it did not author. | E11; `mirror.go` targets loop `dst := filepath.Join(deployDir, rel)`; `manifestTargets` only skips empty/duplicate entries | UNTESTED (`TestMirror_*` fixtures use `{"targets":[]}` or in-tree targets only) | code (+ tests) or a ticket with root cause |
| Question / assumption | SPECULATIVE | reliability | On a destination filesystem that cannot store POSIX mode bits (FAT/exFAT, a WSL `drvfs` mount, a read-only-bits FS), `os.Chmod` in the mirror either silently does not converge or fails, so a re-run could keep reporting `Updated > 0` — contradicting the documented "converged re-run reports 0 updated". On Windows Go reports `0666` for both sides so the compare passes; on Linux the normal case converges (E5). Not observed anywhere. | `mirrorFile` perm compare + `os.Chmod`; no test on such a filesystem | UNTESTED | code/tests (only if a real deployment target is named) |

No Blocker, no REAL Major. On the changed hunks: `--repo` is a local path (no network/URL surface), file modes only move from the checkout to the deploy dir under the same user, errors from `os.Stat`/`ReadFile`/`Chmod`/`Rename` are wrapped and returned, and the write remains atomic (temp file in the destination dir + rename). No hardcoded secret, no injection, no auth path, no unbounded loop and no blocking call was introduced. `INFO` check: no `0666`-mode file is tracked under `harness/`, so mode preservation cannot widen a checked-out file's permissions in practice — exactly one `100755` file exists (`harness/skills/systematic-debugging/find-polluter.sh`), which is the real file the exec-bit half of AC2 protects. The one traversal finding is pre-existing and carries its own disposition below.

### Evaluator rubric

| Dimension | Grade (A-D) | Rationale (one line) |
|-----------|-------------|----------------------|
| Correctness        | B | All three ACs verified by reproduced tests (E2–E7); minor negative-path/coverage gaps (findings 1–2). |
| Verification       | B | `verification.md` names the right tests and I reproduced the Linux side (E2–E7); the Windows/ScriptAnalyzer claims are not reproducible here and the setup "warns" evidence for Linux is partly vacuous. |
| Scope              | A | The WIN-014-attributable diff (`79c89b43`) matches the proposal exactly, with the two setup twins, the CLI flag and the mirror change only. |
| Reliability        | B | Atomic write, wrapped errors, idempotence asserted by an existing test; the non-POSIX-filesystem convergence path is unproven (finding 5). |
| Maintainability    | A | Small functions, no nesting above ~2, `--repo` precedence readable in three lines, comments explain why; gofmt/vet clean (E8, E9). |
| Handoff-readiness  | B | Spec artifacts and promotion answers are present and honest, but `features.json` evidence is still empty after merge (finding 4). |

No dimension is C or D. Mechanical aggregation plus the severity axis (no Blocker, no REAL Major) yields **PASS**.

### Verdict

PASS

### Recommended next steps

Contract set (`proposal.md`, `tasks.md`, `features.json`) is **closed** by this verdict — do not edit it, or the verdict goes stale and a re-review is required. Disposition the tracked minors instead, in `verification.md` (outside the staleness set) or as a follow-up ticket:

- Record in `verification.md`, one line each, the disposition of findings 1–3: apply the cheap test additions (`--repo` with a manifest target; `--repo` at a path with no manifest; a Windows-only or remove the vacuous Linux `grep`) or decline with a reason.
- Finding 4: have the harness run the three `features.json` `verification` commands and capture exit 0, or state in `verification.md` why the machine-readable contract stays `pending` for an already-merged change. Do not hand-edit `features.json`.
- Finding 5 (traversal, pre-existing): ticket it on the board with root cause (`manifestTargets` accepts `..`; the write destination is unvalidated) and fix options (reject a target whose cleaned path escapes `repoRoot` or `deployDir`), rather than folding it into this archive. It does not gate WIN-014's archive.
- Finding 6: no action unless a real deployment target on a non-POSIX-mode filesystem is named; if so, ticket it with the filesystem as the reproduction.
- Windows-only evidence (`go test` on Windows, PSScriptAnalyzer) could not be reproduced from this Linux session and stays **UNVERIFIED** here; if Windows CI does not run these, add the Windows run to the evidence before archive.
