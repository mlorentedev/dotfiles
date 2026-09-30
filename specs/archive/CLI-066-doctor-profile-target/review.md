---
spec: "CLI-066-doctor-profile-target"
verdict: "PASS"
reviewed_sha: "35053b3107e838321985c777e9f8287c8f99013b"
reviewer: "nan/deepseek-v4-flash"
date: "2026-09-30"
---

## Adversarial review

**Scope**: `CLI-066-doctor-profile-target` — the whole spec change at `HEAD = 35053b31`: `dab595e3` ("fix(doctor): the profile doctor measures is the one pwsh names, and the heal is told which file", #1379), the round-1 fix `7a7bd2b1`, and the round-2 fix `35053b31`.
**Sources**: `specs/CLI-066-doctor-profile-target/{proposal,tasks,verification,features}.md`, `review-round-1.md`, `review-round-2.md`; `cli/internal/doctor/checks_profile.go`, `checks_profile_target_test.go`, `checks_profile_heal_test.go`, `checks_profile_test.go`, `checks_profile_thresholds_test.go`, `fs.go`; `scripts/profile-heal.ps1`; `tests/profile-heal-ps1.bats`; diff `8de1cd1996...HEAD` (see finding 4 on that range).

### Spec and task alignment

- `tasks.md`: every `[x]` has diff evidence in `dab595e3`, `7a7bd2b1` or `35053b31` — the doctor check and its four test files, the `-ProfilePath` parameter, the bats case, the spec folder, the round-1 and round-2 review lines. No tick without a diff, no diff without a tick.
- **AC1/AC2** (measure what pwsh names; enumerate otherwise; row says which): implemented in `profileTarget`; `TestCheckProfileFiles_MeasuresThePwshResolvedProfile` asserts the exact question (`-NoProfile -Command $PROFILE`), the `profileQueryTimeout` bound on the seam, that pwsh is not asked when absent, the redirected path in the row, the last-line and `.ps1` shape rules. **Proved non-vacuous by mutation this run** (M2b, M5, M6 below).
- **AC3** (`--fix` heals the file it measured, bounded seam; never runs for a never-written profile): code correct and now **both** halves guarded. The heal fixture lives in `home/Redirected/Docs/…` with a decoy in the first enumerated root, so M1 (heal the enumerated file) goes red; the new `TestCheckProfileFiles_FixDoesNotHealAMissingProfile` makes M7 red. Round-2's REAL Major is closed.
- **AC4** (`-ProfilePath`, `$PROFILE` default, ASCII): `scripts/profile-heal.ps1` defaults as claimed; `grep -cP '[^\x00-\x7F]'` → `0`. `pwsh` is absent on this host, so the bats syntax case self-skips and PSScriptAnalyzer could not be re-run — the analyzer claim is **UNVERIFIED** here, not disputed.
- **AC5** (thresholds linked): non-vacuous, proved by mutation (M4 red). The test also pins the marker strings and the `-ProfilePath` contract, more than the AC asked.
- **AC6** (Windows box): a transcript dated 2026-08-29 in `verification.md`; not reproducible on this host — **UNVERIFIED**, accepted as the only evidence of the real question-and-answer path.
- The proposal's timeout claim ("on a timeout doctor falls back to the enumeration and says so") holds by branch equivalence: a timeout is an `err` from the bounded seam, the path the "pwsh present but fails to answer" subtest exercises. No test names a timeout; none is needed for the branch.

Nothing in the spec folder carries a live `[AGENT-DRAFT]` or `[AGENT-SUGGESTION]` marker: this run called the archive gate's own `FindUnresolvedTags` against the folder and got **0 hits** (the markers in `review-round-1.md:35` / `review-round-2.md:35` sit inside backticks, which `ScanUnresolvedTags` strips).

### Findings

| Severity | Reality | Area | Finding | Evidence | Test (named, or UNTESTED) | Fix location (code / tests / spec / vault) |
|----------|---------|------|---------|----------|---------------------------|---------------------------------------------|
| Minor | REAL | spec artifact / archive pre-flight | `verification.md`'s "Promotion candidates" section is not archive-ready. Its ADR and pattern lines are bare `no` with no reason, and the archive's third pre-flight ("No flag skips it") refuses exactly that; its lesson line answers `no` while `7a7bd2b1` added `docs/lessons/lesson-321-a-test-that-proves-which-source-won-needs-them-to-disagree.md`. So `dotf spec archive` will refuse today, and one answer contradicts the diff. | Probed the tool's own function at HEAD: `CheckPromotions(repo, specDir, …)` → `PROBLEMS=2`: ``ADR-worthy decision for the repo's `docs/adr/adr-XXX.md`? "no" needs a reason`` and the same for the `00_meta/patterns/` line. The lesson line parses (em-dash separator) but disagrees with `git show --stat 7a7bd2b1`. | UNTESTED (`promotion_test.go` covers the parser, not this spec's file) | spec — `verification.md` only (outside the contract set, so this does not invalidate this verdict) |
| Minor | REAL | quality | `checks_profile.go`: `firstLineOr`'s doc comment ("renders a subprocess result as one line…") is now attached to `lastLine`, which it does not describe, and `firstLineOr` has no comment. Introduced by `7a7bd2b1`. | File read of `checks_profile.go` around the two functions; `git log -S 'lastLine returns the last non-blank line'` → `7a7bd2b1`. | UNTESTED (comment) | code (comment-only) |
| Minor | THEORETICAL | features.json / AC6 | f6's verification command is `test "$(uname -s | cut -c1-5)" = MINGW && …`; off Windows it exits 1 while the feature is marked `state: "verified"`, so a CI reader cannot tell "box-only" from "broken". | `uname -s` here is `Linux`; read of `features.json` f6. | UNTESTED | spec — record as a box-only criterion in `verification.md`; the contract set is closed on this PASS, so do **not** edit `features.json` |
| Minor | SPECULATIVE | doctor / message hygiene | The pwsh fallback row echoes pwsh's *first* stdout line via `firstLineOr(out, err)` into a FAIL (`enumerated, pwsh did not answer $PROFILE (WARNING: …)`). A module banner could embed a path or environment detail in the row. Round 2 declined this as the only clue to why pwsh was silent; re-surfaced, does not gate. | Code read of `profileTarget`; the "resolved" case asserts noise is absent, the fallback case asserts only the enumeration substring. | UNTESTED | code — surface only |
| Minor | SPECULATIVE | doctor / target shape | `pathExists` follows symlinks (`os.Stat`), so a dangling symlink at the pwsh-named path is reported `PowerShell profile missing: … — run setup-windows.ps1`. No regression (the enumeration path did the same before), but the remedy is imprecise for a box where setup *has* run. | `cli/internal/doctor/fs.go:12-15`. | UNTESTED | code — surface only |
| Question | — | review scope / launcher base | The stated base `8de1cd1996…HEAD` is **232 commits, 995 files, +76,973/−5,934** — a whole-branch range, not this spec's change (`dab595e3` + `7a7bd2b1` + `35053b31` ≈ 15 files). The spec's own three commits are intact at HEAD with no later edit to their paths (checked with `git log <base>..HEAD -- <paths>`), so the verdict covers this spec. This is the HARNESS-112 / SDD-042 failure family, not this change's. | `git diff --stat 8de1cd1…HEAD`; `git log --oneline 8de1cd1..HEAD -- cli/internal/doctor/checks_profile.go scripts/profile-heal.ps1 tests/profile-heal-ps1.bats` → the three commits only. | UNTESTED (another spec's defect) | vault / follow-up ticket on the launcher (#1551, #1645) |

**Round-1 and round-2 dispositions, re-checked rather than accepted.** Every round-1 finding was already closed in round 2; every round-2 finding is now closed, and I re-ran the mutants behind each rather than reading the claims:

*Mutations run this session, all reverted, tree clean:*

| # | Mutation | Result (expected red) |
|---|----------|-----------------------|
| M7 | missing-profile guard applies only when `!fix` (round 2's blocker) | `TestCheckProfileFiles_FixDoesNotHealAMissingProfile` **FAIL** ✅ |
| M1 | `--fix` heals the enumerated file (`profile = split` before the heal) | `TestCheckProfileFiles_FixRunsTheHealAndVerifiesByConsequence` **FAIL** ✅ |
| M2b | disable the missing-profile guard | `…/pwsh_answers_a_path_that_does_not_exist_yet…` **FAIL** ✅ |
| M3 | unquote the hand-run remedy | `TestCheckProfileFiles_DetectsBUG020Corruption` **FAIL** ✅ |
| M4 | `profileMaxBytes` → `1<<21` | `TestProfileHealThresholdsMatchTheScript` **FAIL** ✅ |
| M5 | accept any pwsh answer (drop the `.ps1` check) | `…/pwsh_answers_something_that_is_not_a_profile_path…` **FAIL** ✅ |
| M6 | `lastLine` → `firstLine` | `…/pwsh_prints_noise_before_the_path…` **FAIL** ✅ |

Round-1 Major 1 (false BUG-020 on a never-written profile) — applied in `7a7bd2b1`, and M7 in round 2 found the `--fix` half unguarded; `35053b31` closes it (M7 red now, passed the whole suite in round 2). Round-1 Major 2 (heal target indistinguishable) — M1 red. Minors 3 (quoting, M3 red) and 4 (answer shape, M5/M6 red) — applied. Round-2 Minor on `golangci-lint` — **UNVERIFIED** here (not run this window; recorded as a limitation). Round-2 Minors on the base range, f6 and the fallback echo — carried above with dispositions.

### Evaluator rubric

| Dimension | Grade (A-D) | Rationale (one line) |
|-----------|-------------|----------------------|
| Correctness | A | AC1–AC5 met by tests that all survive the mutation battery, and the round-1/round-2 negative paths (`--fix` + never-written profile, heal-target source) are covered; no observed defect. |
| Verification | B | Evidence proves each criterion and reproduces here, but AC6 is a box transcript and the PSScriptAnalyzer claim needs a Windows host (no pwsh here) — covered, not reproducible without context. |
| Scope | A | The spec's three commits touch exactly the files the proposal names (doctor check + four tests, the heal script, its bats case, the spec folder, one lesson); the 995-file range is the launcher's stale base, not creep. |
| Reliability | B | Both shell-outs bounded (10 s / 60 s), every fallback names its reason in the row, the heal is verified by consequence and never by exit code; only the dangling-symlink wording is imprecise. |
| Maintainability | B | `profileTarget` ~20 lines, `checkProfileFiles` ~40, no CC > 10, comments explain WHY; one doc comment is misattached to the wrong function. |
| Handoff-readiness | B | Spec complete, tasks ticked with evidence, rounds 1 and 2 dispositioned, a lesson captured; the promotion section is not archive-ready and one answer contradicts the diff. |

### Verdict

**PASS** — no Blockers and no REAL Majors; the round-2 REAL Major is closed and proved by mutation (M7 red). Rubric is all B or above with no D, so the mechanical aggregation is PASS. The remaining findings are Minors, each tracked with a disposition; the two SPECULATIVE ones do not move the verdict by rule.

The change is in good shape. Its central guarantee — detect and heal agreeing on one target because pwsh is asked the same question the heal would — is implemented and pinned by tests that now distinguish both sources on both the measure and the `--fix` paths, which is what three review rounds were about. The threshold pin (`TestProfileHealThresholdsMatchTheScript`) remains the strongest artifact: mutation in both directions turns it red.

`dotf spec archive` is **advisable as a change, but will refuse today** until finding 1 is fixed: the third pre-flight reads `verification.md`'s "Promotion candidates", and two bare `no` answers fail it (reproduced with the tool's own `CheckPromotions`; no flag skips it). That fix is a `verification.md` edit, outside the contract set, so it does not stale this verdict.

### Recommended next steps

The contract set (`proposal.md`, `tasks.md`, `features.json`) is **closed** by this PASS — do not edit it alongside this verdict. Everything below lands outside it.

1. **Finding 1 (spec, `verification.md`).** Give the ADR and pattern lines a reason each (`no: nothing decision-shaped…`), and change the lesson line to the truth: this change *did* add `docs/lessons/lesson-321-a-test-that-proves-which-source-won-needs-them-to-disagree.md`, so it should read `yes: docs/lessons/lesson-321-…md` (a repo path resolves against the repo root) or `no: <reason>` if the line is meant to ask about vault promotion only. Then re-run the archive — this is the only thing standing between this change and the archive gate.
2. **Finding 2 (code).** Reattach the "renders a subprocess result as one line…" comment to `firstLineOr` and write a real one for `lastLine`.
3. **Findings 3–5 (dispositions).** Record in `verification.md` (it is outside the staleness set): f6 is a box-only criterion and its command is expected to fail off Windows; the fallback echo of pwsh's first line is accepted as-is; the dangling-symlink wording is tracked or declined. None of these gate.
4. **Finding 6 (not this change's).** The stale launcher base is already tracked in #1551 and #1645; no action here.

*Reviewer's own evidence, this session:* `go build ./...` exit 0; `go vet ./...` exit 0; `go test ./... -count=1` no `FAIL`/`panic` lines; `go test ./internal/doctor/ -count=1` ok; seven mutants applied and reverted (M7, M1, M2b, M3, M4, M5, M6 — all red), working tree clean apart from the untracked `review-request.json` and this deliverable; two probe tests run and deleted, one for `FindUnresolvedTags` (0 hits) and one for `CheckPromotions` (2 problems, finding 1). `bats tests/profile-heal-ps1.bats` and `golangci-lint` were **not** run in this window (no `pwsh` on this host; time budget) — recorded as UNVERIFIED, not disputed.
