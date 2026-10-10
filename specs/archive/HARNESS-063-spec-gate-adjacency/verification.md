---
tags: [spec, verification, templates]
created: "2026-08-09"
---

# Verification - HARNESS-063-spec-gate-adjacency

## Evidence

All output below was produced in the implementing session (2026-08-09) on branch
`feat/spec-gate-issue-adjacency`.

- [x] **Criterion 1** (advisory report names other open issues + the matched file)
      → `tests/spec-gate-adjacency.bats`, cases *"replaying PR 851 against an open
      #849 flags it"* and *"an issue naming no changed file is not reported"*.
- [x] **Criterion 2** (red-tested against #851; unstripped matching pinned)
      → cases *"replaying PR 851…"* and *"matches a path that appears only inside
      an inline code span"*. Mutation-verified, see below.
- [x] **Criterion 3** (advisory; cannot change the verdict)
      → case *"the report is advisory and cannot change the exit status"*, which
      runs the same fixture with and without the feed and asserts equal status.
- [x] **Criterion 4** (offline path byte-identical)
      → case *"with no flag the output is byte-identical to the previous version"*,
      a characterization test (#672) diffing against
      `origin/main:scripts/check-spec-gate.sh` on the same fixture.

## Test status

```console
$ bats tests/spec-gate-adjacency.bats
1..9
ok 1 adjacency: replaying PR 851 against an open #849 flags it
ok 2 adjacency: matches a path that appears only inside an inline code span
ok 3 adjacency: an issue naming no changed file is not reported
ok 4 adjacency: the issues this PR closes are excluded from its own report
ok 5 adjacency: spec-template and doc filenames do not drag in the backlog
ok 6 adjacency: the report is advisory and cannot change the exit status
ok 7 adjacency: a missing feed file degrades to silence, never to an error
ok 8 adjacency: with no flag the output is byte-identical to the previous version
ok 9 adjacency: --help documents the flag

$ shellcheck scripts/check-spec-gate.sh
shellcheck: CLEAN

$ python3 -c "import yaml; yaml.safe_load(open('.github/workflows/spec-gate.yml'))"
spec-gate.yml: valid
```

Red-first order observed: the suite was written before the implementation and run
against the unmodified script, giving 5 failures / 3 vacuous passes (cases 3, 4
and 7 assert *absence*, so they pass trivially while the feature does not exist).

**Effectiveness probe (does the guard actually fire?).** Criterion 2's whole point
is that a future refactor must not reuse `_strip_markdown_code()` here. Asserting
that in a comment proves nothing, so the mutation was applied and measured:

```console
# Mutation: haystack=$(_strip_markdown_code "$haystack") inserted in _adjacent_open_issues
ok 1 adjacency: replaying PR 851 against an open #849 flags it
not ok 2 adjacency: matches a path that appears only inside an inline code span
```

Case 2 goes red, case 1 stays green — because #849's *title* carries the bare
basename unformatted, so the worked example survives stripping while a
body-only citation does not. That asymmetry is exactly why case 2 exists as a
separate case rather than being folded into case 1.

**End-to-end against the live backlog.** The fixtures could not answer whether the
report is *legible*, only whether it matches — so the real workflow command was
run against all 159 open issues and fed to the script over this branch's own diff:

```console
$ gh issue list --state open --limit 500 --json number,title,body \
    --jq '.[] | [.number, ((.title + " " + (.body // "")) | gsub("[\n\r\t]"; " "))] | @tsv' > issues.tsv
159
$ SDD_PR_BODY="Refs #858" ./scripts/check-spec-gate.sh \
    --base-ref origin/main --head-ref HEAD --adjacency-issues issues.tsv
```

The first run reported **37 issues, 3 of them signal**. The other 34 matched
`proposal.md`, `tasks.md` or `docs/lessons.md` — template and doc filenames that
nearly every PR touches. That is a defect: a wall of noise is read once and then
ignored, which is indistinguishable from having no check.

Fixed by matching only files the gate already counts as production
(`_adjacency_candidate_files`, reusing `_excluded` + `_is_active_spec_path`), and
pinned by case 5. After the fix, on the same input:

```console
[INFO] Adjacent open issues (advisory — this does not gate the PR):
         #854    names scripts/check-spec-gate.sh   BUG-061: pre-push spec-gate false-negative …
         #492    names check-spec-gate.sh           CLI-023: vault + spec-gate cutover …
         #251    names scripts/check-spec-gate.sh   COLD-001: specs/archive/ cold-store …
```

37 → 3, all three genuinely relevant to a change in this file — #854 is the very
offline pre-push path criterion 4 exists to protect, and #492 owns the eventual
port. The check demonstrates itself on its own PR.

- No regressions: full suite `bats tests/*.bats`, re-run *after* the precision fix
  (`53b7d04`) → **1101 passed, 1 failed, 71 skipped** of 1102. The pre-fix run is
  not cited: it measured an artifact state that no longer exists, which is this
  spec's own defect class. The single failure is `install-dotf.bats` *"converges over
  a running dotf"*, which fails identically on pristine `main` at `bed3f1f`
  (verified in this session) and is already tracked as **#807** — the busy-binary
  fixture never holds a binary busy. Untouched by this change.

## Decisions made during implementation

- **Network in the workflow, matching in the script.** `check-spec-gate.sh` also
  runs in the pre-push hook (#854), where no token exists, so a `gh` call inside
  it would either break that path or need a second offline branch. The workflow
  fetches one flattened TSV; the script only reads a file. Side effect: the #849
  red-test is a fixture, not an API mock.
- **Advisory, not a gate.** With 159 open issues, basename matching produces false
  adjacencies. A wrong warning costs a glance; a wrong block costs a merge.
- **Matching over unstripped text, not reusing `_strip_markdown_code()`.** The two
  matchers have opposite error policies — see the function comment and the
  mutation probe above.
- **A short explicit deny-list of generic basenames** (`README.md`, `AGENTS.md`,
  `go.mod`, …) rather than a length/shape heuristic. A heuristic would be shorter
  but unexplainable at the moment someone asks why their issue was not reported.
- **Match only production files** — the second helper reuse in this change, and
  the one that got the scrutiny the first one failed. Inheriting `_excluded()`'s
  policy means an issue about a test or a doc will not surface; that is acceptable
  here because an unhandled *input shape* lives in production code by definition,
  and precision is what makes the report legible enough to act on. Recorded as a
  decision rather than left implicit, because the same reuse instinct produced the
  `_strip_markdown_code` trap two hours earlier.
- **Report placed before every gate and early exit**, so a PR that skips SDD or
  fails archive-on-merge still surfaces its adjacent issues.

## Promotion candidates

- [x] Lesson for the repo's `docs/lessons.md`? yes: docs/lessons/lesson-176-reusing-a-helper-inherits-its-error-policy-not-jus.md — reusing a helper inherits
      its *error policy*, not just its code: `_strip_markdown_code` is correct
      where a false positive is expensive and wrong where a false negative is.
- [ ] ADR-worthy decision? **no** — no architectural boundary moved.
- [ ] New pattern candidate for `00_meta/patterns/`? no: not yet — the underlying
      insight is already covered by `pattern-verify-state-before-acting`; revisit
      if the error-policy-inheritance angle recurs in a second project.

## Closing status (2026-10-10)

- AC1-AC4 hold on main: features f1-f4 pass, each command requiring a matching test that passes (a `bats -f` filter that matches nothing exits 0).
- **The fixture-shape inventory did not ship.** This spec deferred it to #857's fix; #857 closed through #862 and #866 without it. It is #858's remaining direction, so the archive of this spec must reference #858, not close it, and the inventory stays tracked there (comment on #858, 2026-10-10).

## Archive checklist

- [x] `proposal.md` frontmatter set to `status: archived`
- [x] Folder moved: `specs/HARNESS-063-spec-gate-adjacency/` -> `specs/archive/HARNESS-063-spec-gate-adjacency/`
- [ ] Bitácora board ticket moved to Done / closed with PR link (ADR-018). Left unticked on purpose: #858 stays open for its second direction, the fixture-shape inventory.

> **Not archived by this PR.** #858 adopted two directions; this PR ships one.
> The fixture-shape inventory travels with #857, and that is the PR that closes
> #858 and archives this spec. This PR's body carries `Refs #858`, so
> `_check_archive_on_merge` correctly leaves the spec active.
>
> **Superseded 2026-10-10.** #857 closed through #862 and #866 without the
> inventory, so no PR is left to carry it. #2304 archives this spec for the
> direction that shipped and still carries only `Refs #858`. The fixture-shape
> inventory stays open on #858 (see "Closing status" above).

## Review round 1 dispositions (2026-10-10)

Round 1 (`nan/qwen3.8-flash`, FAIL) is committed unchanged with this PR. Dispositions:

| # | Finding | Disposition |
|---|---|---|
| 1 | The wiring pin is satisfied by a comment | **Applied.** `tests/spec-gate-pr.bats` scopes the check to the `Run SDD spec-gate` step. It passes on the workflow as it stands and fails when the flag line is deleted. |
| 2 | `tasks.md` and `proposal.md` record the work as undone | **Applied.** Every box is ticked against what shipped, with notes where a pin is weaker than its box, and `status: verifying`. |
| 3 | `--limit 500` truncates the feed silently | **Applied.** The collect step emits `::warning::` when the feed reaches the limit (420 open today). It has a step-scoped bats pin, and the check was exercised at 500 and 420 lines. |
| 4, 5, 7, 8 | Substring noise, untested deny-list, renamed files, report escaping | **Ticketed:** #2303. |
| 6 | The AC4 pin diffs against a `main` that contains the feature | **Ticketed:** #2303 (golden offline output). The criterion holds and is noted on AC4. |
| 9 | The review base is 769 commits behind | **Existing ticket:** #1727, with this instance added. |
| Q | AC1 says "a PR that closes an issue", but it fires on every PR | **Applied.** AC1 now states the shipped trigger. |
| — | `_report_adjacent_issues` at 39 lines and CC ~16 | **Ticketed:** #2303. |

## Review round 2 dispositions (2026-10-10)

Round 2 (`agy/gemini-3.1-pro-high`, PASS WITH GAPS, reviewed `ca4685d2`) is committed unchanged with this PR.

| # | Finding | Disposition |
|---|---|---|
| 1 | `git diff --name-only` quotes paths, so `*"$base"*` misses them | **Ticketed:** #2303, corrected by measurement. Spaces are not quoted, only control characters, `"`, `\` and non-ASCII bytes. The repo has 0 such paths, so this is theoretical. The fix is `-c core.quotePath=false`. |
| 2-4 | Substring noise, step summary escaping, renamed files | **Already ticketed:** #2303. |
