---
tags: [spec, verification, templates]
created: "2026-09-02"
---

# Verification - CLI-071-triage-queue-transport

## Evidence

- [x] **AC1 — the fetch path is tested** → `TestFetchWithRunnerDrivesTheWholePath`.
      `fetchWith` runs end to end against an injected `ghRunner` with no network.
      This test could not be written before the change: `FetchWithRegistry` called
      `exec.CommandContext` directly and was unreachable.
- [x] **AC2 — no GraphQL remains** → `TestFetchUsesRESTOnly` asserts on the
      requests the seam receives, not merely on the queue that comes back. A
      reintroduced `gh pr list` produces an identical queue and would pass every
      other test in the package, so asserting the transport is the only way to
      keep this true.
- [x] **AC3 — the `[bot]` fold is pinned** → `TestFetchFoldsTheBotLoginSuffix`,
      both declared reviewers, with REST-shaped `coderabbitai[bot]` /
      `github-actions[bot]` authors against a registry declaring bare logins.
- [x] **AC4 — truncation refuses on both axes** →
      `TestFetchRefusesTruncationOnBothAxes`, one subtest per axis.
- [x] **AC5 — the fan-out overlaps and is bounded** →
      `TestFetchFansOutConcurrently` asserts *observed* peak concurrency rather
      than wall-clock, so it is deterministic on a loaded CI runner. Both
      directions are checked: serial fails it, and so does exceeding the cap.
- [x] **AC6 — the renderers see identical values** → the field assertions in
      `TestFetchWithRunnerDrivesTheWholePath`, plus the production smoke test
      below, which is the stronger evidence of the two.

Two behaviours beyond the acceptance criteria are also pinned, because both are
load-bearing and neither had a test: a failure on either axis is returned rather
than softened into an empty queue (`TestFetchReportsFailureRatherThanAnEmptyQueue`),
and an empty `--repo` still resolves to the current repository
(`TestFetchTargetsTheCurrentRepoByDefault`) — the default the session-start probe
depends on, since it passes `""` and runs wherever the shell happens to be.

## Test status

```
cd cli
go build ./...            → clean
go vet ./...              → clean
GOOS=windows go vet ./... → clean
go test ./... -count=1    → all packages ok, no failures
go test -race -count=3 ./internal/prtriage/  → clean
golangci-lint run         → 0 issues   (v2.12.2, the versions.conf pin)
gofmt -l internal/prtriage/ → clean
```

The race detector is run explicitly because this change introduces the package's
only concurrency and `golangci-lint` does not cover it. Three iterations, clean.

Three files elsewhere in `cli/` are unformatted on `main`
(`internal/errors/latch_test.go`, `internal/fsmode/fsmode.go`,
`internal/orca/orca.go`). They are pre-existing and were deliberately left alone
rather than swept into this diff.

**Production smoke test — the strongest evidence here.** The binary was built and
run against the real repository, and its output compared against what the GraphQL
implementation printed for the same repository earlier the same session:

```
$ dotf pr triage-queue
1 pull request(s) awaiting a disposition:

  #1455  feat(harness): fire the persona suggester on every prompt, failing open by construction
         github-actions reviewed, never triaged, 2026-09-02 20:27
         https://github.com/mlorentedev/dotfiles/pull/1455
...
exit=1
```

Byte-identical to the GraphQL output: same pull request, same reviewer, same
reason, same timestamp, same URL, same exit status. AC6 is therefore verified
against live data and not only against a fixture.

**Latency, measured rather than assumed** (R1 was the risk that mattered): three
consecutive runs against the live repository took **0.94 s / 0.96 s / 0.95 s** for
1 + 2 REST calls, against the 5 s session-start budget in `cmd/mem.go`. Cost is
dominated by `gh` process spawn (~0.3 s each), which is why the fan-out overlaps
them; with the cap at 8, ten open pull requests cost two waves rather than ten
round-trips.

**No regressions.** One existing test was removed and one edited, both with cause:

- `TestParseWireBoundaryLimit` tested the truncation guard through `parseWire`, a
  helper that unpacked the GraphQL nesting. Both the helper and that transport are
  gone. Its replacement, `TestFetchRefusesTruncationOnBothAxes`, is strictly
  stronger — it drives the real fetch path and covers the second axis REST
  introduced. A comment at the old site records where the coverage went.
- `session_start_triage_test.go` carried a fixture error string,
  `"gh pr list: exit status 4"`, that this change made unproducible. Updated to
  `"gh api pulls: ..."`. Cosmetic to the renderer under test, and left stale it
  would be lesson 259 committed again in the very PR that cites it.

### Mutation testing

Every new guard was mutated and each mutation was caught. A guard that survives
its own mutation is decorative, and this spec's predecessor shipped exactly that
defect.

