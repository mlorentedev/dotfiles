---
spec: "CLI-119-lane-next"
verdict: "FAIL"
reviewed_sha: "94fea6b6baaa7ed3bdd58ef0f1933351684cfa31"
reviewer: "nan/glm5.3-flash"
date: "2026-10-15"
---

## Adversarial review

**Scope**: CLI-119-lane-next — `dotf lane exec` + `dotf lane next` (herdr backend, `.dotf/lane.json` queue, prompt template, lease), round 2.
**Sources**: `specs/CLI-119-lane-next/{proposal,tasks,verification}.md` + `features.json`; diff `cfffe79de47b31b326ae1222248f78414bd6e55a...94fea6b6` — the whole change since the base the launcher resolved. The lane change is three commits (`58909de7` #2532, `f0c1b9e5` #2555, `94fea6b6` #2578); the rest of the 156-file window is other merged PRs (deploy manifest v6, worktree agent/adopt/migrate, CLI-127, harness bind, release 0.71.0) that carry their own specs, and I did not deep-review them here — stated as a scope limit, not a clean bill.

Everything below was run fresh at `94fea6b6` on this Mac (dotf release 0.71.0, herdr 0.9.3). Working tree left clean apart from the launcher's `review-request.json`.

**Gate re-ran green**: `go build ./...` exit 0 · `go vet ./...` exit 0 · `go test ./internal/lane ./internal/cmd ./internal/worktree -count=1` → `ok` ×3 · `GOOS=windows go vet ./...` exit 0 · `golangci-lint` 2.12.2 on `./internal/lane/... ./internal/cmd/...` → `0 issues.` · `bats tests/herdr-config.bats` → 6/6 ok · all six `features.json` verification commands exit 0 from the repo root. Mutation check on the round-1 Blocker fix: restoring `SilenceErrors` in `cmd/lane.go` turns `TestLane_RefusalsAreShownOnStderr` **red** (reverted); live refusals print: `dotf lane exec --agent copilot …` → `Error: lane: no launch command for agent "copilot" (known: agy, claude, pi)`, exit 1. Live `lane next --dry-run` from this linked worktree decides and prints the manual commands with the deciding binary's absolute path, exit 0. No `[AGENT-DRAFT]`/`[AGENT-SUGGESTION]` tags in the contract files (hits grepped earlier are inside `review-transcript.jsonl` only).

### Spec and task alignment (round-1 dispositions re-verified)

- **Blocker (silenced errors): fixed and regression-tested.** `SilenceErrors` is gone from both leaf commands; the named test asserts stderr and kills the mutation.
- **AC5 (no backend, no `--agent`): met.** The placeholder form prints the four commands with `--agent <agy|claude|pi>` and a note to fill it, exit 0 (`TestNext_NoBackendAndNoAgentPrintsAPlaceholder`). I rule this satisfies AC5's letter — the commands *are* printed, nothing is privileged, and the prior round offered exactly this resolution. The proposal text needed no rewording.
- **Printed worktree path: fixed.** `manualCommands` builds the path from `WORKTREE_ROOT` + `worktree.ResolveWorktreePath`, refuses an unresolvable root (`TestNext_NoBackendWithoutAWorktreeRootRefuses`), and `TestNext_NoBackendPrintsTheCommands` pins the printed path to the layout rule.
- **Release gap: closed.** `dotf --version` → 0.71.0; `dotf lane --help` answers. The successor prompt's `dotf lane next` now resolves on the installed release.
- **Silent bare-`dotf` substitution: fixed loudly** (`TestNext_ASubstitutedBinaryIsSaidSo`; failed `os.Executable()` reported on stderr). **`base` without a slash: refused** (`TestLoadConfig_BaseMustNameARemoteAndABranch`). **Lengths: `Handoff` 39, `runLaneNext` 39** (measured with awk). **The three unhandled failure modes are ticketed**: #2576 OPEN with root cause and options.
- **Stale boxes: ticked** — all six AC boxes in `proposal.md` and every `tasks.md` Closing box are now `[x]`, matching reality.
- AC1–AC6 each map to named, re-run green tests (see `features.json`), and AC6's three `RESULT <agent> PASS` lines are present; AC6 remains prose evidence (a real chain is not re-runnable from here), honestly disclosed as not re-measured live this round.

### Findings

| Severity | Reality | Area | Finding | Evidence | Test (named, or UNTESTED) | Fix location (code / tests / spec / vault) |
|----------|---------|------|---------|----------|---------------------------|---------------------------------------------|
| Major | REAL | input handling / every-project scope | `dotf lane next` **fails outright when the repo's git common dir contains a space**: the prompt is always written under `<git-common-dir>/dotf-lane/`, the `plainWord` guard rejects it, and the error advises "use a PromptDir of letters, digits…" — advice the user cannot follow, because `PromptDir` is derived (`cmd/lane.go`), not configurable. Windows usernames with spaces (`C:\Users\Jane Doe\…`) make this the common case on a first-class OS of this repo. The no-backend `manualCommands` output (`git -C <parent>`, `cd <dir>`, unquoted) breaks on spaces in the same family. | **Reproduced this session**: scratch repo at `/tmp/a b/repo` with a lane config + stubbed `gh` → `Error: lane: the prompt path "/private/tmp/a b/repo/.git/dotf-lane/999.md" has characters a pane's shell would read differently…`, exit 1. Tracked as #2537 (OPEN, root cause + fix options). | UNTESTED — #2537 itself notes the guard is exercised on synthetic space-free paths only; no named test runs `Handoff` with a spaced repo path. | code + tests (ticket #2537): pick a space-free prompt location or make it overridable, and quote the printed manual commands |
| Minor | SPECULATIVE | shell portability | `plainWord` admits `\` and `:` on POSIX, where zsh/bash read a backslash as an escape: a POSIX path containing a literal `\` passes the guard yet reaches the pane's shell mangled. Windows needs the `\`, so the fix is per-GOOS, not per-regexp. | code read of `handoff.go` `plainWord`; no spaced/backslashed path observed on this machine | UNTESTED | code (surface only; do not gate) |
| Minor | THEORETICAL | concurrency | `excludeLease` is a read-modify-write append on the shared `<git-common-dir>/info/exclude`; two concurrent `lane next` runs can interleave and drop an append. Impact is low: the content check makes a re-run re-append. | code read of `lease.go`; no concurrent run observed | UNTESTED | code/tests (could ride along with #2576's start race) |
| Minor | THEORETICAL | performance | `Next` spawns one `gh issue view` per queued item, serially — one subprocess per issue. Harmless at the current queue size; worth batching (`gh issue list --json`) if queues grow. | code read of `lane/gh.go` + `queue.go` | UNTESTED | code (surface only) |
| Question | — | process | `features.json` still carries hand-authored `"state": "passing"`; this round's `verification.md` discloses that honestly, all six commands re-run exit 0 here, and no evidence field is empty — accepted with the disclosure. `specs/CLI-118-worktree-one-root/` inside this review window still has no `review.md`; that is that spec's own archive gate. The ~600-executable-line single PR (verification.md "Decisions") exceeds the ~300-line atomic-PR cap — a documented owner-visible decision, noted for the record. | read of the three files; all six verification commands re-run exit 0 | n/a | process |

### Evaluator rubric

| Dimension | Grade (A-D) | Rationale (one line) |
|-----------|-------------|----------------------|
| Correctness | B | All six ACs verified against code, named tests, mutations and live runs; one reproduced out-of-AC failure (#2537) remains inside the proposal's "every project" scope. |
| Verification | B | Every stated gate re-ran green here, including the six `features.json` commands and a mutation on the Blocker fix; AC6's chain and the Windows exec path remain prose/compile-only, as disclosed. |
| Scope | B | The three lane commits match the proposal plus the declared `.dotf/` data and CI row; the wide diff window is the launcher's base spanning other merged specs, and the one-PR-instead-of-two line count is declared. |
| Reliability | B | Errors are loud and tested, refusals leave the backend untouched, the layout rule is reused, release gap closed; the spaced-path hard failure and ticketed #2576 modes keep it from A. |
| Maintainability | B | Clean seams (`Runner`/`Backend`/`IssueSource`), WHY comments, functions ≤39 lines, lint clean; the `plainWord`/`\` portability wrinkle and the duplicated guard-and-print knowledge are the residue. |
| Handoff-readiness | B | Contract boxes ticked, dispositions written, promotions answered (lesson 438), remaining risks disclosed — while open #2537 is not named anywhere in `verification.md`'s dispositions. |

Aggregation: all **B**, no D → the rubric alone reaches PASS; the severity path independently reaches **FAIL** on the single **REAL Major**. Per the rule, the more severe path wins.

### Verdict

**FAIL**

Not a repeat of round 1: every Blocker, Major and Minor that round raised is fixed, tested or ticketed, and I verified each fix mechanically (mutation, live run, release check). What stops the archive is one reproduced defect the round did not cover: the command this spec exists to ship cannot run at all for any repository checked out under a path with a space — a class the proposal's own "every OS and every project" puts in scope, and whose error message tells the user to change a setting that does not exist. It is loudly ticketed (#2537), which satisfies the implementer's track-or-fix duty, but a REAL Major is a fix-and-re-review finding, not a tracked-and-pass one.

### Recommended next steps

Minimum set to flip to PASS (all land outside the contract set — code, tests, `verification.md` — so nothing here forces a contract edit):

1. **Fix the spaced-path failure (#2537)** — write the successor prompt somewhere space-free (or make the location overridable so the guard's own advice is followable), and quote the paths in `manualCommands`' printed lines. Scope: code.
2. **Add the named regression test** — a `Handoff`/`runLaneNext` case whose repo path contains a space, asserting either success or the intended behaviour; without it the fix is UNTESTED and the gate stays shut. Scope: tests.
3. **Disposition #2537 in `verification.md`** — record it as applied with the reproduction above as the before/after; name it alongside #2576 so the round's record is complete. Scope: verification.md.
4. **Re-review at the new sha** (`dotf spec review CLI-119-lane-next`): the delta is small, everything else in this round verified green, and findings 2–4 in the table are surface-only.

**Is `dotf spec archive CLI-119-lane-next` advisable now? No.** It would archive with a FAIL-signed review on record; items 1–3, then a fresh review, are the path to archive.
