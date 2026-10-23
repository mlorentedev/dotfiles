---
tags: [spec, verification, templates]
created: "2026-08-18"
---

# Verification - TOOL-017-deterministic-review-triage

## Evidence

Map every acceptance criterion from `proposal.md` to concrete proof (commit hash, test name, or observed behavior).

- [x] Criterion 1 -> superseded by `dotf pr triage-queue` (#1057, #1101); owner accepted it on #1062, 2026-10-10
- [x] Criterion 2 -> `cli/internal/prtriage/prtriage.go` reads `harness/review-attestation.json`; test `TestTheRealRegistryServesBothConsumers`
- [ ] Criterion 3 -> declaration and reader landed (`Classify`, tests `TestClassify`, `TestTheRealRegistryDeclaresTheClassificationMarker`, bats `pr-agent: the reviewer is asked for every class marker the registry declares`); the consumer landed with tests `TestUnclassifiedFindings`, `TestFetchReportsUnclassifiedInlineFindings`, `TestFetchReadsNoInlineFindingsWithoutADeclaration`, `TestFetchMarksUnreadInlineFindings`, `TestPrTriageQueue_NamesUnclassifiedFindings`; it reports only once the registry declares `classification.reviewers`

## Test status

- Test suite: `cd cli && go test ./...` -> exit 0 (2026-10-22, #2926 branch); `golangci-lint run` -> 0 issues; `GOOS=windows go vet ./...` -> ok
- Manual smoke test: the #1059 replay rerun at `1719024f` with `--defect-when-passes` (see `tasks.md`); the AC4 gate run on #2924's own triage record -> `[OK]`, exit 0
- No regressions in existing test suite: yes

## Decisions made during implementation

Brief log of non-obvious trade-offs or course corrections taken during the work. Routine choices belong in commit messages, not here.

- 2026-10-20, re-baseline: the spec was resumed rather than abandoned (owner). AC1 and AC2 were met outside the spec folder and are ticked with their evidence; AC7 is partly met by `dotf pr land` at PR granularity.
- The class marker is declared in `harness/review-attestation.json` (`classification`), not in Go, so a class change is a registry edit (AC2's rule applied to AC3). A bats contract test ties `.pr_agent.toml` to it.
- Measured before choosing the marker (2026-10-20): PR-Agent wrote the gradation as `**Header (REAL)**` (#2833), as the body's first word `REAL.` (#2832), or not at all (#2839). Hence a whole line `Class: <CLASS>`, matched exactly; two lines that disagree are a conflict and stay unclassified.
- 2026-10-21, consumer: an unclassified finding is REPORTED by `dotf pr triage-queue` and does not make a PR pending. Making it pending would change the land gate before the marker has a measured adoption window (the proposal's escalation risk). Live: the first review after #2845 (#2849, claude-haiku) emitted the marker on 2 of 2 inline findings.
- The findings are read from `pulls/N/comments`: PR-Agent's COMMENTED review has an empty body, so the reviews endpoint the verdict reads carries no finding text (lesson 448, applied to the classifier). They are read only for a PR the queue reports, and only when the registry names a classifying reviewer, so the session-start probe pays nothing extra by default.
- Which reviewers are asked for the marker is declared in the registry (`classification.reviewers`), not inferred: CodeRabbit's inline findings carry its own severity format and would all read as unclassified.
- 2026-10-21, owner decisions on #1062: AC4-AC6 option A (a reproduction is a committed failing test or a recorded non-zero exit, never a model's judgement); AC7 option B (the PR-level gate stands). The registry now names `github-actions` as the one classifying reviewer.
- 2026-10-21, AC5 (`dotf pr reproduce`): a check that never ran as designed (start failure, 126/127, a signal, a timeout) is `could-not-run`, a state of its own (owner), because its non-zero exit is not evidence of a defect. The command exits 1 for a row without a verdict (`could-not-run`, `did-not-land`), which keeps main.go's one exit contract.
- `--probe` is a mutation, not an uncommitted test: under option A a test that names the defect is committed and run without a probe. The patch lives outside the tree, since an untracked file is a dirty tree and is refused.
- The probe is applied and its targets read from the repository root, because a patch names files from there. The first version read cwd-relative paths and reported a probe that had landed from `cli/` as `did-not-land` (lesson 512).
- Landing and restoring compare each target's git blob id (`git hash-object`), not its raw bytes. On the Windows runner (autocrlf) a file git called clean was LF on disk while every git write rendered it CRLF, so a mode-only probe failed to revert by byte comparison although its content never changed (lesson 512, second instance). Targets are restored by checking them out from HEAD, which the clean-tree check makes exactly how they were found. A first fix that only changed the restore, on the hypothesis that `git apply -R` was at fault, failed the same way on the same runner.
- `WaitDelay` bounds `Wait` after a timeout kill, because a child holding the output pipe outlives the kill.
- The command applies nothing and decides nothing: a human pastes its row into `## Review triage`. Gating `apply` on it is AC4, the next slice.
- 2026-10-22, AC4 and AC6 (owner: the gate refuses, option A over report-only). The check reads the existing human-written `## Review triage` table rather than a new format: a finding's row is the first naming its `path:line` (bounded, so `a.go:4` is not `a.go:40`; the base name will do), and its disposition the first later cell opening with apply, defer, refuted or skip, so a disposition word in the item cell is not read. `apply` needs `Reproduction: reproduced`, `refuted` needs `Reproduction: did-not-reproduce`, `skip` of a REAL finding is refused, `defer` passes. Only the newest triage record with standing (#2870) counts, and only for findings it could have seen.
- The gate lives in the queue, not in `land`: a held PR is pending, so `land` refuses it through the condition it already checks, and `triage-queue` names the finding and what its row lacks. A triaged PR now costs one inline read when the registry declares a classifying reviewer; the read is shared with the AC3 report.
- 2026-10-22, #2926: a check is inverted by the harness (`--defect-when-passes`), never by the shell. `sh -c '! check'` maps 127 to 0, so a check that never ran refuted the finding it was meant to test (lesson 518). The inversion is applied after the could-not-run cases are classified, so those stay `could-not-run` in both directions.
- The row names the direction (`defect when the check passes`), because the same exit code now means opposite things on two rows and a reader cannot tell them apart otherwise.
- The row quotes its command and escapes `|`. Dogfooding the gate on #2924 produced two rows whose quoted pattern lost its quotes, so the recorded command was not the one that ran; a pipe inside the row also splits the triage table's cell. The gate still reads such a row, since the disposition is the first cell after the naming one.
- `classify` lost its `started` parameter: a mutation removing `!started` survived, because a check that never started keeps exit -1 and `exit < 0` already catches it. Deleted rather than kept as documentation, per the mutation discipline.

## Promotion candidates

Before archiving, flag what (if anything) should be promoted to the vault. If all three are "no", archive in repo is the only persistence.

- [x] Lesson for the repo's `docs/lessons.md`? yes, written: lesson 512 (a landing check reads paths and content in the frame the patch wrote them), from AC5
- [x] ADR-worthy decision for the repo's `docs/adr/adr-XXX.md`? no: the decisions are the owner's rulings on #1062 (option A, the gate refuses, PR-level guard), recorded in this log and on the issue; none outlives the spec as an architectural choice
- [ ] New pattern candidate for `00_meta/patterns/`? Only if this recurs in >1 project. candidate, not yet: "a reviewer's class is a hypothesis; reproduce before apply" would apply to kubelab's PR-Agent too, but it has run in one repo only; decide at archive

## Archive checklist

- [ ] `proposal.md` frontmatter set to `status: archived`
- [ ] Folder moved: `specs/TOOL-017-deterministic-review-triage/` -> `specs/archive/TOOL-017-deterministic-review-triage/`
- [ ] Bitácora board ticket for this spec moved to Done / closed with PR link (ADR-018)
- [ ] Promotions above executed (if any)
