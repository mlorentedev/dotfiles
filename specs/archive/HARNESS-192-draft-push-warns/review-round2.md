---
spec: "HARNESS-192-draft-push-warns"
verdict: "FAIL"
reviewed_sha: "71fc476e632b349747ecb8461a9e64e24d80d79f"
reviewer: "nan/qwen3.8-flash"
date: "2026-11-12"
---

## Adversarial review

**Scope**: HARNESS-192-draft-push-warns (issue `mlorentedev/dotfiles#3364`)
**Sources**: `specs/HARNESS-192-draft-push-warns/{proposal,tasks,verification,features}.md` ·
`git diff 75cccbd061054bf2dbb9c3bd162cc1cb35b63335...HEAD` (10 files, +355/-5) ·
`git log --first-parent --no-merges 75cccbd0..HEAD` → exactly the spec's two commits
(`1a0d45e9` implementation, `71fc476e` spec folder); no other work arrived through a merge.
Contract digests in `review-request.json` match `dotf spec digest HARNESS-192-draft-push-warns`
for `features.json`, `proposal.md`, `tasks.md` — this review describes the current contract.

### Spec and task alignment

- **AC1 (draft → warn + exit 0, both shells, plus `dotf gate spec`)** — holds. Proven by
  `AC9: a draft PR at push time is warned, not refused, and CI keeps the refusal (#3364)`, which
  runs through `spec_gate_both` (`tests/lib/spec-gate-parity.bash`): zsh leg, bash leg, and the real
  `dotf gate spec` binary on the same env. I confirmed the parity leg is not decorative: mutating one
  word of the Go warning (`refuses`→`rejections`… `rejects`) and rebuilding the binary turned that
  test red with `script (0) and dotf gate spec (0) disagree`.
- **AC2 (`false`/unset refused; `skip-archive` without rationale still refused on a draft)** — holds
  for the states the spec names. `AC9: a PR that is not a draft is refused as before` and
  `AC9: a draft does not waive a skip-archive label without its rationale` both green; loosening the
  shell condition from `== "true"` to `-n` turns the first red (mutation run, reverted), so the
  exact-match fail-closed reading is genuinely pinned. Ordering (skip-archive before the draft
  branch) is asserted by the third test.
- **AC3 (adapter hands the gate the flag; `--jq` carries `draft`; **leaves it unset when there is no
  PR to resolve**)** — **holds only under a clean caller environment.** See finding H-1: the
  fall-through paths do not scrub an inherited `SDD_PR_DRAFT`, and the gate then prints a draft
  warning for a PR that was never resolved.
- **AC4 (CI keeps the refusal)** — holds, and better than the tests alone show:
  `scripts/spec-gate-pr.sh` contains zero occurrences of `SDD_PR_DRAFT` (`grep` + the named test),
  `.github/workflows/spec-gate.yml` sets no such env, and `dotf gate spec` on an event carrying
  `draft: true` with the variable unset exits 1 with the `[FAIL]` (named test).
  The premise that a red `spec-gate` actually refuses the merge is backed in git, not assumed:
  `forge/branch-protection.json` lists `spec-gate` as a required status context for
  `mlorentedev/dotfiles` with `enforce_admins: true`, and `spec-gate.yml` triggers on
  `[opened, synchronize, reopened, labeled, unlabeled, edited]` — `ready_for_review` is absent, which
  is exactly the stale-green the proposal's "Out of scope" refuses to rely on. Whether that declared
  protection is applied live is GitHub-side: **UNVERIFIED** (drift detection exists as
  `dotf forge protection check` / `cli/internal/doctor/checks_branch_protection.go`).
- **Freshness of `[x]` ticks**: every implementation box in `tasks.md` maps to a named test I ran
  green. The unticked box "PR opened referencing this spec folder" is consistent with an
  about-to-merge PR; I did not query GitHub (out of scope for this run).
- No `[AGENT-DRAFT]` / `[AGENT-SUGGESTION]` tags remain in the spec files (grep).

### Evidence I produced in this session

