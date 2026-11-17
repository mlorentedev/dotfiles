---
spec: "HARNESS-187-lane-identity-git-dir"
verdict: "PASS"
reviewed_sha: "c79d475d8a4ebf39af6cdb53d3d16517efb7cf23"
reviewer: "nan/mimo-v2.6-flash"
date: "2026-11-11"
---

## Adversarial review

**Scope**: HARNESS-187-lane-identity-git-dir (#3261) — round 2, the whole change
**Sources**: `specs/HARNESS-187-lane-identity-git-dir/{proposal,tasks,verification,features}.md|json`;
`git diff 3ef5dfa6b7386ee2a39cb25e129caf915b3a119e...HEAD` (branch commits `98210cea`→`c79d475d`
plus `5b1df35f`, an envelope-hygiene deletion unrelated to the feature); round 1
(`review-round1.md`, FAIL on `594d1764`). No merges brought other work into the range
(`git log --first-parent --no-merges` lists only these 11 commits; the diff touches only
spec, lane/hooks, guard and docs files).

Contract digests in `review-request.json` match `dotf spec digest HARNESS-187-lane-identity-git-dir`
exactly (all three files). No `[AGENT-DRAFT]`/`[AGENT-SUGGESTION]` tags in any contract file;
the literal `[AGENT-DRAFT]` string occurs once, inside `review-round1.md`'s negated sentence
("No [AGENT-DRAFT]… tags remain") — whether the archive gate scans prose that way is
UNVERIFIED (gate implementation out of scope).

### Spec and task alignment

- **Round-1 REAL Major is fixed and I re-proved it independently.** `identity.go` now
  short-circuits `gitDir == commonDir` and treats a directory at the record path as no
  record; `TestIdentity_ReadsTheRecordFromTheWorktreeGitDir` creates `.git/dotf-lane/` on
  the main checkout. Mutation: removing both guards reproduced round 1's exact error
  (`read …/.git/dotf-lane: is a directory`) and turned the named test red; reverted.
- **All five ACs verified fresh, by name, in this session:**
  - AC1 → `TestLaneInitRecordsTheIdentityInEachLanesGitDir`,
    `TestLaneInitRestoresADeletedIdentityRecord`, `TestLaneInitIdempotent` — `go test ./internal/lane/` ok.
  - AC2 → `TestPreCommitLaneGuard` case 10 (unregistered name passes; record blocks through a
    real pre-commit hook; block survives `git worktree move`; whitespace record passes) — ok.
  - AC3 → `TestLaneInitRefusesALaneDirectoryOfAnotherRepository`,
    `TestLaneInitRefusesAPlainDirectoryAtALanePath` — ok (error names path + owner; metadata
    byte-identical; no marker written; switch/park refused; identity intact; `HasNamedLanes` false).
  - AC4 → `TestIdentity_ReadsTheRecordFromTheWorktreeGitDir`,
    `TestWriteIdentity_IsIdempotent` — ok (lane, moved lane, no record, main checkout with
    prompt dir, outside git).
  - AC5 → full `go test ./... -count=1` exit 0 (all packages ok, 2026-11-11 on HEAD);
    `grep -q canonical_lanes git-hooks/lib/lane-guard.sh` exits 1.
- **features.json f1..f5 verification commands** each run individually above — all PASS.
- **Mutation red-proofs (run and reverted this session):**
  1. Wrong record path in the guard → `TestPreCommitLaneGuard` red at the hook-commit
     assertion (the test pins the record-based block).
  2. Name-based condition added to rule 5b → red at "unregistered worktree merely named
     'spike-lane' … to pass" (the test pins "never by name").
  3. Round-1 main-checkout fix removed → red with the exact EISDIR error (the test pins
     the fix).
- **Static**: `go build ./...`, `go vet ./...`, `GOOS=windows go vet ./...` clean;
  `golangci-lint run ./...` (pinned 2.12.2 on PATH) reports `0 issues`; `shellcheck -s sh`
  and `sh -n` on the guard clean.
- **Design claim re-verified on a scratch repo**: record in the worktree git admin dir is
  read by the guard exactly as the Go reader does; a main checkout carrying the
  prompt **directory** at `.git/dotf-lane` passes rule 5 (`-f` is false for a directory) —
  observed live, and rule 6 still blocks `main` at the main checkout.
- **No new git floor**: `--path-format=absolute` already existed at the base commit
  (`git show 3ef5dfa6:cli/internal/lane/named.go`), so `identity.go`'s form adds nothing;
  the guard keeps the file's pre-existing `case` absolutisation idiom.
- **Follow-ups verified open on the board**: #3318 (same-basename layout), #3320
  (prompt-dir/record name collision) — both exist and are OPEN; #3261 open.
- **Round-1 dispositions** recorded in `verification.md` with evidence (applied ×2,
  ticketed ×1, declined ×3) — matches the diffs I read.
- **Scope**: diff matches the proposal. Two documented side-changes beyond the proposal's
  "What": `switch`/`park`/`HasNamedLanes` gained the same foreign-worktree refusal (asserted
  by the AC3 test itself, rationale in verification.md "Decisions"), and `5b1df35f` deletes
  `.agent/unattended.yml` (run-envelope hygiene, documented in its message).
- `tasks.md` ticks match the diff; the closing "PR opened" item is honestly unticked
  (no PR is claimed; UNVERIFIED whether one exists — this worktree sits detached on the
  spec's head).

### Findings

| Severity | Reality | Area | Finding | Evidence | Test (named, or UNTESTED) | Fix location |
|----------|---------|------|---------|----------|---------------------------|--------------|
| Minor | THEORETICAL | guard rule 5b | An **unreadable** identity record (chmod 000) fails **open**: `sed … 2>/dev/null` yields an empty name, the `if [ -n ]` guard skips the block, and a branched leaseless lane commits; Go's `Identity` returns an error for the same state (Stat succeeds, ReadFile → EACCES). Reproduced on a scratch repo (record chmod 000, branch checked out → guard exit 0). Nothing in normal flows produces it (`writeIdentity` writes 0644); it needs manual corruption or a botched restore. | reproduced 2026-11-11 (scratch repo, guard exit 0); code read of `lane-guard.sh:107-114` | UNTESTED — no named test for an unreadable record | code (block on read failure, or record the asymmetry) — implementer dispositions in `verification.md` |
| Minor | THEORETICAL | guard/Go record semantics | **Multi-line record residue**: the guard reads the first line only (`sed … 1p` → reports `line1`), `Identity` returns the whole trimmed content (`TrimSpace` → `"line1\nline2"`). Both sides block/answer "a lane", but the identity strings diverge. Round 1's empty-record divergence was fixed (and is now tested); this is the adjacent residue. `writeIdentity` never writes this content. | deterministic from code (`lane-guard.sh:108` vs `identity.go:44-48`); no repro of a real occurrence | UNTESTED (case 10 covers whitespace-only, not multi-line) | code (read identical bytes on both sides) or decline with a comment |
| Minor | SPECULATIVE | lane init | `writeIdentity` writes in place (ReadFile-compare → `os.WriteFile`, no temp+rename); a crash mid-write could leave a truncated record that `Identity` answers `ok=true` with a wrong name. Single small write, init under the worktree lock — probability very low. | code read of `identity.go:69-76` | UNTESTED | code (write temp + rename) — surface only; do not gate |
| Minor | THEORETICAL | spec drift | `tasks.md` AC2 green step names `rev-parse --absolute-git-dir` for the guard; the implementation uses the script's pre-existing `case` absolutisation idiom. Behavior proven by tests; rationale recorded in `verification.md` ("Decisions"); I verified no new git floor. Round 1 already declined this — carried forward for visibility only. | `tasks.md` vs `lane-guard.sh:34-37`; `git show 3ef5dfa6` base | `TestPreCommitLaneGuard` (behavior) | none — declined and recorded |
| Question | THEORETICAL | guard rule 6b (pre-existing) | Rule 6b blocks **any** leaseless detached worktree — including unregistered ones — with a message claiming "at the main checkout"; observed live in a scratch linked worktree. So AC2's "not blocked by name" holds only for checked-out branches (case 10a uses a branch). Rule 6b predates this spec unchanged (`git show 3ef5dfa6:git-hooks/lib/lane-guard.sh`), so it is an interaction note, not a defect of this change. | scratch repro (detached worktree → 6b message); base guard | UNTESTED (new shape, pre-existing rule) | none this spec — owner may note for a follow-up |
| Question | — | portability | Windows runtime remains UNVERIFIED on this box (macOS). Declared in the proposal's Risks; `GOOS=windows go vet` and `GOOS=windows` lint clean; CI's Windows leg is authoritative. | proposal.md Risks; local runs | — | none (declared) |

### Evaluator rubric

| Dimension | Grade (A-D) | Rationale (one line) |
|-----------|-------------|----------------------|
| Correctness        | B | All 5 ACs verified with negative paths, three mutation red-proofs and a live scratch-repo probe; the residual gaps are corrupted-record handling (reproduced fail-open under chmod-level corruption) — minor negative paths. |
| Verification       | A | Every criterion maps to a named test, each run fresh in this session; features.json commands run individually; digests checked via `dotf spec digest`; full suite, vet, pinned lint, shellcheck all green on HEAD. |
| Scope              | B | Diff matches the proposal; switch/park/HasNamedLanes refusals ride with AC3 and are documented, plus one unrelated envelope deletion (`5b1df35f`) called out in its message. |
| Reliability        | B | Refusal happens before any write; a deleted record self-heals on re-init (tested 1→0→0); idempotent. Fail-open on an unreadable record keeps this below A. |
| Maintainability    | B | Small functions, one absolutisation idiom per file, sh/Go drift guarded by the shared `lane.LaneIdentityFile` constant in tests; the `dotf-lane` name collision is the smell (ticketed #3320). |
| Handoff-readiness  | A | ADR-061, lesson 579, runbook updated with the record's lifecycle; #3318/#3320 filed and verified OPEN; round-1 dispositions recorded with evidence; promotion lines answered; PR item honestly unticked. |

### Verdict

**PASS** — no blockers; no majors (round 1's REAL Major is fixed and re-proven by mutation
red-proof this session); all open findings are Minors/Questions tagged THEORETICAL or
SPECULATIVE, each with a disposition line; rubric is B or above in all six dimensions.

Severity × reality: the only reproduced defect mechanism (fail-open on an unreadable record)
requires manual corruption no code path produces, and is Minor — it does not gate. The
multi-line divergence and non-atomic write are THEORETICAL/SPECULATIVE and cannot, by
themselves, move the verdict.

### Recommended next steps

Contract set (`proposal.md`, `tasks.md`, `features.json`) is **closed** by this PASS — no
contract edits requested. For the implementer to disposition in `verification.md`
(applied / ticketed / declined with a reason), not as archive conditions:

- Row 1 (unreadable record fails open): decline with a reason or fold a read-failure branch
  into rule 5b with a named test — the cheapest hardening of the guard's only fail-open path.
- Row 2 (multi-line divergence): decline with a comment, or make one side read what the
  other writes; both are one-liners outside the contract set.
- Rows 3-6: surface only; row 4 already declined in round 1; row 6 is pre-existing rule 6b
  and belongs to a follow-up only if the owner wants the message corrected.

### Completion notes (Definition of Done, reviewer side)

- **Debt**: the findings table is the debt; each row carries a fix location and a disposition
  path. No defect was left unmentioned.
- **Knowledge**: nothing new beyond what ADR-061 and lesson 579 record; the review file is
  the record of the fail-open probe and the red-proofs.
- **Board**: skipped — the reviewer writes only `review.md`; #3261 state (OPEN) observed and
  reported, not modified.
- **Review**: no PR is claimed by the spec (`tasks.md` closing item unticked); skipped with
  that reason. No PR comments made, per the run's constraints.
- **Evidence**: every claim above carries command output produced in this session (test runs,
  mutation round-trips, scratch-repo guard probes, `dotf spec digest`, lint/vet/shellcheck,
  full suite). Working tree left clean apart from `review.md` (and the launcher's
  `review-request.json`), both inside the spec folder.
- **Second opinion**: not sought — this run *is* the independent second pass (round 1 was a
  different model, `nan/glm5.3-flash`, whose findings I re-verified rather than inherited);
  no further model pass was available or requested inside this run. Where I differ from
  round 1: I accept its verdict's core, and I add the reproduced rule-5b fail-open (round 1
  flagged only the empty-record side) and the rule-6b interaction note.