| Mutation | Detected by |
|---|---|
| drop the `[bot]` login fold | `TestFetchFoldsTheBotLoginSuffix` |
| set `commentFanout = 1` (serialise the fan-out) | `TestFetchFansOutConcurrently` |
| remove the comment page guard | `TestFetchRefusesTruncationOnBothAxes` |
| remove the pull-request page guard | `TestFetchRefusesTruncationOnBothAxes` |
| map `url` instead of `html_url` | `TestFetchWithRunnerDrivesTheWholePath` |
| swallow a per-PR comment error | `TestFetchReportsFailureRatherThanAnEmptyQueue` |
| reintroduce `gh pr list` | `TestFetchUsesRESTOnly` |
| keep the GraphQL `createdAt` spelling | `TestFetchWithRunnerDrivesTheWholePath` |

**The `features.json` commands were themselves checked for vacuity**, because the
previous spec shipped a verification command naming a test that did not exist and
it passed anyway:

```
$ go test ./internal/prtriage/ -run '^TestThisDoesNotExist$'                       → exit 0   ← the trap
$ go test ... -v -run '^TestThisDoesNotExist$' | grep -q '^--- PASS: TestThis…'    → exit 1   ← the form used
```

Every entry in `features.json` uses the second form, so a renamed or deleted test
fails its own verification instead of silently passing it.

## Decisions made during implementation

**The ticket's proposed fix could not work, and establishing that was the first
task.** It called for a one-line swap to `GET /repos/{o}/{r}/pulls` with a field
mapping that never mentioned comments — but the comments *are* the algorithm
(`prtriage.go:95-141` compares reviewer timestamps against the triage marker), and
that endpoint returns none. The honest shape is 1 + N. The ticket body was
corrected in place before any code was written; this is the third ticket in four
whose premise did not survive measurement.

**The seam is the deliverable; the transport is the implementation.** Framing it
the other way round would have justified the work on an outage observed exactly
once, and left the actual defect — the only path in the package with no test was
the only path observed failing — untouched regardless of transport.

**`normaliseLogin` was promoted from defensive to load-bearing, so it got a test.**
The registry declares `github-actions`; REST returns `github-actions[bot]`. Under
GraphQL the fold was insurance. Under REST every match depends on it, and removing
it does not error — it silently empties the queue, which is precisely the failure
this package exists to prevent (#1033).

**No new dependency.** `golang.org/x/sync` is not in `go.mod`, and adding one is
its own Discipline Gate trigger. The bounded fan-out is a semaphore channel and a
`WaitGroup`, about fifteen lines.

**Concurrency is asserted by observed peak, not by wall-clock.** A timing
assertion would be flaky on a loaded runner and would eventually be deleted or
loosened into meaninglessness. Peak in-flight is deterministic and checks both
directions — serial fails, and so does unbounded.

**One unreadable pull request fails the whole answer.** A PR whose comments could
not be fetched is not a PR with no comments, and returning the rest as a complete
queue would reintroduce "cannot compute reads as nothing pending" through the back
door.

**Deliberately not done: the distinct exit statuses.** `pr.go:49-51` documents the
shared status as a reasoned choice. Refuting that argument is its own ticket, and
folding it in silently would have been a public-contract change smuggled inside a
transport fix.

**Two other `gh pr list` sites were found and deliberately left alone.** The
lesson-259 prose sweep surfaced them, so they are recorded here rather than left
as an unstated skip: `scripts/bitacora-rollout.sh:132,196` and
`harness/skills/catchup/SKILL.md:72`. Both carry the same GraphQL dependency this
change removed from `prtriage`. Neither was migrated and neither was ticketed:
they bind no doctrine, neither has been observed failing, and a vendor condition
is not a defect in them. If GraphQL refusals recur, this note is where to start.

**Also noticed, also out of scope:** `gh api pulls: exit status 1` still hides
gh's own stderr reason (`HTTP 403`, `Not Found`). That opacity is pre-existing —
the old message had the same shape — and unwrapping `*exec.ExitError` to surface
`Stderr` is three lines that belong to whoever decides the error contract, not to
a transport change.

## Promotion candidates

- [x] Lesson for the repo's `docs/lessons/`? **Yes** — a seam is not a testing
      detail: the path with no seam is the path with no test, and it will be the
      path that fails in production. Worth writing because the transport looked
      like the bug and was the smaller half.
- [ ] ADR-worthy decision? **No.** ADR-020 already governs the CLI convergence;
      this changes one package's transport inside it and establishes nothing new.
- [ ] New pattern candidate for `00_meta/patterns/`? **No** — not yet observed in
      a second project. Revisit if the same "vendor CLI called directly, therefore
      untestable" shape turns up elsewhere.

## Archive checklist

- [ ] `proposal.md` frontmatter set to `status: archived`
- [ ] Folder moved: `specs/CLI-071-triage-queue-transport/` -> `specs/archive/CLI-071-triage-queue-transport/`
- [ ] Bitácora board ticket for this spec moved to Done / closed with PR link (ADR-018)
- [ ] Independent adversarial review recorded in `review.md` with a passing verdict
- [ ] Promotions above executed (if any)
