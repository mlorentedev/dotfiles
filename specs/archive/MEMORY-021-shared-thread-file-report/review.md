---
spec: "MEMORY-021-shared-thread-file-report"
verdict: "PASS-WITH-GAPS"
reviewed_sha: "a28674bfc90ce5a903e74d1cb03fcbaa2e9c1cf0"
reviewer: "nan/mimo-v2.6-flash"
date: "2026-11-14"
---

## Adversarial review

**Scope**: MEMORY-021-shared-thread-file-report (issue mlorentedev/dotfiles#3521)
**Sources**: `specs/MEMORY-021-shared-thread-file-report/{proposal,tasks,verification,features,review-request}.md|json` + `git diff 439a54b30702f3539ed3947e5c6777cf4b9fbc51...HEAD` (single commit `a28674bf`, no merges in range)

### Spec and task alignment

- **Range**: `git log --first-parent --no-merges base..HEAD` → one commit (`a28674bf`). Diff = spec folder + 6 code files, all named by the proposal (`park_threads.go`, `thread_archive.go`, `sweep.go` + their tests). No scope creep; no test deletions (the diff's single `-` line across `cli/**/*_test.go` is the `r.record(...)` input literal replaced with itself plus `Shared:`).
- **Contract digests**: `dotf spec digest MEMORY-021-shared-thread-file-report` matches `review-request.json` byte for byte (`features.json a597bce6…`, `proposal.md 00c718f8…`, `tasks.md 95c61e5e…`) → review is fresh at write time. No `review: waived`, no `risk:` field (so a second signature is not required); reviewer id `nan/mimo-v2.6-flash` is a pool entry with vendor NaN (eligible first signer). No `[AGENT-DRAFT]`/`[AGENT-SUGGESTION]` tags in the folder. Issue #3521 is OPEN (`gh issue view`).
- **AC1 — behaviour**: verified. Red→green: the three test files overlaid on a `git archive base` copy of `cli/` fail to compile exactly as `verification.md` claims (`got.Shared undefined` at `thread_park_archive_test.go:194`, `unknown field SharedThreads` at `park_archive_test.go:136`); at HEAD they pass. Report bytes match the proposal's wording exactly — park line `archived   thread "feat-x": its thread file is another thread's and stays in threads/, so the archived entry's link does not lead to it` (`park_threads.go:19`), sweep line `  - archived thread: feat-x (its thread file is another thread's and stays in threads/)` (`park_threads.go:36`). Routing confirmed: `lane park` (`lane.go:154`), `worktree done` (`worktree.go:774`) → `reportParkThreads`; `worktree sweep` (`worktree.go:501`) and `lane apply` (`lane_apply.go:72`) → `reportSweepThreads`. `moveThreadFile`'s only behaviour change is the new `shared` return: `held == nil` still returns "not ours, no error", `CheckThreadFile != nil` still leaves the file, errors still propagate and are still joined in `moveThreadFiles`.
- **AC1 — the JSON clause**: the struct tag is correct and I proved the wire bytes with a probe in a `mktemp` copy (never in the repo): `{"reaped":null,…,"shared_threads":["feat-x"],…}` with the case, key absent when empty, so the proposal's "omitempty, byte-identical" resolution holds. **No repository test asserts any of that** → Finding 1. Also note the proposal's "`worktree sweep` … its JSON report" has no emitter today → Finding 3.
- **AC2**: `awk` (features.json f2) exits 0 — every function in `thread_archive.go` ≤ 40 lines. CC: `.golangci.yml` confirms `gocyclo` runs only under `internal/cmd/` (`path-except: internal/cmd/`), so the proposal's "lint cannot see this" is accurate; I ran gocyclo myself with `min-complexity: 9` over `internal/mem` — **no function in `thread_archive.go` reaches CC 10** (corroborating `verification.md`'s hand counts ≤ 8). Full evidence re-run this session: `go build ./...`, `go vet ./...`, `GOOS=windows go vet ./...`, `go test ./... -timeout 5m` (all packages ok), `golangci-lint run ./...` and `GOOS=windows golangci-lint run ./...` (0 issues each), features.json f1 (3 named tests PASS, exit 0) and f2 (exit 0). The "existing archive test unchanged" wording has a caveat → Finding 4.
- **Tasks**: every `[x]` has diff or run evidence; no tick found without support. `features.json` states are `pending` with empty evidence, which is the only state an agent may write.

### Findings

| Severity | Reality | Area | Finding | Evidence | Test (named, or UNTESTED) | Fix location |
|----------|---------|------|---------|----------|---------------------------|--------------|
| Major | THEORETICAL | tests / AC1 JSON clause | AC1's "`shared_threads` in JSON" clause has no covering test, and `verification.md` mis-attributes it to `TestSweepReportRecordsThreadArchives`, which only asserts the Go field and never marshals. Current bytes are correct (my probe), so the exposure is drift: the sibling wire name `lane_skips` has an explicit contract test (`TestSweepReportNamesLaneSkipsOnTheWire`, whose comment says "The JSON name is the wire contract the report's readers hold on to") while this one has none, and features.json f1 does not exercise it either. | `grep -rn shared_threads cli --include=*.go` → only `sweep.go:142`; marshal probe output above; `verification.md` AC1 line | UNTESTED | tests (marshal test beside `TestSweepReportNamesLaneSkipsOnTheWire`) + `verification.md` wording (outside contract set) |
| Major | THEORETICAL | tests / negative path | Nothing guards the negative side of `Shared`: a mutation making `moveThreadFile` report `shared` on every success (inline thread and own-file thread included) **survived** a full `go test ./internal/mem` (its only 2 failures are probe-copy artifacts — missing repo-root `session-start-config.json` and `../../../harness/skills/autopilot/SKILL.md` — both green in the real tree) plus `go test ./internal/cmd ./internal/worktree -run 'Park\|Thread\|Sweep\|Archive\|Done\|Handoff'` (ok, 9.9s/20.3s). So every park could claim every archived thread's file belongs to another thread, with the suite green. The complementary mutant (shared never reported) **is** caught at `thread_park_archive_test.go:195`, so the positive path is covered and only the negative path is open. | mutation runs in a `mktemp` copy of HEAD; both mutants' outputs above | UNTESTED (no named negative test) | tests (assert `Shared` empty on the normal-archive and inline-thread paths) |
| Minor | REAL | spec wording | AC1 presumes "`worktree sweep` … its JSON report gains `shared_threads`", but no command emits `SweepReport` as JSON: `newWorktreeSweepCmd` registers only `repoDir`/`dryRun`, `--json` exists only on `worktree list` (`worktree.go:265`), `lane apply --json` encodes only `{changed}`, and no non-test code marshals `SweepReport`. The clause can only mean the struct tag, which exists. Inherited framing from MEMORY-020 (`archived_threads`/`kept_threads` are in the same position and archived). | grep of `"json"` flag registrations and `json.Marshal`/`Encoder` sites in `cli/internal/cmd` | n/a (observation) | spec reading → record the disposition in `verification.md` (contract set closed under this verdict) |
| Minor | REAL | spec vs code (literal) | AC2 says "every existing archive test **unchanged**", but two pre-existing tests were modified: `TestArchiveParkedThreadLeavesAFileAnotherKeyHolds` (existed at base `439a54b3`, gained the `Shared` assertion) and `TestSweepReportRecordsThreadArchives` (input literal extended). Changes are additive — the retention bar's substance (nothing weakened, nothing deleted) holds and both pass — but the literal wording is not satisfied. | `git show base:…` proves both tests pre-exist; test-file diff has one `-` line only | `TestArchiveParkedThreadLeavesAFileAnotherKeyHolds`, `TestSweepReportRecordsThreadArchives` (both PASS) | spec reading → disposition in `verification.md`; no contract edit requested |

Observations (not findings): a shared key prints twice — the MEMORY-020 `archived … to …` line plus the new shared line (and `  - archived thread: k` plus `  - archived thread: k (…)` in sweep) — because the key sits in both `Archived` and `Shared`. The proposal lists both lines, so this is its letter, not a defect. Error path is unchanged: the shared line prints before the ERROR/WHY/FIX return in `reportParkThreads`, exactly as the archived/kept lines always did.

### Evaluator rubric

| Dimension | Grade (A-D) | Rationale (one line) |
|-----------|-------------|----------------------|
| Correctness        | B | Every clause's behaviour verified by re-run tests and probes (message bytes, red→green, CC/line limits); not A because the negative path of the new field is unguarded (Finding 2). |
| Verification       | B | Commands in `verification.md`/`features.json` all reproduced green this session; not A because one evidence clause names a JSON field no test touches (Finding 1). |
| Scope              | A | One commit, diff equals the proposal's What/Out-of-scope exactly, no deletions, no unrelated files. |
| Reliability        | B | Error propagation, write order and ERROR/WHY/FIX behaviour preserved through the `parkTarget`/`archiveParked` split; no new failure path introduced. |
| Maintainability    | B | All functions ≤40 lines and CC ≤9 in `thread_archive.go` (objectively measured); clear extraction (`takesThread`, `blockEnd`, `sharedFileNote`); not A for the missing negative-path test. |
| Handoff-readiness  | B | Spec triad + features.json landed in the same commit, promotion declines reasoned, archive checklist pending; not A because `verification.md` over-claims the JSON coverage and the board ticket (#3521) is still open. |

### Verdict

**PASS WITH GAPS** — no Blocker, no REAL Major, no rubric D; two open Majors (Findings 1–2), both tagged THEORETICAL (the defect they describe has never been observed — current output is correct per probes — what is demonstrated is that a future regression would ship green) and both UNTESTED, which caps this verdict at PASS WITH GAPS per the test-traceability gate.

### Recommended next steps

Contract set (`proposal.md`, `tasks.md`, `features.json`) is closed under this verdict — none of the following asks for a contract edit; each is for the implementer to disposition in `verification.md` (applied / ticketed / declined with a reason) or carry into a follow-up ticket:

1. **tests** — add `TestSweepReportNamesSharedThreadsOnTheWire` mirroring `TestSweepReportNamesLaneSkipsOnTheWire`: marshal with the case (`"shared_threads":["feat-x"]`) and empty (key absent). Covers Finding 1 and the omitempty half of the proposal's risk resolution.
2. **tests** — assert `Shared` (and `SharedThreads`) stay empty on the normal-archive and inline-thread paths; this is the mutation that currently survives the suite (Finding 2).
3. **`verification.md`** — correct the AC1 evidence line: `TestSweepReportRecordsThreadArchives` covers the `SharedThreads` *field*, not the `shared_threads` JSON name; record dispositions for Findings 3 and 4 there.
4. **follow-up (optional)** — decide whether `worktree sweep` will ever have a JSON emitter for `SweepReport` (Finding 3); if yes, the wire test in (1) becomes the gate for it.

**Verdict: PASS WITH GAPS.** `dotf spec archive` is **advisable once the dispositions above are written into `verification.md`** (outside the staleness set, so it can be edited without invalidating this review): the gate's pre-flight — fresh passing review, contract digests, no `[AGENT-DRAFT]` tags, promotion candidates answered — is satisfied as recorded here. To reach a plain PASS, findings 1 and 2 need their named regression tests (tests only; no re-review of the contract is implied, but the tests must exist before the gaps are called closed).
