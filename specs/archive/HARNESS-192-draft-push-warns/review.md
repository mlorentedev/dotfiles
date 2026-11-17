---
spec: "HARNESS-192-draft-push-warns"
verdict: "PASS"
reviewed_sha: "0a05fc0338f3ce8b4b89ac16bceb51ac52221b85"
reviewer: "nan/glm5.3-flash"
date: "2026-11-12"
---

## Adversarial review

**Scope**: HARNESS-192-draft-push-warns (issue `mlorentedev/dotfiles#3364`)
**Sources**: `specs/HARNESS-192-draft-push-warns/{proposal,tasks,verification}.md` + `features.json` ·
`git diff 75cccbd061054bf2dbb9c3bd162cc1cb35b63335...HEAD` (10 files, +429/−6) ·
`git log --first-parent --no-merges 75cccbd0..HEAD` → exactly the spec's four commits
(`1a0d45e9` implementation, `71fc476e` spec folder, `0590276c` the round-1 fix,
`0a05fc03` the round-1 dispositions); no other work arrived through a merge.
Contract digests in `review-request.json` match `dotf spec digest
HARNESS-192-draft-push-warns` for `features.json`, `proposal.md` and `tasks.md` — this
review describes the current contract at `0a05fc03`.

### Spec and task alignment

- **AC1 (draft → `[WARN]` + exit 0, both shells, plus `dotf gate spec`)** — holds.
  `AC9: a draft PR at push time is warned, not refused, and CI keeps the refusal (#3364)`
  runs through `spec_gate_both` (`tests/lib/spec-gate-parity.bash`): zsh leg, bash leg,
  and the real `dotf gate spec` binary on the same env (the helper inherits
  `SDD_PR_DRAFT=true`, and its event deliberately carries no `draft` field). I confirmed
  the parity leg is not decorative: mutating one phrase of the Go warning
  (`still active`→`remains active`), rebuilding, and running that test alone turned it
  red with `script (0) and dotf gate spec (0) disagree`. Mutation reverted; tree clean.
- **AC2 (`false`/unset refused; draft + `skip-archive` without rationale still refused)** —
  holds. `AC9: a PR that is not a draft is refused as before, whatever the flag says` and
  `AC9: a draft does not waive a skip-archive label without its rationale` green. Loosening
  the shell condition from `== "true"` to `-n` turned the first red (mutation run,
  reverted) — the exact-match fail-closed reading is genuinely pinned, not incidentally
  passing. Ordering (skip-archive branch before the draft branch) is asserted by the second.
- **AC3 (adapter hands the gate `SDD_PR_DRAFT`; unset with no PR; `--jq` carries `draft`)** —
  holds now **unconditionally**, which is the delta from round 1. `unset SDD_PR_DRAFT` sits
  as the first executable line after `set -euo pipefail`, before every fall-through path
  (`missing gh`, `missing jq`, detached HEAD, failed lookup, no open PR), and the value is
  recomputed from the resolved PR (`jq -r 'if .draft == true then "true" else "false" end'`),
  so a resolved ready PR outranks a stray `true`. Six named tests pin each path. I verified
  the pin bites: deleting the `unset` line turned **five of the six** stray tests red
  (mutation run, reverted) — the sixth (`does not outrank a resolved PR that is ready`)
  correctly stayed green, because that path is guarded by the recomputation, not the unset.
- **AC4 (CI keeps the refusal)** — holds. `scripts/spec-gate-pr.sh` has zero occurrences of
  `SDD_PR_DRAFT` (grep + named test), no file under `.github/` names it, and
  `AC9: dotf gate spec does not read the event's draft field…` runs the real binary on an
  event with `draft: true` and the variable unset: exit 1 with the `[FAIL]`. The premise
  that a red CI `spec-gate` actually refuses the merge was verified by round 1 against
  `forge/branch-protection.json` (required context, `enforce_admins: true`); whether that
  declared protection is applied live remains **UNVERIFIED** this round (GitHub-side).
- **Round-1 dispositions** (recorded in `verification.md`, applied in `0590276c`/`0a05fc03`)
  — each verified: the Major H-1 fix exists with six named tests and bites under mutation;
  `usage()` now says `draft state`; evidence is repointed to `1a0d45e9`/`0590276c`, both
  ancestors of HEAD (`git merge-base --is-ancestor` on `1a0d45e9` → true); the stdlib
  imports sit in the stdlib group and the pinned `golangci-lint` 2.12.2 (cache cleaned) is
  clean on both `GOOS`. The speculative `--pr-draft` flag was declined with a stated reason;
  the coverage-split question was answered with the named test that holds the property.
- **Freshness of `[x]` ticks**: every implementation box maps to named tests I ran green.
  The unticked closing box "PR opened referencing this spec folder" is **stale, not
  missing**: PR #3380 is open (`gh pr view`, `isDraft: false`) and its body references
  `specs/HARNESS-192-draft-push-warns/`. See finding 1.
- No `[AGENT-DRAFT]` / `[AGENT-SUGGESTION]` tags in the contract spec files (grep).
- Issue #3364 is OPEN and self-assigned; board state matches an in-window spec.

