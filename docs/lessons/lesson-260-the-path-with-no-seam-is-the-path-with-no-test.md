# 260 - The path with no seam is the path with no test, and it is the one that fails in production

**Date:** 2026-09-02
**Area:** CLI, testing, PR triage

## What happened

`dotf pr triage-queue` failed on 2026-09-02 with `gh pr list: exit status 1`.
GitHub's GraphQL bucket was refusing every list and write while REST served the
same data throughout. That command is the mechanism the pr-stewardship doctrine
names by hand — run it at session start, and again before reporting PR work
complete — so #1454 was filed against the transport: swap the GraphQL call for
REST.

Measuring before writing found two things the ticket had wrong, and one it had
not looked for.

**Wrong: the proposed fix could not work.** It called for a one-line swap to
`GET /repos/{o}/{r}/pulls`, with a field mapping listing `number`, `title`,
`head.ref`, `draft`, `user.login`. But the queue's algorithm is a timestamp
comparison between the newest reviewer comment and the newest recorded triage
(`prtriage.go:95-141`) — **the comments are the algorithm**, and that endpoint
returns none of them. `gh pr list --json comments` is a single GraphQL query
returning pull requests with their comments *nested*; REST has no such nesting.
The honest shape is 1 + N calls, with the join moved into our code.

**Overstated: "unavailable."** Both consumers already failed loud. The command
exits non-zero with the reason; the session brief injects
`[pr-triage] queue could not be computed: …` followed by *"This is not an empty
queue"*. The worst failure the ticket implied — an outage reading as a clear
queue — did not exist. What was missing was a workaround once the signal fired,
not the signal.

**Not looked for, and the actual defect:** `FetchWithRegistry` called
`exec.CommandContext(ctx, "gh", …)` directly. No injection point, therefore no
test — the package's four tests all covered pure functions. **The one path
observed failing in production was the only path in the package nothing
exercised**, and that was true whichever transport it spoke.

## The lesson

A missing seam does not read as missing coverage. The package looked well
tested: four tests, a clean domain, careful comments about failure modes. The
gap was invisible because the untested part was the part that *reaches outside* —
and reaching outside is exactly what cannot fail in a unit test, so nothing ever
drew attention to its absence.

The correlation is not a coincidence. The code that talks to the network, the
filesystem or another binary is simultaneously the code most likely to fail in
production and the code that is hardest to test *without a seam* — so the two
properties select for the same lines. Wherever a package calls `exec`, `http` or
`os` directly from a function with real logic in it, that function is both the
riskiest and the least covered.

**So when a production failure points at a subsystem, ask what made that path
untestable before asking what made it fail.** The transport was the presenting
symptom and the smaller half. Framing the work as "migrate to REST" would have
justified it on an outage observed exactly once, and left the real defect in
place regardless of which API it called.

## What was done

The seam came first and the transport followed:

```go
type ghRunner func(ctx context.Context, args ...string) ([]byte, error)

func FetchWithRegistry(ctx context.Context, repo string, reg Registry) ([]Status, error) {
	return fetchWith(ctx, execGH, repo, reg)
}
```

The exported signature is unchanged; `fetchWith` takes the runner. Eight tests
now drive the fetch path with canned payloads, no network and no `gh` on PATH.

Three consequences of the transport change were only visible once it was
testable, and each became a test:

- **`normaliseLogin` was promoted from defensive to load-bearing.** The registry
  declares `github-actions`; REST returns `github-actions[bot]`. Under GraphQL
  the `[bot]` fold was insurance. Under REST every match depends on it — and
  removing it does not error, it silently empties the queue.
- **REST paginates on two axes where GraphQL paginated on none.** A truncated
  pull-request list makes a PR go *missing*, which is visible. Truncated
  comments make the verdict *wrong*: an untriaged PR can read as triaged. Both
  refuse now; the second matters more.
- **1 + N met a five-second budget.** The session-start probe runs on every
  session start under that bound. The fan-out is capped at 8 and the concurrency
  test asserts *observed peak* rather than wall-clock, because a timing
  assertion on a loaded runner gets loosened until it means nothing.

Measured after: 0.94 / 0.96 / 0.95 s against the live repository, and output
byte-identical to what the GraphQL implementation printed for the same repo
earlier the same session.

## Related

- Lesson 259 — a justification outlives its mechanism. Applied here as a sweep of
  the *prose*, which caught a fixture error string (`"gh pr list: exit status 4"`)
  this change had made unproducible.
- #1033 — the login-spelling trap that this change moved onto the critical path.
- `docs/adr/adr-020-tooling-cli-go-convergence.md` — the CLI convergence this sits inside.
