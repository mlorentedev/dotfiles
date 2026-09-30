---
spec: "CLI-090"
verdict: "PASS-WITH-GAPS"
reviewed_sha: "827fe52a5b01444117a16399d88dbe5d5999048e"
reviewer: "nan/deepseek-v4-flash"
date: "2026-09-30"
---

## Adversarial review

**Scope**: CLI-090 (cross-platform bootstrap and recovery updates), diff `9d611cb9f92bae0524d13c54c70693ff3bbeb600...HEAD` (`827fe52a`)
**Sources**: `specs/CLI-090/{proposal,tasks,verification,features}.md|json`; `scripts/install-dotf.sh`; `scripts/install-dotf.ps1`; `tests/install-dotf.bats`; `tests/install-dotf-ps1.Tests.ps1`; `tests/install-dotf-ps1.bats`; `README.md`; CI run 36738902605 (PR #1883 log).

### Spec and task alignment

- **All four ACs have named coverage**, and the CLI-090-attributable diff is exactly the files the proposal implies: `scripts/install-dotf.{sh,ps1}`, `tests/install-dotf.bats`, `tests/install-dotf-ps1.Tests.ps1`, `README.md`, plus `specs/CLI-090/`.
- **Nothing is stale by construction**: `proposal.md` and `tasks.md` reproduce the launcher's `contract_digests` byte-for-byte under the gate's normalisation, and `features.json` matches too once Go's HTML escaping of `&&` → `\u0026\u0026` in f4 is applied (`d293414f…`). No `[AGENT-DRAFT]` / `[AGENT-SUGGESTION]` tag is present in any of the three contract files.
- **Every `[x]` in `tasks.md` has diff or execution evidence.** Re-run here: `bats tests/install-dotf.bats tests/install-bootstrap.bats tests/install-dotf-ps1.bats` → 38/38 ok, exit 0; `shellcheck scripts/install-dotf.sh` → exit 0; `bash -n scripts/install-dotf.sh` → ok; features f4 command → exit 0.
- **Red-green holds for the new negative path.** Removing the semver guard in `_dotf_latest_version` (mutated to `if false; then`, then reverted) turns `@test "latest-release resolver rejects malformed release metadata"` red while the accept case stays green. The "never source a utils.sh from the working directory" guard is likewise exercised by `@test "a raw installer stream never sources a utils.sh from its current directory"`.
- **Evidence-integrity note**: the Pester proof in `verification.md` belongs to head `ffa74aeb`, not to `HEAD`. I fetched run 36738902605 (`gh run view 36738902605 --log`) and confirmed `tests/install-dotf-ps1.Tests.ps1 (10 tests)` and `Tests Passed: 87, Failed: 0`; that run's `headSha` is `ffa74aeb`, and PR #1883 is still open. `HEAD` (`827fe52a`) adds only `verification.md`, so the PS evidence is inherited rather than regenerated on the reviewed commit. Not a defect (docs-only delta), but the archive should not claim CI ran on `827fe52a`.
- **Scope is not the spec's to answer for**: the reviewed range opens with CLI-090's own merge (`66a90b75`, #1805) and then carries ~29 unrelated merged PRs (#1819…#1870), 181 files in `--stat`. The CLI-090 change itself is tight; the wide diff is a property of the base the launcher resolved, not scope creep by this change.
- **`pwsh` is not installed on this host** (verified: `which pwsh` empty), so the PowerShell half is verified by code reading plus the fetched CI log, never by local execution. That limitation is carried into every PS finding below.

### Findings

| Severity | Reality | Area | Finding | Evidence | Test (named, or UNTESTED) | Fix location (code / tests / spec / vault) |
|----------|---------|------|---------|----------|---------------------------|---------------------------------------------|
| Major | THEORETICAL | AC3 / resilience (PowerShell) | AC3 says a failed metadata/download/checksum/extraction step "exits non-zero". The POSIX side does (`install_dotf "$@"` is the script's last command). The PowerShell standalone guard discards the result — `$null = Install-Dotf -Version $Version …` — and `Install-Dotf` funnels every failure into `catch { Write-Warning …; return $false }`, so `pwsh -File scripts/install-dotf.ps1` ends at exit 0 after a failed latest-release lookup (the new `Get-DotfVersion` throw lands in that same catch). Not reproduced (no pwsh); argued from the run-guard and catch, both read directly. The guard is pre-existing, but this change made the raw path depend on it for failure visibility. | `scripts/install-dotf.ps1` run-guard + `Install-Dotf` catch; no test anywhere asserts a PS process exit code | UNTESTED | code + tests (assert the PS exit code, or narrow AC3 — contract set, so dispose in `verification.md` / a follow-up ticket) |
| Major | THEORETICAL | AC2 / verification (PowerShell end-to-end) | The README-documented Windows recovery (`irm … \| iex`) is never exercised end-to-end. The two Pester cases mock `Test-Path` and `Invoke-RestMethod` and only call `Get-DotfVersion`; `tests/install-dotf-ps1.bats` (tests 19–30) greps source text. POSIX has a real raw-stream install case (`a raw installer stream resolves its release without a checkout pin`); PowerShell has no counterpart that reaches a placed `dotf.exe` from an unpinned state, though pwsh is available in the Windows job. | `tests/install-dotf-ps1.Tests.ps1` `Describe 'Get-DotfVersion'` (resolver only); `tests/install-dotf-ps1.bats` static assertions | UNTESTED (installer path; resolver cases are named above) | tests (`tests/install-dotf-ps1.Tests.ps1`) |
| Minor | REAL | cross-shell path resolution (zsh) | `_DOTF_SOURCE="${BASH_SOURCE[0]:-}"` + `[ -f "$_DOTF_SOURCE" ]` drops the `$0` fallback the base used, so under zsh `_DOTF_SCRIPT_DIR` is empty where it previously resolved. Reproduced: `zsh -c 'source scripts/install-dotf.sh; echo $_DOTF_SCRIPT_DIR'` → `[]` on HEAD, `/tmp/ztest/scripts` on base. Consequence under zsh: the checkout `utils.sh` is never sourced (the five local fallbacks replace it) and the `versions.conf` branch is unreachable. No end-to-end break is demonstrated: `zsh scripts/install-dotf.sh` is a **no-op both before and after** (base and HEAD exit 0 with no output and no network call, because `! (return 0)` succeeds under zsh), so the executed path was already dead. AGENTS.md requires bash *and* zsh, and `docs/lessons/lesson-005-bash-source-0-is-empty-in-zsh.md` (touched in this same range) asks to test "bash, zsh, and raw-stdin execution"; no zsh case exists. | zsh A/B against `git show 9d611cb9:scripts/install-dotf.sh`; HEAD `DIR=[]` vs base `DIR=[/tmp/ztest/scripts]` | UNTESTED (no zsh case in `tests/install-dotf.bats`) | code (file-checked `$0` fallback) + tests |
| Minor | REAL | features.json f4 (contract set) | f4's `grep -q '^dotf version$' README.md` is not platform-scoped. Mutation: deleting the POSIX block's `dotf version` line leaves the f4 command at exit 0, because the Windows block's line still matches. The POSIX half of AC4's "states the final version assertion" is therefore unguarded, and `tasks.md` claims a "non-vacuous verification command". | mutation run (deleted POSIX `dotf version`; `features.json` f4 command still exit 0) | f4 itself (mutation evidence above) | features.json — **contract set**: do not edit under this verdict; carry as a follow-up ticket or fix at the next contract touch |
| Minor | REAL | docs / security | The new POSIX recovery one-liner (`curl … scripts/install-dotf.sh \| bash`) is presented two lines below a "Verify before piping" note that covers only `install.sh`. The recovery path resolves an arbitrary latest release and installs it, so it deserves the same inspection advice. | `README.md` recovery-vs-bootstrap block | UNTESTED (docs) | docs (`README.md`) |
| Minor | THEORETICAL | maintainability / error taxonomy (PowerShell) | The new `try/catch` in `Get-DotfVersion` wraps the deliberately distinct malformed-metadata error inside the lookup-failure error, so a non-semver tag is reported as `latest-release lookup failed: latest-release metadata has no semver tag`. AC2 asks for malformed metadata to "fail loudly"; the test only passes because it globs `*no semver tag*`. | `scripts/install-dotf.ps1` `Get-DotfVersion`; Pester case `refuses release metadata whose tag is not a semver version` | named Pester case above (it would pass on either message shape) | code |
| Minor | THEORETICAL | maintainability (POSIX) | `_dotf_latest_version` parses JSON with `sed 's/.*"tag_name"…/'` + `head -n1`: a line carrying more than one `tag_name` field yields the last, and a tag whose numeric prefix is followed by a quote inside a longer string is accepted. No repro; both fixture shapes (`{"tag_name":"v9.9.9"}`, `not-a-version`) are covered. | code read; `tests/install-dotf.bats` "latest-release resolver accepts/rejects …" | named bats cases above | code |

### Evaluator rubric

| Dimension | Grade (A-D) | Rationale (one line) |
|-----------|-------------|----------------------|
| Correctness        | B | All four criteria met on the exercised paths (POSIX re-verified here); negative-path gaps: PS installer end-to-end and AC3's exit-code/metadata sub-cases. |
| Verification       | B | Every POSIX claim reproduced in this session (38/38 bats, shellcheck 0, f4 0, red-green on the semver guard); PS half confirmed from the fetched CI log but not reproducible locally; f4's platform-scope mutation hole. |
| Scope              | B | The CLI-090 diff matches the proposal; the 181-file range carries ~29 unrelated merges because of the resolved base, which is not attributable creep. |
| Reliability        | B | Failure ordering is sound (resolve → fetch → verify → stage-beside-target → rename) so no failure touches an installed binary; the PS exit-status gap is tracked above. |
| Maintainability    | B | Short functions and WHY-comments; minor brittle sed JSON parsing and a conflated PS error message. |
| Handoff-readiness  | A | proposal/tasks/verification/features all updated; lesson-005 updated in the same range; promotion candidates answered (`no` each with a reason). |

### Verdict
PASS WITH GAPS

Two open Majors, both **THEORETICAL** (argued from code, PowerShell not executable on this host), plus five Minors — no Blocker and no REAL Major, and no rubric D. The `severity × reality` rule is what decides this, not severity alone.

### Recommended next steps

Route by set. The contract set is **closed** by this verdict: editing `proposal.md`, `tasks.md` or `features.json` now would invalidate the review that permitted the archive.

- **`verification.md` (outside the contract set — record a disposition per finding: applied, ticketed, or declined with a reason):** findings 1, 2, 3, 5, 6, 7.
- **Follow-up ticket (do not fix in this spec):** AC3's platform scope and features.json f4's platform-scoped `dotf version` assertion — the two contract-set items. Note f4 currently goes green when the POSIX assertion is deleted.
- **Code + tests, if the two theoretical Majors are to be closed:** an end-to-end Pester case that drives the unpinned PS installer to a placed `dotf.exe` and asserts the process exit code on failure; a zsh case in `tests/install-dotf.bats` for path resolution (restore a file-checked `$0` fallback in `scripts/install-dotf.sh` first); a distinct error message for malformed metadata.
- **Docs:** add the "verify before piping" note to the recovery one-liner in `README.md`.
- **Archive:** `dotf spec archive CLI-090` is advisable in the current state — `PASS-WITH-GAPS` is a recognized passing verdict (`cli/internal/spec/review.go` `Blocks()`), the contract digests are fresh, and no `[AGENT-DRAFT]` tag remains. The minimum actions to flip the verdict to PASS would be fixing finding 3 and adding the PS end-to-end/exit-code tests (findings 1–2); none of them blocks the archive.
