---
id: "GUARD-004-triage-loop-closure"
type: spec
status: archived # draft | implementing | verifying | archived
created: "2026-08-20"
issue: "mlorentedev/dotfiles#1099"   # repo#NNN — GitHub issue / Project item that tracks this spec
tags: [spec, proposal]
template_version: "1.0"
---

# GUARD-004-triage-loop-closure

## Why

<!-- from issue #1099 -->

GUARD-002 made a green check mean *"a review happened"*. `dotf pr triage-queue`
answers the other half — *did anyone act on it?* — and both halves of that second
question were missing.

**Nothing wrote the marker.** The queue decides a PR is still pending by reading
a `## Review triage` comment back off it. Measured 2026-08-19: zero occurrences
of that marker in `harness/skills/pr-review-triage/SKILL.md`, in its deployed
copy, or on any pull request. The skill printed its table to the conversation and
stopped, so the queue could never drain — #1085 sat in it, untriaged, from 01:36.

The contract was already written down and only half-honoured.
`harness/review-attestation.json` says: *"One string, two consumers, declared
once. The skill writes it and the queue reads it, and putting it in either would
have made the pair a two-file agreement nobody checks — the failure class this
repo spent 2026-08-17 into the 18th cataloguing."* That is a precise description
of the state it was in. The CLI's help text made it worse by asserting the skill
records the table — a claim about behaviour that did not exist, shipped.

**Nothing ran the query.** `triage-queue` appeared in no `settings.json`, hook or
workflow. Its own help states the constraint: *"no push channel reaches an agent
session — so the mechanism an agent can use is a query it runs at a checkpoint"*.
There was no checkpoint, so it ran only when an agent remembered — the opposite
of this repo's rule that determinism comes from code, not memory.

## What

The writer half: the `pr-review-triage` skill posts its disposition table on the
PR under the marker the registry declares, **including when there was nothing to
dispose of**.

The wake-up half: `dotf mem session-start` asks the queue, on every harness.

## Design decisions

**The empty case is recorded, not skipped.** *"CI green, no review findings"* is
a disposition. An unrecorded empty triage leaves the PR in the queue forever,
which is indistinguishable from nobody having looked — and a queue that never
drains is one nobody reads. The queue re-opens by itself the moment a reviewer
speaks again, so recording early costs nothing.

**Native mechanism where one exists; an instruction floor everywhere else.**
This follows ADR-032 rather than inventing a posture. Claude has a session-time
execution surface, so the probe is wired into it — that is the native leg, not a
Claude-only solution. Both CLI assembly paths carry the section (the agnostic
`Brief` and the Claude adapter) so no second implementation is needed when
another harness gains a surface.

But measured 2026-08-20, **nothing invokes `dotf mem session-start --format`**:
opencode and agy consume static instruction files and expose no session hook. A
capability with no consumer is the same shape as the reader-with-no-writer this
spec exists to fix, one layer up. So the floor is an **instruction**, carried by
the always-injected `pr-stewardship` doctrine: it names `dotf pr triage-queue`
and the `## Review triage` marker explicitly, and renders through
`compile-harness.sh` into `AGENTS.md` — the one artifact every harness reads.

For a harness with no session-time execution surface, instruction-level
determinism is the maximum the platform offers; there is no hook to wire. That
is ADR-032's "enforcement is presence + dispatch" applied honestly rather than
claimed. Wiring native hooks per harness is a follow-up leg, one per harness that
has a surface to wire.

The same render reaches the reviewer for free: `.pr_agent.toml` declares
`repo_context_files = ["AGENTS.md", ".claude/CLAUDE.md"]`, so PR-Agent reads the
same doctrine the agents do, from the same SSOT.

**Three states, and the middle one is why this is careful.** *Pending* names the
PRs. *No reviewer registry* is silent: most repositories do not run this loop,
and reporting "could not compute" in every one of them would train the reader to
skim the line, which is only worth anything while it stays rare. *Present but
unanswerable* is loud and denies the empty reading outright. That last rule is
inherited rather than invented — `pr triage-queue` exits non-zero when it cannot
answer precisely so an unanswerable queue is never read as an empty one, and
swallowing the error one layer up would have rebuilt the blind spot.

**Latency is bounded at five seconds.** Session start is a budget nobody
volunteered for; an unreachable API must cost one visible message, not a hung
shell.

**One code path for one question.** The gh query and its domain conversion move
into `prtriage.Fetch`, consumed by both the command and the brief. A second copy
would be a second place for the conversion to drift.

**Fixed at the vault SSOT.** The skill deploys to every harness, so the same
reader-with-no-writer gap observed downstream in kubelab closes with it.

## Acceptance criteria

- [x] The skill records `## Review triage` on the PR, including the empty case.
- [x] The marker is quoted from `harness/review-attestation.json` rather than
      restated, so it keeps one definition and not two.
- [x] `dotf mem session-start` asks the queue on the agnostic path and the Claude
      adapter both.
- [x] The checkpoint is stated as doctrine in the vault SSOT and renders through
      `compile-harness.sh --refresh` into `AGENTS.md` and `ai/claude/CLAUDE.md`,
      so every harness receives it whether or not it can execute anything at
      session start — and the PR reviewer receives it too, via
      `repo_context_files`.
- [x] The package comment no longer claims consumers the agnostic path does not
      have, and records the deploy-time staleness constraint for whoever builds
      the first one.
- [x] A repository with no reviewer registry renders no section at all.
- [x] A repository with a registry but no reachable `gh` reports the failure and
      states that it is not an empty queue.
- [x] The gh call is bounded by a context deadline.
- [x] `go build`, `go vet`, `GOOS=windows go vet`, `go test ./...`,
      `golangci-lint run` at the pinned version, and the full bats suite are green.

## Risks / open questions

The section adds a network call to session start. Bounded at five seconds and
skipped entirely where no registry exists, so the common case outside this
repository pays nothing. If the budget proves too generous in practice it is one
constant.

`dotf spec init` scaffolded three files rather than four (no `features.json`) —
that is #1076, filed and open, not caused here.