| Command | Result |
|---|---|
| `bats tests/spec-gate-archive.bats tests/spec-gate-prepush.bats` | 1..58, 0 failures |
| `bats tests/check-spec-gate.bats tests/spec-gate-adjacency.bats tests/spec-gate-pr.bats tests/spec-gate-pr-real.bats tests/spec-gate-prepush-real.bats tests/guard-zsh-bash-rematch.bats tests/check-knowledge-gate.bats tests/spec-gate-status.bats` | 1..122, 0 failures (180 total, no regressions) |
| `cd cli && go test ./internal/cmd/ ./internal/gate/ -count=1` | `ok` / `ok` |
| `go vet ./internal/cmd/` and `GOOS=windows go vet ./internal/cmd/` | clean (rc 0) |
| `shellcheck scripts/check-spec-gate.sh scripts/spec-gate-prepush.sh scripts/spec-gate-pr.sh` | clean (rc 0) |
| mutations: Go warning text / shell `-n` condition / adapter `--jq` losing `draft` | two red, one red (see H-2) — all reverted, `git status` clean apart from spec-folder files |
| stray-`SDD_PR_DRAFT` repro (gate + adapter, bash and zsh, in `mktemp -d`) | reproduces H-1 |
| `dotf spec digest HARNESS-192-draft-push-warns` | matches `review-request.json` digests |
| `golangci-lint run` (the pin, both GOOS) | **UNVERIFIED — not re-run this round**; the lint claim in `verification.md` rests on the author's output |

### Findings

