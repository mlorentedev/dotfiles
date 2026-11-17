---
spec: "CI-017-gate-required-check"
verdict: "PASS-WITH-GAPS"
reviewed_sha: "3a6283ff107a25befd806de6aa60a0b8cb2a070a"
reviewer: "nan/qwen3.8-flash"
date: "2026-11-12"
---

## Adversarial review

**Scope**: CI-017-gate-required-check (dotfiles #3306; PR #3387 = PR 1 the `ci-gate` job, PR #3419 = PR 2 protection + Windows job-level skip)
**Sources**: `specs/CI-017-gate-required-check/{proposal,tasks,verification,features}.json` + `git diff 39bbdb36...HEAD`, restricted to the spec's own commits `027a882b`, `35bdbe72`, `3a6283ff`; live GitHub state via `gh api` (protection, check-runs, run jobs); `tests/lib/check-workflow-contexts.py`; bats files `workflow-job-names`, `ci-os-matrix`, `ci-path-filtering`, `ci-npm-cli-cache`.

Contract digests in `review-request.json` match `dotf spec digest CI-017-gate-required-check` for `features.json`, `proposal.md`, `tasks.md` — this review is about the current contract. No `[AGENT-DRAFT]` / `[AGENT-SUGGESTION]` tags remain in the spec folder.

### Spec and task alignment

- **AC1 — verified.** `ci.yml` has `ci-gate` with `if: always()`, `needs` = the other 8 jobs, `uses: mlorentedev/ci-workflows/.github/actions/gate@2355330358344bb08c0b7a305be4ed6488f23d0c`, `with.needs: ${{ toJSON(needs) }}`. Re-ran at HEAD: `bats tests/workflow-job-names.bats` → exit 0 (the new case is test 7 of 7); `python3 tests/lib/check-workflow-contexts.py .` → exit 0; `actionlint .github/workflows/ci.yml` → exit 0. The pin is real, not a moving tag: `gh api repos/mlorentedev/ci-workflows/git/ref/tags/v0.6.0` → object `2355330358344bb08c0b7a305be4ed6488f23d0c`, exactly the SHA in the `uses:` line.
- **AC2 — verified, and verified better than the artifact does.** Production evidence, read this session: run `43356781947` (`test` failure → `ci-gate` **failure**), run `43356346861` and run `43357097683` (jobs `cancelled`, nothing failed → `ci-gate` **failure**) — so `always()` did run the gate past a failed/cancelled need and the composite judged red. `6929dd90` (PR #3387 head) `ci-gate` conclusion `success`, matching `verification.md`.
- **AC3 — verified against live ground truth.** `gh api repos/mlorentedev/dotfiles/branches/main/protection` → `["review-attestation","cli-gate","spec-gate","knowledge-gate","from-zero-gate","ci-gate"]`, i.e. `ci-gate` in place of `changes`/`lint`/`lint-powershell`/`test`/`test-windows`, and byte-equal to `forge/branch-protection.json`. `dotf forge protection check` itself was not re-run here (owner/auth path); the live read above is the stronger form of the same check.
- **AC4 — verified end-to-end.** Docs-only PR runs `43357086470` and `43356756224`: `test-windows`, `test-macos`, `pi-nan-package` = **skipped** (no runner), `ci-gate` = **success**. So the job-level skip does what CI-013 could not do while `test-windows` was required.
- **Guard relocation is sound.** `test-windows`' new job-level `if` is byte-identical to the condition its 13 step guards carried, so the two reductions are equivalent inside the job: `failure() && (push || code)` → `failure()`, and `(push || code) && push && !cache-hit` → `push && !cache-hit`. At HEAD only those two steps keep an `if:`.
- **Scope clean.** The three spec commits touch only `ci.yml`, `forge/branch-protection.json`, four `tests/*.bats` and the spec folder. The other 15 commits in the range arrived through merges and none interacts with the gate (checked `42ba1f13`'s `persist-credentials` lines, which the `ci-gate`/`test-windows` hunks sit beside).
- **Tasks.md** is ticked consistently with the diff for every implementation box; the two closing boxes are overstated for AC3 and AC2 respectively — findings 2 and 3.

### Fail-closed behaviour I demonstrated (mutation probes, all reverted; tree clean)

| Probe | Result |
|---|---|
| Drop `integration` from `ci-gate.needs` | test 7 red **and** checker red (`aggregate gate ci.yml:ci-gate does not need 'integration'`) |
| Remove `if: always()` from `ci-gate` | test 7 red **and** checker red ×8 (rule 5, one per need) — `verification.md` credits only the test; the checker also holds it |
| Delete `test-windows`' job-level `if:` | CI-013 red, HARNESS-041 red (checker green, correctly: nothing required is skipped) |
| Point the job-level `if:` at the wrong output (`powershell`) | HARNESS-041 red — the new `continue` does not accept a wrong-output guard |
| Re-add `test-windows` to `forge/branch-protection.json` while its job can skip | checker red ×2 (rule 4 + rule 5) |
| Rename the `ci-gate` job | checker red (`required check 'ci-gate' is reported by no pull-request job: it would stay pending`) **and** test 7 red |
| **Remove the `github.event_name == 'push' \|\|` disjunct from `test-windows`' job `if:`** | **ALL GREEN** — 23/23 bats, checker exit 0. See finding 1. |

### Findings

| Severity | Reality | Area | Finding | Evidence | Test (named, or UNTESTED) | Fix location (code / tests / spec / vault) |
|---|---|---|---|---|---|---|
| Major | THEORETICAL | tests / `ci.yml` | The job-level guard now carries CI-013's "every push runs them" half alone, and nothing asserts the `github.event_name == 'push'` disjunct. If it is ever dropped, `changes.code == 'false'` on a docs-only **push to main** skips `test-windows` entirely — main has no merge gate to notice. Mutation survived: removing the disjunct left all 23 cases and the checker green. **Inherited, not introduced**: I replayed the same edit on `39bbdb36`'s `ci.yml` (strip `push \|\|` from the step guards) and HARNESS-041's rule still reported zero unguarded steps, so the property was never tested — but this change deleted the last file where the disjunct was spelled out per step for `test-windows`, and its own comment (`# …a documentation-only pull request starts no Windows runner`) now promises it. | `bats tests/ci-os-matrix.bats tests/ci-path-filtering.bats tests/workflow-job-names.bats tests/ci-npm-cli-cache.bats` → `1..23`, no `not ok`; checker exit 0 — both with the disjunct present and without it | UNTESTED. Home is `CI-013: every Windows or macOS job on a pull request is gated before a runner starts` (`tests/ci-os-matrix.bats`), whose `gated = "needs.changes.outputs." in str(job.get("if",""))` is output-agnostic and event-blind | tests |
| Minor | REAL | verification / `features.json` | `CI-017-gate-required-check-f2`'s verification command is `gh api …/commits/$(git rev-parse HEAD)/check-runs … \| grep -qx success`. Re-run verbatim at the reviewed SHA it exits 1: `{"message":"No commit found for SHA: 3a6283ff…","status":"422"}` — `3a6283ff` is unpushed, and even when pushed a docs-only commit's `ci-gate` is not `success` by construction. The command can only pass on the one PR head the evidence already names, so `state: "passing"` is a snapshot, not a reproducible check, and `tasks.md`'s closing box "every acceptance criterion has … a non-vacuous verification command" is ticked on its strength. The criterion itself is unaffected — I reproduced AC2 from live runs instead. | `gh api repos/mlorentedev/dotfiles/commits/3a6283ff…/check-runs` → exit 1; contrast run `43356346861` → `ci-gate` failure, `43357086470` → success | f2 itself (the artifact is the defect, not the path) | spec (`features.json` is in the contract set, so fixing it means fix + re-review; disposition it in `verification.md` instead — see next steps) |
| Minor | THEORETICAL | tests / `forge/branch-protection.json` | AC3 — the spec's headline deliverable — has no repo test asserting the required set. Deleting `ci-gate` from the declaration passes bats, the CI-004 checker and `dotf lint fleet` green: rule 2 checks required ⇒ reported, never "ci.yml is aggregated by a required gate". Mitigations that do exist and I verified: `dotf doctor` (`cli/internal/doctor/checks_branch_protection.go`) compares declared vs live, the checker fails closed on the opposite edit (probe 5), and live protection today is correct. So the direction left unguarded is a widening of the merge gate by a config edit. | grep for `required_status_checks` in `tests/` hits only `ci-os-matrix.bats` (reads it to exempt required jobs) and `workflow-job-names.bats` (synthetic fixtures); probe 5 + probe 6 above | UNTESTED (no named case asserts `ci-gate ∈ required checks` and the five `ci.yml` contexts ∉) | tests |
| Question | UNVERIFIED | vault / fleet naming | ADR-001 decision 3 names the check `gate`; ci-workflows' own declaration requires `gate` (`forge/branch-protection.json` → `mlorentedev/ci-workflows: ["gate"]`), while dotfiles now requires `ci-gate` for the reason `spec-gate.yml` already reports `gate`. The deviation is recorded in `proposal.md`'s risks and `verification.md`'s decisions, but both leave the spec folder at archive and #3306 closes with it, so the reconciliation ("rename spec-gate's job and amend the ADR, or amend ADR-001 to say `ci-gate`") has no tracker I could find inside this repo. Owner call; I did not search ci-workflows' issues. | `proposal.md` Risks bullet 1; `forge/branch-protection.json` | n/a | vault (a ci-workflows issue or an ADR-001 amendment), plus the ADR-001 literal name the spec leaves out of scope |
| Minor | SPECULATIVE | observability | One required context per workflow removes per-job granularity from the merge box: a red `ci-gate` points at a run, not at `test` or `integration`; the eight checks that used to name the failing job are still reported, just not required. This is what ADR-001 decision 3 buys and what the fleet gate pattern accepts, so surface only — do not gate on it. | Live protection list (6 contexts for 5 workflows' worth of ci.yml jobs) | n/a | — (surface only) |

### Evaluator rubric

| Dimension | Grade (A-D) | Rationale (one line) |
|---|---|---|
| Correctness | A | All four ACs hold, each proven against live GitHub state or runs rather than against the spec's own narration; negative paths (failed need, cancelled need, skipped need) observed. |
| Verification | B | f1 and f4 re-run green here, checker and `actionlint` exit 0, `dotf lint fleet` reproduces the exact three #3336 findings claimed; f2 is a snapshot that fails when re-run (finding 2) and f3 needs owner auth, though the live `gh api` read replaces it. |
| Scope | A | Three commits touching `ci.yml`, the protection declaration, four test files and the spec folder — nothing the proposal does not name; the merged-in 15 commits are untouched. |
| Reliability | B | Fail-closed in six of seven directions I probed, and the composite is SHA-pinned so a tag move cannot silently widen it; the judgement itself lives in ci-workflows (`tests/gate.bats`), unreachable from here → the gate's internals are UNVERIFIED, covered only by production behaviour. |
| Maintainability | B | `ci-gate` carries a comment that states why `always()` and why the name; HARNESS-041's new `continue` is one branch, commented, and still output-specific; the step-guard removal left the job shorter, not cleverer. No function-length or CC>10 surface introduced. |
| Handoff-readiness | B | PR #3387 and #3419 both carry `## Knowledge`; #3419 has `## Review triage`; the preflight-timeout defect is a real open issue (#3434, verified); promotion lines answered with reasons; the naming deviation is the one loose thread (finding 4). |

### Verdict

**PASS WITH GAPS**

No Blocker. No rubric D, no rubric C. Every open Major is THEORETICAL and, on the evidence I gathered, inherited from CI-013/HARNESS-041 rather than created here — so it is tracked with a disposition, not blocking. `dotf spec archive` is **advisable** for this contract: the review is fresh (digests match), the verdict is passing, and the SHA examined is the current HEAD. Archive it on the contract as it stands; do **not** edit `proposal.md`, `tasks.md` or `features.json` in the same pass, which would invalidate this review.

### Recommended next steps

All of these belong **outside** the contract set (code, tests, `verification.md`, or a follow-up ticket) — on a PASS WITH GAPS the contract is closed, and a `features.json`/`tasks.md` edit here would orphan this verdict. Disposition each one in `verification.md` (applied / ticketed / declined with a reason) or file it:

1. **Finding 1 (tests, the one worth scheduling).** Add a named assertion that a job-level OS gate keeps `github.event_name == 'push'` — extend `CI-013: every Windows or macOS job on a pull request is gated before a runner starts` to reject a job `if:` that can skip on a push, for `test-windows`, `test-macos` and `pi-nan-package` together. Verify by mutation: the same edit I made must go red.
2. **Finding 3 (tests).** One bats case asserting `ci-gate` is in `forge/branch-protection.json`'s required contexts and `changes`/`lint`/`lint-powershell`/`test`/`test-windows` are not, so a widening of the merge gate by config edit fails CI instead of waiting for `dotf doctor`.
3. **Finding 2 (`verification.md`, not `features.json`).** Record here that f2's command is only satisfiable on a PR head where `ci-gate` succeeded, and cite the re-runnable substitutes proven this session — runs `43356781947` (failure), `43356346861`/`43357097683` (cancelled), `43357086470`/`43356756224` (skip + success). If the command text itself should be fixed, that is a contract edit: do it **before** archive and re-review, or leave it and let the next spec's features.json template avoid `$(git rev-parse HEAD)`.
4. **Finding 4 (owner, vault).** Decide where the `gate` vs `ci-gate` name reconciliation lives once #3306 closes — a ci-workflows issue or an ADR-001 amendment — since `proposal.md` explicitly leaves the ADR out of scope and the deviation record disappears with the archive.
5. **Optional, per the spec's own decision note.** `integration`, `test-macos` and `pi-nan-package` are now blocking through `ci-gate`. The stated quarantine policy ("a job that turns flaky is quarantined with a ticket, not left non-blocking") is the right one; the Azure-mirror-outage failure mode named in `proposal.md`'s risks is the first candidate to test that policy against.
6. After archive, move the folder to `specs/archive/CI-017-gate-required-check/` and close #3306 with the PR links (#3387 + #3419) so bitácora flips to Done.
