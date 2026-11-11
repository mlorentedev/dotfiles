---
spec: "HARNESS-190-lane-doctor"
verdict: "PASS"
reviewed_sha: "b6d74f792889c69d6c92856d393e76507e5b24fa"
reviewer: "nan/glm5.3-flash"
date: "2026-11-11"
---

## Adversarial review

**Scope**: HARNESS-190-lane-doctor / `git diff 4e0281f69a14fa801a76482cd84927473bc589cd...HEAD` (13 spec commits per launcher; `git log --first-parent` in the range contains exactly those 13 — no other work rode in).
**Sources**: `specs/HARNESS-190-lane-doctor/{proposal,tasks,verification,features.json}`, prior rounds `review-round1.md` / `review-round2.md`, `cli/internal/lane/{audit,identity,named}.go` + tests, `cli/internal/cmd/lane_doctor.go` + test, `docs/runbooks/guide-persistent-named-lanes.md`, `docs/adr/adr-061-…`, `docs/lessons/lesson-582-…`.

### Spec and task alignment

- All four ACs in `proposal.md` map to named tests and `features.json` f1–f4; every `tasks.md` implementation box has diff evidence. Only "PR opened referencing this spec folder" is unticked — correct at review time.
- Contract digests in `review-request.json` match `dotf spec digest HARNESS-190-lane-doctor` on all three contract files — the review is not stale against the contract.
- The two prior rounds' fixes resolve their findings: `assertWorktreeOf` is now consulted even when the single record sits at the canonical path (audit.go `auditLane`, the round-1 "wiped and recreated" case, asserted against a misplaced "lease" finding in `TestAuditLanes_ReportsABrokenLane/a plain directory at a lane path git still registers`); round-2's foreign-worktree contract is implemented via `errNotAWorktree` / `strangerFix` and pinned by `TestAuditLanes_LeavesAnotherRepositorysWorktreeAlone` with the forbidden-words assertion (no "prune"/"repair"/"move the directory" against a foreign worktree) plus a proof the named repair works end-to-end.
- The three shapes round 2 found untested (`empty entry`, lease state, dual-record) now have named cases in `TestAuditLanes_ReportsACorruptedMarkerByField` and `TestAuditLanes_ReportsABrokenLane`.
- The mv/`git worktree move` registry claim (why the audit enumerates `<common dir>/worktrees/*/` instead of `git worktree list`) is recorded as lesson-582 and in the runbook — matches the proposal's prose.
- "No auto-repair; every FIX names its command" holds: no code path mutates anything; `Audit` carries strings only.

### Findings

| Severity | Reality | Area | Finding | Evidence | Test (named, or UNTESTED) | Fix location (code / tests / spec / vault) |
|----------|---------|------|---------|----------|---------------------------|---------------------------------------------|
| Minor | THEORETICAL | `laneRecords` | An admin dir whose identity record reads fine but whose `gitdir` file is missing/unreadable is silently skipped (`!ok || gerr != nil` → `continue`), not reported. A lane in that state is then diagnosed with a message that misnames the problem — "registered at a path that no longer exists" (bogus path) or "carries no identity record" — instead of the corrupt gitdir. Argued from the code; never observed, no repro attempt. | code read of `audit.go:laneRecords` | UNTESTED | code + tests (report an unreadable `gitdir` the way an unreadable record is) |
| Minor | SPECULATIVE | exit contract | AC4 specifies exit 0/1 but not the audit-failure path; the implementation adds exit 2 (stderr message). Documented in the code comment, consistent with "non-zero when there is one"; only the spec is silent about it. | code read of `lane_doctor.go:runLaneDoctor` = UNTESTED | specs (post-archive note; contract set is closed) |

Non-findings checked and dismissed: the set-difference rule in `parseMarker` only runs when name validation produced no findings — correct, since a marker finding stops the audit by design; `anyLaneExists` includes git-recorded lanes so an occupied-lane-path clone is audited, not waved off with the fresh-clone line; `hasDrivePrefix` is checked before the separator rule so `C:\x` reports as a drive; relative `gitdir` under `worktree.useRelativePaths` is joined against the admin dir; findings go to stdout with `SilenceErrors`, so no cobra `Error:` line.

**Fresh verification run this session** (reviewer's own evidence): `go vet ./internal/lane ./internal/cmd` clean; `golangci-lint run ./internal/lane/... ./internal/cmd/...` → 0 issues (2.12.2, the pin); `gofmt -l` empty; `go test ./internal/lane/ ./internal/cmd/ -count=1` green (20.4s / 36.7s); `GOOS=windows go vet` clean. **Mutation probe**: disabling the `errNotAWorktree` branch in `strangerFix` turns `TestAuditLanes_LeavesAnotherRepositorysWorktreeAlone` red on both subtests, then reverted and the tree rebuilt clean — the round-2 regression test is non-vacuous, not a tautology.

### Evaluator rubric

| Dimension | Grade (A-D) | Rationale (one line) |
|-----------|-------------|----------------------|
| Correctness        | B | All four ACs pass named tests I re-ran; one unobserved negative-path gap (unreadable gitdir silently skipped). |
| Verification       | A | Evidence is reproducible commands with anchored test names; the reviewer re-ran vet/lint/tests and a mutation probe, all green/red as claimed. |
| Scope              | A | Diff contains exactly the spec's surface (audit, subcommand, docs, spec folder); the `dotf doctor` sweep entry is explicitly out of scope and ticketed (#3327). |
| Reliability        | B | Error paths handled (unreadable record, empty root, absent marker); the one gap is the skipped gitdir above; no mutation of state anywhere in the audit. |
| Maintainability    | A | golangci-lint 0 issues incl. cyclop/funlen after both rounds' refactors; largest new function well under the 40-line/CC-10 limits. |
| Handoff-readiness  | A | Runbook §7.6 + known-limit section, lesson-582, ADR-061 sentence, follow-up ticket #3327, both review rounds' dispositions recorded with rationale and measurements. |

### Verdict

**PASS** — no Blockers, no Majors; both findings are Minor and neither alone moves the verdict (one THEORETICAL, one SPECULATIVE); rubric all B or above.

### Recommended next steps

For the implementer to disposition in `verification.md` (applied / ticketed / declined) — the contract set is closed, so these are not proposal/tasks edits:

- Report an admin dir with an unreadable `gitdir` as a finding the way an unreadable identity record already is, with a test in `TestAuditLanes_…` for that shape (fix: code + tests; follow-up ticket if not picked up now).
- Optionally record the exit-2 failure path in the runbook §7.6 alongside the exit 0/1 contract.

`dotf spec archive` is advisable in the current state.

---

*Reviewer note: verified by running, not by reading claims — vet/lint/tests fresh this session on `b6d74f79`, contract digests matched, and the one mutation probe above. The full 55-package suite re-run claimed in verification.md was not re-executed here (time cap); the two touched packages were, and both were green. Everything not run by this reviewer is marked accordingly. 40-hex `reviewed_sha` from `git rev-parse HEAD`: `b6d74f792889c69d6c92856d393e76507e5b24fa`.*