| Severity | Reality | Area | Finding | Evidence | Test (named, or UNTESTED) | Fix location |
|---|---|---|---|---|---|---|
| **Major** | **REAL** | trust boundary / false output (AC3) | The pre-push adapter does not scrub an inherited `SDD_PR_DRAFT` on its fall-through paths, so a stray `SDD_PR_DRAFT=true` in the developer's environment (in the dotfiles repo, a plausible `~/.zshenv`/profile vector) turns the local archive-on-merge **refusal into a warning that exits 0** on pushes where **no PR was resolved at all** — and the text then asserts "this draft PR closes an issue whose spec is still active" about a PR that does not exist, with advice ("Before marking it ready…") aimed at the wrong moment. AC3's observable outcome ("leaves it unset when there is no PR to resolve") is therefore conditional on the caller's environment rather than guaranteed by the adapter. | Reproduced: with `PATH` holding only bash/dirname/cat and `env -i SDD_PR_DRAFT=true`, `spec-gate-prepush.sh` hands the gate `SDD_PR_DRAFT=[true]` (control without it: `[<unset>]`); running `check-spec-gate.sh` with `SDD_PR_DRAFT=true`, no `SDD_PR_*` and a commit message `Closes #123` against an active spec exits **0** with the `[WARN]` under **both** bash and zsh, while the same run without the variable exits 1 with `[FAIL]` | **UNTESTED** — `spec-gate-prepush: a failed lookup falls through…` and `…missing gh falls through…` assert `[<unset>]` only from a clean env; no case exports a stray value | code + tests: `unset SDD_PR_DRAFT` at the adapter's entry (or `exec env -u SDD_PR_DRAFT` in `_run_gate`'s no-PR paths), plus named cases per fall-through path (missing gh, missing jq, failed lookup, empty answer, no branch) asserting the gate sees `[<unset>]` |
| Minor | REAL | doc drift | `spec-gate-prepush.sh`'s `usage()` still says it "Resolves the labels/body/author of the open PR"; it now also resolves `draft`. The file header was updated, the `--help` text was not. | Read of `scripts/spec-gate-prepush.sh` lines 43-47 vs the new `draft` key at line 82; test `spec-gate-prepush: --help exits 0 and documents the fall-through` checks only the fall-through sentence | UNTESTED (help text) | code (non-contract) |
| Minor | REAL | evidence provenance | `verification.md` cites `ef7080ed` for all four ACs; that commit is **not an ancestor of HEAD** (`git merge-base --is-ancestor ef7080ed HEAD` → false). It is the pre-amend copy of `1a0d45e9`, which `features.json` cites correctly. After a squash-merge `ef7080ed` is unreachable, so the evidence cannot be checked out by a later reader. | `git cat-file -t ef7080ed` → commit, `git log --oneline -1 ef7080ed` shows the same subject as `1a0d45e9`; not in the review range | n/a (artifact hygiene) | spec artifact `verification.md` — outside the contract set, so fixable without invalidating a verdict |
| Minor | THEORETICAL | maintainability | Stdlib imports `"io"`, `"os"` were appended to the third-party group in `cli/internal/cmd/gate_spec_rules.go` instead of the repo's stdlib-first group; `go vet` is silent, an import-order linter would not be — and I did not re-run the pinned `golangci-lint` to confirm the "0 issues" claim. | diff hunk at `gate_spec_rules.go:1-10` | n/a | code (non-contract); re-run `golangci-lint run ./internal/cmd/...` both GOOS |
| Minor | SPECULATIVE | design | The waiver is an untyped env var read by both tiers. A `--pr-draft` flag on `dotf gate spec` / `check-spec-gate.sh`, passed explicitly by the pre-push adapter, could not be inherited by accident and would make H-1 unreachable by construction; the CI adapter would simply never pass it. Surface only — do not gate on it. | code read; no repro needed | n/a | — (author's call; if declined, say so in `verification.md`) |
| Question | — | coverage split | `spec-gate-prepush: a draft PR hands the gate SDD_PR_DRAFT=true` stayed **green** when `draft` was deleted from the `--jq` projection, because the stub `gh` returns the already-adapted shape and never applies the filter. The property is still caught — `spec-gate-prepush: the --jq expression picks the first PR, maps REST to the shape the adapter reads…` went red — so no coverage is missing, but a single mutation cannot tell which half of REST→env broke. Accept as-is or note it. | mutation run (reverted): `-f 'draft'` → ok 1; `-f 'the --jq expression'` → not ok | named (test 52) | — |

**Honest counterweight on H-1's severity.** It is Major, not Blocker: the merge-time invariant is not
weakened (CI never receives the variable, and `spec-gate` is a required context per
`forge/branch-protection.json`), and the local tier was already bypassable with `--no-verify`. What
makes it a REAL Major rather than a nit is that it contradicts a stated acceptance criterion's
observable outcome, produces a factually false user-facing claim about a PR that was never resolved,
and sits in the exact pre-PR window where specs historically do not get archived — the very failure
class SDD-038 exists to close.

### Evaluator rubric

| Dimension | Grade (A-D) | Rationale (one line) |
|---|---|---|
| Correctness | B | AC1/AC2/AC4 verified with negative paths; AC3's "leaves it unset" outcome fails under an inherited env var (H-1) |
| Verification | B | Every AC maps to named tests I re-ran green (180 tests, 0 failures) plus three mutation probes; one orphaned commit citation and one untested path |
| Scope | A | Diff is exactly the two spec commits + lesson 590; no creep, no unrelated changes |
| Reliability | C | Fail-closed on unset/non-`true`/lookup failure, but every no-PR fall-through path inherits the waiver and then prints a draft claim it cannot support |
| Maintainability | B | Short functions, one duplicated warning text held by a parity test that provably bites; import-group smell |
| Handoff-readiness | B | Lesson 590 written and promoted, promotion lines answered, tasks ticked with evidence; "PR opened" still unticked, evidence sha orphaned by the amend |

### Verdict

**FAIL** — one **REAL Major** (H-1) that is **UNTESTED**. Rubric otherwise all B except Reliability
(C), which alone would only demand PASS WITH GAPS; the severity path governs.

### Recommended next steps

Contract set (`proposal.md`, `tasks.md`, `features.json`) is **not** what needs changing — the fix is
code + tests, so the digests above survive it. Minimum set to flip this to PASS:

1. **H-1, code** — make `scripts/spec-gate-prepush.sh` own the variable unconditionally: `unset
   SDD_PR_DRAFT` at entry (so an inherited value cannot survive), or `exec env -u SDD_PR_DRAFT` in the
   `_run_gate` calls that run with no resolved PR. Keep the `"true"`-only comparison in both tiers.
2. **H-1, tests** — named bats cases in `tests/spec-gate-prepush.bats` that `export SDD_PR_DRAFT=true`
   and assert the stub gate logs `SDD_PR_DRAFT=[<unset>]` for each fall-through path: missing `gh`,
   missing `jq`, `gh` exit ≠ 0, empty answer (no open PR), no branch. One of these must fail before
   step 1 lands (red-first), per the repo's TDD standing order.
3. **Re-run and paste** — `bats tests/spec-gate-archive.bats tests/spec-gate-prepush.bats`,
   `shellcheck scripts/check-spec-gate.sh scripts/spec-gate-prepush.sh`, and the pinned
   `golangci-lint run ./internal/cmd/...` on both GOOS (the claim I could not verify).
4. **Minor, code** — update `spec-gate-prepush.sh`'s `usage()` to name `draft`; move `"io"`/`"os"`
   into the stdlib import group.
5. **Minor, `verification.md`** (non-contract) — repoint the AC evidence from `ef7080ed` to the
   in-range `1a0d45e9`, and disposition H-1/H-2/this round's Minors there (applied / ticketed /
   declined with a reason) rather than in chat.
6. Re-review after 1–2; the range and digests are unchanged, so the next round can be focused.

Is `dotf spec archive HARNESS-192-draft-push-warns` advisable now? **No** — it would archive on a FAIL
review; the archive gate should refuse it.