### Evidence I produced in this session

| Command | Result |
|---|---|
| 9 bats files naming the two scripts (`check-knowledge-gate`, `check-spec-gate`, `guard-zsh-bash-rematch`, `spec-gate-adjacency`, `spec-gate-archive`, `spec-gate-pr-real`, `spec-gate-pr`, `spec-gate-prepush-real`, `spec-gate-prepush`) | 1..183, 0 failures (verification.md's 177 + the six tests `0590276c` added, measured on `1a0d45e9`) |
| `cd cli && go test ./internal/cmd/ ./internal/gate/ -count=1` | `ok` / `ok` |
| `go vet ./internal/cmd/` and `GOOS=windows go vet ./internal/cmd/` | clean (rc 0) |
| `golangci-lint run ./internal/cmd/...` (2.12.2, the pin, `go clean -cache` first), both GOOS | 0 issues |
| `shellcheck scripts/check-spec-gate.sh scripts/spec-gate-prepush.sh` | clean |
| mutation: adapter `unset SDD_PR_DRAFT` deleted | 5 stray tests red — reverted |
| mutation: shell condition `== "true"` → `-n` | not-a-draft test red — reverted |
| mutation: Go warning text changed, binary rebuilt | AC9 parity test red (`script and dotf gate spec disagree`) — reverted |
| `dotf spec digest HARNESS-192-draft-push-warns` vs `review-request.json` | match |
| `git status` after all probes | clean apart from spec-folder review files |
| `gh pr view 3380` / `gh issue view 3364` | PR open (not draft, references spec folder); issue open, assigned |

### Findings

| Severity | Reality | Area | Finding | Evidence | Test (named, or UNTESTED) | Fix location (code / tests / spec / vault) |
|----------|---------|------|---------|----------|---------------------------|---------------------------------------------|
| Minor | REAL | contract bookkeeping | `tasks.md`'s closing box "PR opened referencing this spec folder" is unticked while PR #3380 is open and its body references the spec folder — the tick is stale, and `tasks.md` is in the contract set frozen by this review, so ticking it now would invalidate the verdict. | `gh pr view 3380` (open, body names `specs/HARNESS-192-draft-push-warns/`); tasks.md last section | n/a (artifact hygiene) | none in the contract set — record the disposition in `verification.md` (PR #3380 exists and references the folder; the box is lag, not missing work) |
| Question | — | process | PR #3380 is **not a draft**, so the waiver this change delivers does not apply to its own remaining review-window pushes: a post-review fix push to the ready PR is still refused at push time (the exact #3355 loop, one round later). The change is correct and CI's refusal is intentional; the author's own PR just doesn't benefit yet. | `gh pr view 3380` → `isDraft: false`; code read of the gate's condition | n/a | process, not code: convert #3380 to draft while review rounds continue, or use the documented escapes for the specific push |
| Minor | SPECULATIVE | behavior note | On a draft the archive-on-merge check now **continues** into the later checks (dependencies skip, skip-sdd, LOC) instead of ending the run as the FAIL path does — so a draft whose only SDD problem is the archive invariant prints a `[WARN]` and keeps going. Both tiers behave identically (parity green), and a PR that touches a spec is LOC-exempt, so no misbehavior is observed or reachable through a plausible input. | code read (`_check_archive_on_merge` returns 0 → later checks run; `archiveOnMerge()` returns nil → `escapes()` runs); 183-test run green | named — the AC9 parity tests assert exact status and output on both legs for the warning case | none — by design, consistent with a warning rather than a verdict; surface only |

### Evaluator rubric

| Dimension | Grade (A-D) | Rationale (one line) |
|-----------|-------------|----------------------|
| Correctness        | A | All four ACs verified on happy and negative paths, with three mutation probes proving the pins bite; no observed defects |
| Verification       | A | Every AC maps to named tests re-run green this session (183 bats + go test + vet + lint both GOOS) plus reproducible mutations |
| Scope              | A | Diff is exactly the four spec commits + lesson 590; no creep, no unrelated changes |
| Reliability        | A | The round-1 inherited-variable hole is closed at the adapter's entry with one named test per fall-through path, proven red by mutation |
| Maintainability    | B | One warning text exists twice (shell + Go) by documented parity design (CI-018) and is held by a test that provably bites; functions short, lint clean on the pin |
| Handoff-readiness  | A | Lesson 590 written and promoted, round-1 dispositions recorded per finding, promotion lines answered; one stale unticked tasks box (finding 1) |

### Verdict
PASS

### Recommended next steps
- Record in `verification.md` (outside the contract set) the disposition of finding 1: PR #3380 is open and references the spec folder; the unticked `tasks.md` box is bookkeeping lag that cannot be ticked without invalidating this review.
- Before pushing any further fix to PR #3380, decide finding 2: convert the PR to draft for the remaining review rounds, or use a documented escape for that push.
- Finding 3 needs no action; it is recorded for the archive's audit trail.
- `dotf spec archive` is **advisable** in the current state: verdict PASS, no contract change requested, review and evidence fresh at `0a05fc03`.
