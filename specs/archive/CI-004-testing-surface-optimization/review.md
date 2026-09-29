---
spec: "CI-004-testing-surface-optimization"
verdict: "PASS-WITH-GAPS"
reviewed_sha: "302ac3ad346a87687391c8e8ce603a1af5f16a50"
reviewer: "nan/mimo-v2.5"
date: "2026-09-28"
---

## Adversarial review

**Scope**: CI-004-testing-surface-optimization (issue #1739), rows P0.1, P0.2, P0.3, P0.5, P0.6, P0.7, P1.3 (declined), P1.5 (declined) — round 2, the whole change `88a6591cf94319df7b2d0e378abeacdd3a3c712b...HEAD`, not the delta since round 1
**Sources**: `specs/CI-004-testing-surface-optimization/{proposal,tasks,verification,features}.md`, `specs/CI-004-testing-surface-optimization/review.md` (round 1, `8a88d46`), `git diff 88a6591cf94319df7b2d0e378abeacdd3a3c712b...HEAD`, live GitHub API (job logs, runs), GitHub's own documentation

### Spec and task alignment

Freshness first: I recomputed the launcher's normalised contract digests from disk (checkbox state and
frontmatter `status:` folded, `features.json` re-encoded without `state`/`evidence`, per
`cli/internal/spec/contract_digest.go`) and all three match `review-request.json` —
`proposal.md` `3472634889…`, `tasks.md` `67902500…`, `features.json` `ee7cc778…`. The contract set has
not moved since this round was launched, so nothing below is stale against its own request.

Round 1 (`nan/qwen3.8-flash`, PASS-WITH-GAPS on `8a88d46`) left one Major and five Minors. Every one
now carries a disposition in `verification.md` ("Review dispositions"), and the two that asked for
changes were applied in `302ac3a` (the only CI-004 commit since round 1): `main()` split into
`collect_reporters()` + `main()`, and AC8's evidence restated as a re-runnable per-step log count.
I verified both on the code, not on the claim: `mccabe` reports every function at CC ≤ 8 (`main` 8,
`collect_reporters` 6, `display_names` 5, `gate_gaps` 6) and `main` is 25 lines; the AC8 log count
re-ran live (below). Round 1's Major **decline** I checked against GitHub's documentation rather than
accepting: docs.github.com states verbatim *"A job that is skipped will report its status as
'Success'. It will not prevent a pull request from merging, even if it is a required check"* — so the
round-1 premise ("a required check from a job-level `if:` stays pending forever") is wrong and the
decline holds. What survives of that finding is narrower and recorded as Minor 1 below.

Every acceptance criterion was re-proved with a command this round:

- **AC1** — `bats tests/guard-no-gui.bats tests/vault-health.bats`: 12/12 ok in **2168 ms** (features f1
  bound 5000 ms, baseline 30.9 s). Mutation: delete `3>&-` from `_launch_fake` → `not ok 9 guard: a
  fake GUI launched by this file does not hold bats' fd 3`, i.e. the leak and its detector both stay
  live; reverted, tree clean.
- **AC2/AC3** — `bats tests/skills-pipeline.bats tests/guard-lesson-numbers-unique.bats`: exit 0,
  30 ok in 42 s, including `ok 25 a write into a shared setup_file home fails` and `ok 26 skills-pipeline
  deploys at most 6 times per run` (the budget test is the file's last test, so it counts every deploy
  in the run). `chmod -R a-w` on both shared homes + `teardown_file` restore present in `setup_file`.
- **AC4** — features f3 exits 0: `test-windows` → `timeout-minutes: 20`; loan comment replaced by the
  distribution and the "raise only with a run that crossed it and a ticket" rule.
- **AC5** — mutation: pattern reverted to `*--user-data-dir=*bats-run*` in `setup_suite.bash` → `not ok 11
  guard: the stray detector ignores a test-shaped process from another bats run`; reverted, tree clean.
  Both halves green at HEAD (f4 exit 0), including the empty-`BATS_RUN_TMPDIR` guard.
- **AC6** — f5 exit 0 (`bats --jobs … --no-parallelize-within-files` in `ci.yml`), with the
  `command -v parallel` precondition. The CI step times (96/95/97/97/97 s vs 99.5 s bar, vs 172/165 s
  adjacent serial) were re-fetched and verified by round 1 and are unchanged in `verification.md`.
  The local-figures provenance note round 1 asked for is now written in the dispositions (and I confirm
  this box has no `parallel` binary, so `bats --jobs` cannot reproduce them here).
