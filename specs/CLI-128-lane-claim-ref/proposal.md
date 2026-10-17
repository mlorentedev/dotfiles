---
id: "CLI-128-lane-claim-ref"
type: spec
status: draft # draft | implementing | verifying | archived
created: "2026-10-16"
issue: "mlorentedev/dotfiles#2551"   # repo#NNN — GitHub issue / Project item that tracks this spec
tags: [spec, proposal]
template_version: "1.0"
wip_override: "owner decision on #2551 (option B, 2026-10-16); one override open at a time, the next spec init waits for this one to archive (11 active, limit 10, 2026-10-16)"
---

# CLI-128-lane-claim-ref

> Atomic claim of a queued issue on a git ref (epic #2546, ADR-050 decision 2).

## Why

<!-- from issue #2551: CLI-128: atomic lane claim on a git ref -->

Today the claim on a queued issue is a forge assignee (Standing Order #8), which a session reads and then writes: two sessions that read "free" at the same time both start the lane. #2576 (closed by #2600 and #2611) left the same shape open in two places, the find-then-create of a branch and the read-modify-write append to `info/exclude`. A claim must be one atomic step that either wins or says who holds it, and it must live in git so that a session that can push can coordinate with nothing else running (ADR-050).

## What

A claim is the ref `refs/dotf/claims/<issue>` (form (a), accepted by the owner on #2551, conditional on the probe below), pointing at a claim commit: an empty-tree commit whose message records the holder, the host and the time. Taking it is `git push --force-with-lease=<ref>: <claim-sha>:<ref>`; the empty expected value makes the server refuse a ref that already exists.

- `lane.Claim` (new, `cli/internal/lane/claim.go`): `Acquire`, `Holder`, `Release` over a small `Git` seam, so the production path and the tests run the same code.
- A loser gets a typed `ClaimedError` naming the holder and when it claimed, never a generic push failure.
- `dotf lane next` claims the first free queued issue before it creates anything, and moves to the next free item when it loses. A loser starts nothing, and a claim that cannot be read or taken starts nothing either (fail closed). A branch whose worktree already exists is not claimed again: it was claimed when it was started, and a second run must not read its own claim as a loss.
- `lane next` does not set the forge assignee. `Next` skips assigned issues, so assigning would turn a second invocation for the same item into a second lane; the claim ref is the record, and `lane status` shows it.
- The holder is the checkout (`user@host:<worktree>`), not the process: `lane claim` and a later `lane release` from one worktree are the same claimant, two sessions in two worktrees are not.
- `dotf lane claim <issue>` and `dotf lane release <issue>` expose the same primitive to every harness, the autopilot skill included: one implementation, no Claude-only path.
- `dotf lane claims` shows every claim with its holder and age, and flags one whose issue is closed. A stale claim is a human decision, as adoption is (D2): `release --force` is the resolving command, never an automatic expiry.
- The claim push skips the repository's pre-push hooks: it publishes no branch, and the spec gate reads a branch.

## Out of scope

- Editing the autopilot skill or other harness files (frozen paths); this change ships the verbs they call.
- Automatic expiry or takeover of a stale claim.
- Landing, the queue order, and the iris backend.
- Automatic release of a claim on land or park (see `verification.md`).

## Risks / open questions

- A forge may refuse a non-branch ref or a custom namespace. Resolved 2026-10-17: the opt-in probe test (`DOTF_CLAIM_PROBE_REMOTE`) passed on GitHub and on Gitea, and ADR-050 decision 2 is decided. A forge not yet probed is probed by the same test.
- A dead session leaves a claim that nobody releases. The age is shown and a human releases it; an expiry would be a second source of truth.
- A forge outage or a missing push right stops claims: `lane next` fails closed with the reason and starts nothing, as an outage already stops CI.
- Reading the holder needs the claim commit's object: `Holder` fetches the ref into a private namespace and reads the message, so the claim carries its own identity with no extra store.

## Acceptance criteria

- [x] AC1: Acquire on a free issue wins; a second Acquire of the same issue, with the same or another sha, is refused (real git, a local bare remote).
- [x] AC2: the refusal is a `ClaimedError` that names the holder and the time of the claim, read from the claim commit.
- [x] AC3: N concurrent Acquires of one issue produce exactly one winner.
- [x] AC4: `lane next` claims before it starts, skips to the next free item on a loss and starts nothing for the loser or when the claim cannot be taken.
- [x] AC5: `lane release` deletes a claim its caller holds, refuses another holder's without `--force`, and a second release is a no-op.
- [x] AC6: the claim ref is not fetched by a default clone and appears in no branch list.
- [x] AC7: the ref form is verified by the probe on GitHub and on Gitea and ADR-050 decision 2 is amended from open to decided.

## References

- Bitácora board: mlorentedev/dotfiles#2551 (epic #2546)
- Related ADR: `docs/adr/adr-050-one-lane-contract-for-every-repo-agent-and-forge.md` (decision 2)
- Lessons: 451 (a reconcile must see the work it started), 454 and 457 (one oracle, exercised through the production wiring)
