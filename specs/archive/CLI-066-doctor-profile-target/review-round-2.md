---
spec: "CLI-066-doctor-profile-target"
verdict: "FAIL"
reviewed_sha: "d054816fc67c4b6b88d567ba2588d340973e7b71"
reviewer: "nan/deepseek-v4-flash"
date: "2026-09-30"
---

## Adversarial review

**Scope**: `CLI-066-doctor-profile-target` — the whole spec change: `dab595e` ("fix(doctor): the profile doctor measures is the one pwsh names, and the heal is told which file", #1379) plus the round-1 fix `d054816` (HEAD), read at `HEAD = d054816`.
**Sources**: `specs/CLI-066-doctor-profile-target/{proposal,tasks,verification,features}.md`, `review-round-1.md`; `cli/internal/doctor/checks_profile.go`, `checks_profile_target_test.go`, `checks_profile_heal_test.go`, `checks_profile_thresholds_test.go`, `checks_profile_test.go`; `scripts/profile-heal.ps1`; `tests/profile-heal-ps1.bats`; diff `8de1cd1996...HEAD` (see finding 2 on that range).

### Spec and task alignment

- `tasks.md`: every `[x]` has diff evidence in `dab595e` or `d054816` — the four Go test files, the `-ProfilePath` parameter, the bats case, the spec folder, and the round-1 line. No tick without a diff.
- **AC1/AC2** (measure what pwsh names, enumerate otherwise, row says which): implemented in `profileTarget`; `TestCheckProfileFiles_MeasuresThePwshResolvedProfile` asserts the exact question, the 10 s bound, that pwsh is not asked when absent, the redirected path in the row, and — new in `d054816` — the last-line and `.ps1` shape rules.
- **AC3** (`--fix` heals the file it measured, bounded seam): the code does it and the argv assertion now distinguishes the two sources — round-1 finding 2 is genuinely closed (proved by mutation below). **But one clause of the round-1 Major-1 fix is unguarded**: `verification.md` claims "`--fix` does not run the heal for it" and no test exercises that — finding 1.
- **AC4** (`-ProfilePath`, `$PROFILE` default): `scripts/profile-heal.ps1:190` defaults as claimed; the parameter and the default are pinned both by the bats case and by `TestProfileHealThresholdsMatchTheScript`. `pwsh` is absent on this host, so the bats syntax case skips and PSScriptAnalyzer could not be re-run — the analyzer claim is **UNVERIFIED** here, not disputed.
- **AC5** (thresholds linked): **non-vacuous, proved by mutation this run.** `profileMaxBytes = 1<<20` → `1<<21` fails `TestProfileHealThresholdsMatchTheScript`. The test also pins the marker strings and the `-ProfilePath` contract, which is more than the AC asked.
- **AC6** (Windows box): a transcript dated 2026-08-29 in `verification.md`; not reproducible on this host — **UNVERIFIED**, accepted as the only evidence of the real question-and-answer path.
- The proposal's timeout claim ("on a timeout doctor falls back to the enumeration") holds by branch equivalence: a timeout is an `err` from the bounded seam, which is the path the "pwsh present but fails to answer" subtest exercises. No test names a timeout; none is needed for the branch.
- Suite evidence run fresh at this sha: `go build ./...` OK, `go vet ./...` OK, `GOOS=windows go vet ./...` OK, `go test ./... -count=1` exit 0, `go test ./internal/doctor/ -count=1` ok, `bats tests/profile-heal-ps1.bats` 16/16 with case 13 self-skipping (no pwsh).

### Findings

| Severity | Reality | Area | Finding | Evidence | Test (named, or UNTESTED) | Fix location (code / tests / spec / vault) |
|----------|---------|------|---------|----------|---------------------------|---------------------------------------------|
| Major | REAL | tests / AC3 negative path | Round-1 Major 1 had two halves: the false BUG-020 diagnosis **and** "`--fix` runs the heal for a file that has never existed". `d054816` fixes both in code, but only the diagnosis is tested. The existing missing-profile case runs with `fix=false`; no test runs `fix=true` against a pwsh-named nonexistent profile. The exact defect round 1 classified REAL can come back on the `--fix` path with nothing to catch it. | Mutation this run: `if !pathExists(profile) {` → `if !pathExists(profile) && !fix {`, reintroducing the round-1 defect for `--fix`. `go test ./internal/doctor/ -count=1` → **ok**; `go test ./... -count=1` → **no FAIL lines**. Mutant reverted (`git checkout`), suite re-run green, tree clean. | **UNTESTED** — the fix is a subtest running `checkProfileFiles(sys, nil, rep, true)` with a pwsh-named profile that does not exist, asserting one FAIL containing `PowerShell profile missing:` and `unwant: "BUG-020"`, plus a heal fake that `t.Fatal`s if invoked (the existing `run` harness already `t.Fatal`s on the unbounded seam, so the same trick works). | tests |
| Minor | THEORETICAL | review scope / launcher base | The stated base `8de1cd1996…HEAD` is **991 files, +76,544/−5,934, 230 commits** — a whole-history range, not this spec's change (`dab595e` + `d054816` ≈ 15 files). The spec's work is squash-merged months behind HEAD, so the resolved base is stale for it (the HARNESS-112 / SDD-042 failure family, and round 1's question 5). I reviewed the spec's own two commits and the HEAD state of every file they touch, and checked `git log 8de1cd1..HEAD -- <those paths>`: only `dab595e` and `d054816`, no later edits, so the change is intact at HEAD. | `git diff --stat 8de1cd1…HEAD`; `git show --stat dab595e d054816`; `git log --oneline 8de1cd1..HEAD -- cli/internal/doctor/checks_profile.go scripts/profile-heal.ps1 tests/profile-heal-ps1.bats` | UNTESTED (the base-resolution defect is another spec's) | vault / follow-up ticket on the launcher (#1551, #1645) — not this change |
| Minor | THEORETICAL | features.json / AC6 verifier | f6's verification command is `test "$(uname -s '\|' cut -c1-5)" = MINGW && …`. Off Windows the `test` fails, so the command exits 1 while the feature is marked `state: "verified"` — a reader on Linux/CI cannot reproduce it and cannot tell "box-only" from "broken". The evidence is a box transcript, which is legitimate; the command's shape is not self-describing. | Read of `features.json` f6; `uname -s` on this host is `Linux`, so the command exits 1 here. | UNTESTED | spec (contract set — allowed now, because this round is FAIL; otherwise route the note through `verification.md`) |
| Minor | SPECULATIVE | doctor / message hygiene | When pwsh answers something unusable, the fallback row echoes pwsh's *first* output line via `firstLineOr(out, err)` (`enumerated, pwsh did not answer $PROFILE (WARNING: some module banner); …`). Module banners can embed paths or environment detail into a FAIL line. The "resolved" case asserts the noise is absent (`unwant: "WARNING"`); the fallback case asserts only the enumeration substring. | Code read of `profileTarget`; no reproduction with sensitive noise, no box observation. | UNTESTED | code (echo only a reasoned error, not the raw first line) — surface only, does not gate |
| Minor | REAL | tooling (not this change) | `golangci-lint` was not run in this window; the verification claim of "0 issues" at the pinned version is unreproduced here. | not run | UNTESTED | — (recorded as a review limitation) |

Nothing in the spec folder carries `[AGENT-DRAFT]` or `[AGENT-SUGGESTION]` (grepped; the only hits are inside `review-transcript.jsonl`, which echoes this skill's own text).

Round-1 dispositions, re-checked rather than accepted: Major 1 — applied, and the message pin plus the mutant below confirm the diagnosis half (M2). Major 2 — applied, and the heal fixture now lives in `Redirected/Docs/` with a decoy in the first enumerated root, so the round-1 mutant goes red (M1). Minor 3 (quoting) — applied, both paths are double-quoted and `TestCheckProfileFiles_DetectsBUG020Corruption` asserts it (M3 red when the quotes are removed). Minor 4 (answer shape) — applied: last non-blank line, `.ps1` suffix, else fall back (M5 and M6 red when each rule is removed).

*Mutations run this session, all reverted, tree clean apart from the untracked `review-request.json` and this deliverable:*

| # | Mutation | Result (expected red) |
|---|----------|----------------------|
| M1 | `--fix` heals the enumerated file (`profile = split` before the heal) | `TestCheckProfileFiles_FixRunsTheHealAndVerifiesByConsequence` **FAIL** ✅ |
| M2 | drop the missing-profile early return | `TestCheckProfileFiles_MeasuresThePwshResolvedProfile` **FAIL** ✅ |
| M3 | unquote the hand-run remedy | `TestCheckProfileFiles_DetectsBUG020Corruption` **FAIL** ✅ |
| M4 | `profileMaxBytes` → `1<<21` | `TestProfileHealThresholdsMatchTheScript` **FAIL** ✅ |
| M5 | accept any pwsh answer (drop the `.ps1` check) | `TestCheckProfileFiles_MeasuresThePwshResolvedProfile` **FAIL** ✅ |
| M6 | `lastLine` → `firstLine` | `TestCheckProfileFiles_MeasuresThePwshResolvedProfile` **FAIL** ✅ |
| M7 | missing-profile guard applies only when `!fix` | **whole suite passes** ❌ → finding 1 |

### Evaluator rubric

| Dimension | Grade (A-D) | Rationale (one line) |
|-----------|-------------|----------------------|
| Correctness | B | AC1–AC5 met and the code is right on every path I exercised, including the `--fix` + missing one; the gap is that the `--fix` half of round-1 Major 1 is not guarded. |
| Verification | C | Six of seven mutations land red and the AC5/AC3 pins are genuinely load-bearing, but `verification.md` states "`--fix` does not run the heal for it" with no named test, and AC6 has only a transcript. |
| Scope | A | The spec's two commits touch exactly the files the proposal names (doctor check + four tests, the heal script, its bats case, the spec folder, one lesson); the 991-file range is the launcher's stale base, not creep. |
| Reliability | B | Both shell-outs are bounded (10 s / 60 s), every fallback names its reason in the row, the heal is verified by consequence and never by exit code, and the heal's own preflight still bounds what it reads. |
| Maintainability | A | `profileTarget` ~15 lines, `checkProfileFiles` ~35, no CC > 10, comments state WHY (the CLI-064 split and the round-1 review), script stays ASCII. |
| Handoff-readiness | B | Spec complete, tasks ticked with evidence, a lesson captured in `d054816`, round-1 findings dispositioned; one verification claim is unbacked. |

### Verdict

**FAIL** — one **REAL Major** with **no named covering test**, the same class and the same mutation method that failed round 1. Rubric has a C but no D, which alone would only require PASS WITH GAPS; the reality axis is the more severe path and governs.

The change is otherwise in good shape and the round-1 fixes are real, not cosmetic: every mutant for every claim the spec makes now lands red, including both round-1 Majors and both Minors. What remains is one unguarded clause of the fix — the `--fix` path for a profile that was never written — which is the exact symptom round 1 asked to be closed.

`dotf spec archive` is **not advisable** in this state, and would refuse anyway, since this is a FAIL.

### Recommended next steps

The blocking fix lands **outside the contract set** (tests only), so it does not invalidate the next round's digest check.

1. **Finding 1 (tests).** Add the subtest named above. The harness in `checks_profile_heal_test.go` already has everything it needs: run `checkProfileFiles(fx, nil, rep, true)` where the fake `$PROFILE` answer names a path that does not exist and the enumeration finds nothing, assert exactly one FAIL containing `PowerShell profile missing:` with `BUG-020` absent, and have the heal fake `t.Fatal`. Then re-run mutant M7 from this review: the suite must go red.
2. **Findings 2 and 3 (not this change's to fix, and minor).** Record the dispositions in `verification.md` (`verification.md` is outside the staleness set): the base-range defect is the launcher's (#1551, #1645); f6's Windows-only verifier is a box criterion and its command should read as such.
3. **Finding 4 (surface only, does not gate).** Consider echoing only the error, not pwsh's first output line, in the fallback row.
4. Then re-run `dotf spec review CLI-066-doctor-profile-target`. The re-review must re-run the M7 mutant (the claim this round could not confirm) and may re-run M1/M2 to confirm the round-1 fixes stayed in place.

*Reviewer's own evidence, this session:* `go build ./...` / `go vet ./...` / `GOOS=windows go vet ./...` OK; `go test ./... -count=1` exit 0; `go test ./internal/doctor/ -count=1` ok; `bats tests/profile-heal-ps1.bats` 16 ok, 1 skip (no pwsh); seven mutations applied and reverted (six red as expected, M7 survived), working tree clean apart from the untracked `review-request.json` and the `review.md` deliverable.