- **AC7** — `bats tests/workflow-job-names.bats tests/guard-lesson-numbers-unique.bats`: 7/7 ok.
  Three mutations, each red: drop `release-snapshot` from `cli-gate`'s inline `needs` → `not ok 1`;
  rename `cli.yml`'s lint job back to `lint` → `not ok 1 workflow job display names are unique across
  workflows`; (round 1 additionally proved the `paths:`-filter and needs-exemption doors). I also
  walked the live config: of the seven required contexts, `cli-gate` is `if: always()` and
  `lint`/`lint-powershell`/`test`/`test-windows`/`spec-gate` carry no job-level `if:` at all — no live
  required check sits behind a condition.
- **AC8** — f7 exit 0 (`GH_TOKEN` on the doctor-gate step, `issues: read` on the job). Round 1 could not
  fetch the logs; this round could: jobs `109223397978` and `109231917127` each return **43**
  `set the GH_TOKEN environment variable` lines, **all** inside `.\setup-windows.ps1` (which keeps no
  token by design) and **0** in `Run … doctor-gate.ps1` — a step that is present in the log. The
  round-1 disposition's claim reproduced exactly.
- **AC9/AC10** — both declines stand: f8 and f9 exit 0 (verification numbers present, no `cache-from`
  in the build step); no buildx/build-push action exists in the range; the layer split (48 s of
  per-commit work) and the critical-path numbers (integration ends 103–164 s in, test-windows ends
  every run) support declining rather than shipping.
- **Suite** — `bats tests/*.bats`: **1696 tests, 1695 ok**, the single failure
  `the vendored oh-my-zsh git-plugin snapshot is still fresh against the real install` (#1641,
  environmental, known-failing serially too). `shellcheck -S warning` on the edited bats/bash files: rc 0.
  `actionlint` on `ci.yml` + `cli.yml`: rc 0.
- **Features / hygiene** — f1–f9 all exit 0 as written. All nine entries are `state: "pending"` with
  empty `evidence`, correct for an agent-written file. No `[AGENT-DRAFT]`/`[AGENT-SUGGESTION]` tag in the
  spec's prose. Lessons 309 and 318 exist and are indexed (`docs/lessons/_index.md` rows verified by the
  lesson guards above). #1739 is OPEN and assigned. Working tree clean apart from this file and the
  launcher's `review-request.json`.

### Findings

| Severity | Reality | Area | Finding | Evidence | Test (named, or UNTESTED) | Fix location (code / tests / spec / vault) |
|----------|---------|------|---------|----------|---------------------------|---------------------------------------------|
| Minor | THEORETICAL | AC7 guard (residual of round 1's Major) | The decline of round 1's Major is correct on the mechanism (GitHub docs: a skipped job reports Success and does not block a required check), but two things survive it. (a) The checker's and the proposal's rationale — "a required check … would stay pending" — is the *wrong* reason for a non-matrix job-level `if:`; the real door is the opposite: a required check that reports `skipped` is **satisfied without ever running** (a vacuous pass). (b) The guard therefore neither detects nor documents that door. No live required context trips it (verified above: only `cli-gate`, under `always()`), and `cli-gate` itself is the correct pattern, so this is rationale-plus-coverage, not a live bypass. | GitHub docs quoted above; `tests/lib/check-workflow-contexts.py` error text "it would stay pending"; round-1 fixture (exit 0) reproduced the gap; live config walk (7 required contexts) shows no conditional reporter | UNTESTED — `tests/workflow-job-names.bats` has no `gate_fixture` case for "required context held by a non-matrix job with a job-level `if:`"; round 1's fixture showed the checker passing it | tests + code (correct the error text/docstring; add the fixture) — outside the contract set, post-verdict |
| Minor | THEORETICAL | AC7 guard (API-posted statuses) | `collect_reporters` treats an API-posted status (`-f context=…`, i.e. `review-attestation`) as always-reporting whenever the *workflow* has no `paths:` filter — it never reads the posting job's `if:`. `attestation`'s `if` includes `github.event_name == 'pull_request'`, so today the required context always posts; if that condition ever narrowed, `review-attestation` would never appear on some PR, the required check would sit pending and block every merge, and the guard would stay green (silent-guard gap, availability rather than bypass). | `tests/lib/check-workflow-contexts.py` `collect_reporters()` line `reporters.setdefault(ctx, []).append((…, not filtered))`; `review-attestation.yml:227` + job `if` | UNTESTED — no fixture covers a status-posting job with a job-level `if:` | tests (extend the status branch) — outside the contract set |
| Minor | REAL | repo health (out of this change) | Main's `integration` job failed on the two most recent `ci.yml` pushes (runs `36516883213`, `36518108046`): `Error: failed to install: opencode, copilot, bw, yarn` during the image build, with the in-container doctor then reporting its usual FAILs. Observed live this round, not inferred. It is **not** CI-004's: none of the nine CI-004 commits touches `tests/Dockerfile.integration` or `setup-linux.sh`, and the only `versions.conf` change in the range is release-please's `DOTF_VERSION` bump; `integration` is also not a required context, so protection is unaffected. Filed here because the Debt rule does not let a noticed defect pass unremarked. | `gh api …/actions/jobs/109246233147/logs` (failing build lines); `git show --name-only` over the nine commits | n/a (pipeline observation) | ticket outside this spec — recommend a bitácora issue if one does not already cover installer flakiness; record the disposition in `verification.md` |
| Minor | THEORETICAL | scope / review base | The mandated range still contains ~140 files of other specs' merged work (CLI-036's age-blob deletion, SEC-006, AI-046, #1794–#1817, the merge of `main` at `bf638e0`), so the diff is not "the spec's change" alone. Round 1 raised the same question; #1551/#1727 track it. I graded CI-004's nine commits inside the range and treated the rest as other changes carrying their own archived reviews. | `git diff --stat 88a6591…HEAD` (147 files) vs the nine-commit file list above | n/a | launcher/process (tracked #1551/#1727), not this change |
| Question | — | contract hygiene | `tasks.md`'s last box ("Independent review through `dotf spec review …`") is unticked; this verdict is that review. Ticking it is safe — `contract_digest.go` folds checkbox state into the digest, so a tick cannot stale this review (I verified the normalisation reproduces the launcher's digests exactly). | digest recomputation; `contract_digest_test.go` "ticking checkboxes changed the contract digest" fails by design | n/a | spec (`tasks.md`) — safe to tick after this file lands |

### Evaluator rubric

| Dimension | Grade (A-D) | Rationale (one line) |
|-----------|-------------|----------------------|
| Correctness | B | All ten criteria reproduced here by command or mutation (AC1–AC8) or declined with the numbers the criterion offered (AC9/AC10); the only open correctness notes are guard-coverage rationale gaps with no live config tripping them. |
| Verification | B | f1–f9 exit 0, four mutations red-then-reverted, full suite 1695/1696 with the known #1641, AC8's log count re-fetched live (round 1 could not); AC6's CI step times rest on round 1's fetch, unchanged since. |
| Scope | B | CI-004's own commits stay inside the proposal's rows and touch no setup twin; the mandated base drags other specs' merges into the range (documented, tracked); one disclosed side-change (versions-step grouping) clears a pre-existing actionlint SC2129. |
| Reliability | B | Fails closed where it matters: `parallel` precondition, `cli-gate` red on any non-success/skipped result and `needs` covering every PR job, run-scoped stray detector with an empty-prefix guard, read-only shared homes, step-scoped `GH_TOKEN`. Residual holes are in the guard's coverage, not in the gates. |
| Maintainability | B | `mccabe` CC ≤ 8 everywhere, `main()` 25 lines (round 1's 41-line finding fixed), helpers small and commented with the incident behind each rule; suite green except the environmental #1641. |
| Handoff-readiness | A | Every round-1 finding dispositioned in `verification.md` with measurements, declines recorded with numbers, lessons 309 and 318 written and indexed, #1739 open and assigned, only the review box unticked. |

### Verdict

**PASS WITH GAPS**

No Blocker, no REAL Major inside this change. The one REAL finding (main's `integration` red) is
observed live but is outside CI-004's diff and outside its required contexts. Both open guard findings
are THEORETICAL and untripped by live configuration; round 1's Major was declined for a reason I
checked against GitHub's own documentation and it holds. Rubric is B or above in all six dimensions.
Per `severity × reality`, the gaps are tracked, not blocking.

`dotf spec archive` is **advisable** once the dispositions below are recorded. The contract set
(`proposal.md`, `tasks.md`, `features.json`) is closed on this verdict — do not edit it, or the review
goes stale; `verification.md`, code and tests remain open.

### Recommended next steps

All outside the contract set — for the implementer to disposition in `verification.md` (applied,
ticketed, or declined with a reason), or to carry into a follow-up ticket:

1. **Guard rationale + fixture (Minor 1):** correct the "it would stay pending" text in
   `check-workflow-contexts.py` and in the AC7 prose *quotation only* (not `proposal.md`) to GitHub's
   actual semantics (skipped → Success → satisfied-without-running), and add a `gate_fixture` case for
   a required context held by a non-matrix job with a job-level `if:` so the door is asserted rather
   than argued.
2. **API-status `if:` blind spot (Minor 2):** teach `collect_reporters` to read the posting job's `if:`
   for `-f context=` statuses, with a fixture where a narrowed condition leaves `review-attestation`
   unposted.
3. **The red `integration` job (Minor 3):** check for an existing issue on the installer flakiness
   (`opencode, copilot, bw, yarn` failing inside the image build on two consecutive main pushes); if
   none exists, file one with the two run ids — the Debt rule does not let it pass unremarked.
4. **Tick the review box** in `tasks.md` (safe: checkbox state is folded out of the contract digest),
   then run the harness over `features.json` so the nine `pending` entries get terminal states with
   evidence, and archive with this `review.md` in place at `302ac3ad346a87687391c8e8ce603a1af5f16a50`.
5. Carry round 1's still-open SPECULATIVE note (the deploy counter matches only a standalone `--deploy`
   token) into CI-006/CI-008 rather than this change; it was declined once and nothing in this round
   raises its evidence.
