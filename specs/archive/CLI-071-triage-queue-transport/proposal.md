---
id: "CLI-071-triage-queue-transport"
type: spec
status: archived # draft | implementing | verifying | archived
created: "2026-09-02"
issue: "mlorentedev/dotfiles#1454"   # repo#NNN — GitHub issue / Project item that tracks this spec
tags: [spec, proposal]
template_version: "1.0"
---

# CLI-071-triage-queue-transport

## Why

<!-- from issue #1454: CLI-071: dotf pr triage-queue dies with GraphQL, so the command that binds Definition of Done §4 is unavailable exactly when gh is degraded -->

`dotf pr triage-queue` is the mechanism the always-injected pr-stewardship
doctrine names by hand: run it at session start, and again before reporting any
PR work complete. Its fetch path calls `exec.CommandContext(ctx, "gh", …)`
directly at `fetch.go:60`, with no injection point — so `FetchWithRegistry` is
unreachable from a test, and it has none. The four tests in the package cover
only the pure functions. **The one path observed failing in production on
2026-09-02 is the one path nothing exercises**, and that is true whichever
transport it speaks.

The transport is the second problem and the smaller one. `gh pr list --json` is
a GraphQL call, and GraphQL refused every write and list for a stretch on
2026-09-02 while REST served the identical data throughout. The command binds a
doctrine, so a single point of failure it does not need is worth removing — but
"unavailable" was overstated in the ticket and is corrected in its body: both
consumers already fail loud, and neither reads an outage as an empty queue.

## What

After this change:

1. `FetchWithRegistry` takes its GitHub access through an injected seam rather
   than reaching for `exec` itself, so the fetch path is drivable from a test
   with canned payloads and no network.
2. The queue is assembled from REST only — `GET /repos/{o}/{r}/pulls` for the
   open list, `GET /repos/{o}/{r}/issues/{n}/comments` per pull request — so a
   GraphQL refusal no longer takes the command down.
3. The session-start probe stays inside its five-second budget with the per-PR
   fan-out bounded and measured, not assumed.
4. Rendered output is byte-identical for identical input: this is a transport
   and testability change, not a behaviour change.

## Out of scope

- **Distinct exit statuses for "pending" vs "could not answer."** `pr.go:49-51`
  documents the shared status as a reasoned decision. Refuting that argument is
  its own ticket, not a rider on this one.
- **The triage algorithm.** `Evaluate`, `reviewOutput` and `lastTriage` are
  untouched; if their behaviour changes, this change is wrong.
- **The `pr-review-triage` skill and the reviewer registry.** This package lists
  and never applies, and nothing here may erode that.

## Risks / open questions

- **R1 — the session-start latency budget (must be resolved before merge).**
  `mem.go:204` bounds the probe at 5s and it runs on every session start. Serial
  1+N against ten open PRs is eleven round-trips and can exceed it. Trading an
  intermittent unavailability for a predictable degradation on the hot path
  would be a net loss, so the fan-out is bounded and the latency measured.
- **R2 — two pagination axes instead of none.** GraphQL returned PRs and their
  comments in one response. REST paginates both. The old `parseWire` refused
  when a full page of PRs came back rather than silently truncating; that
  refusal must survive on **both** axes, because a PR whose comments were
  truncated yields a *wrong* verdict rather than a visibly missing one.
- **R3 — `normaliseLogin` moves onto the critical path.** The registry declares
  `github-actions`; REST returns `github-actions[bot]`. `normaliseLogin` already
  folds the two, but under GraphQL that was defensive and under REST it is
  load-bearing on every match. Removing it would empty the queue silently — the
  exact failure this package exists to prevent (see #1033).
- **R4 — field names differ.** GraphQL `url` is REST `html_url`; GraphQL
  `comments[].author.login` is REST `user.login`; `createdAt` is `created_at`.
  A silent zero-value here degrades a timestamp comparison into a wrong answer,
  not an error.

## Acceptance criteria

- [x] **AC1 — the fetch path is tested.** `FetchWithRegistry` runs end-to-end in
      a test against an injected fake returning canned REST payloads, with no
      network and no `gh` on PATH. Removing the seam makes the test fail.
- [x] **AC2 — no GraphQL remains.** The package invokes only REST endpoints; a
      test asserts the requests the seam receives, so a reintroduced `pr list`
      is caught mechanically rather than by review.
- [x] **AC3 — the bot-login fold is pinned by a REST-shaped payload.** A fixture
      carrying the literal `github-actions[bot]` and `coderabbitai[bot]` resolves
      against a registry declaring the bare logins. Deleting `normaliseLogin`
      fails this test.
- [x] **AC4 — truncation still refuses on both axes.** A full page of pull
      requests, and a full page of comments on any one of them, each produce an
      error rather than a short answer.
- [x] **AC5 — the session-start budget holds, measured.** With the fan-out
      bounded, a fake with a per-call delay representing ten open PRs completes
      inside the 5s context, and the test fails if the calls are issued serially.
- [x] **AC6 — the renderers see identical values.** Both renderers are pure
      functions of `[]Status`, and that type is untouched, so "output unchanged"
      reduces to "every consumed field is populated identically". A fixture pins
      all six — `Number`, `Title`, `URL`, `Reviewer`, `At`, `Reason` — because
      REST spells three of them differently (`html_url`, `user.login`,
      `created_at`) and a mismapping yields a hollow entry, not an error.

## References

- Bitácora board: `mlorentedev/dotfiles#1454` (see the `issue:` frontmatter field)
- `docs/adr/adr-020-tooling-cli-go-convergence.md` — the CLI convergence this package sits inside
- Prior art for the seam: `cli/internal/doctor` `System` (`newSys(env, onPath, cmdOut)`)
- #1033 — the login-spelling trap this change moves onto the critical path
