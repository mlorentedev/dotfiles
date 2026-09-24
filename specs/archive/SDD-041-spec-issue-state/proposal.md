---
id: "SDD-041-spec-issue-state"
type: spec
status: archived # draft | implementing | verifying | archived
created: "2026-09-23"
issue: "mlorentedev/dotfiles#1087"   # repo#NNN — GitHub issue / Project item that tracks this spec
tags: [spec, proposal, sdd, doctor, archive-on-merge]
template_version: "1.0"
---

# SDD-041-spec-issue-state

> Epic #1625 (HARNESS-138), item **W1.1**. Acceptance reference: AC-1.1.

## Why

The archive-on-merge gate (SDD-038) fires only when a merged PR carries a
closing keyword (`Closes #N`) for an issue that an active spec declares. An
issue closed any other way (by hand, or by a PR without the keyword) never
enters that path, so its spec stays active forever. On 2026-09-22 that left
**16 active `dotfiles` specs tracking a CLOSED issue**. Fifteen are linked
through `issue:` frontmatter. One, GOV-004, carries its link only as prose on
line 85 (`- GH issue: [#673](…)`), which a frontmatter-only scan cannot see.
A stale active spec is an alibi for the spec gate, because any large PR can
touch ten lines of a shipped spec and pass. A gate evaluated at PR time
cannot observe a close that has no PR. The net therefore has to be
**detective**: something that asks the forge, on demand, whether every active
spec still tracks an open issue.

## What

1. **An issue-link resolver in `cli/internal/spec`.** It reads `proposal.md`
   and returns the spec's tracking issue and where the link was found.
   - The `issue:` frontmatter field is authoritative. Accepted forms:
     `owner/name#N`, `name#N` (the owner is taken from the home repo) and
     `#N` (the home repo itself).
   - **Prose fallback, as a migration aid only.** The resolver accepts a
     *labelled* line: after list, quote and bold markers, the label is
     `issue`/`issues`, `GH` or `GitHub`, optionally prefixed by a qualifier
     (`GH issue`, `hive issue`, `github_issue`). It takes the first reference
     on that line whose **owner is the home repo's owner**. A label that
     carries `upstream`, `related` or `sister` is never a tracking link.
     These are measured shapes: `Upstream issue: anthropics/claude-code#59870`
     (BUG-004) and `Related upstream issue: modelcontextprotocol/python-sdk#2610`
     (hive HIVE-104) point at projects we do not own. An unlabelled `#N` in
     running text (`(GH #197)`, `PR #121`) is never a link.
2. **`dotf spec audit`.** Scoped to the current checkout, so it works in any
   repo that carries `specs/`, including `hive`. It classifies every active
   spec (`specs/*/`, excluding `specs/archive/`):

   | State | Severity |
   |---|---|
   | tracking issue OPEN, linked in frontmatter | ok |
   | tracking issue CLOSED | **FAIL** (zombie) |
   | reference resolves to no issue (404), or points at a pull request | **FAIL** (unresolvable) |
   | `issue:` present but malformed | **FAIL** (unresolvable) |
   | linked only in prose, issue open | WARN (normalise to frontmatter) |
   | no link at all | WARN (unlinked) |
   | the forge could not answer (auth, network, rate limit) | **unanswerable** |

   It exits non-zero on any FAIL **or** any unanswerable lookup. This is the
   same contract as `dotf pr triage-queue`: a question that could not be
   answered is never reported as a clear one.
3. **A `spec-issue-state` section in `dotf doctor`** (the full sweep, not
   `--quick`). It is a thin wrapper over the same audit, run against the
   dotfiles checkout. FAIL maps to FAIL and WARN to WARN. An unanswerable
   lookup is a WARN that names the cause, and `gh` absent from PATH is a
   Skip. Neither is ever a PASS.
4. **Normalise GOV-004.** Add `issue: "mlorentedev/dotfiles#673"` to its
   frontmatter, and add a test that fails if any active spec in this repo is
   linked only in prose. That test needs no network.

Lookups use the REST endpoint (`gh api repos/O/R/issues/N`), which has its
own 5,000/h budget. A 404 there cleanly means "no such issue", as opposed to
"could not ask". A bounded worker pool runs the lookups: about 40 specs per
sweep, sequential, would add seconds to every doctor run.

## Out of scope

- **`scripts/check-spec-gate.sh`, the closing-keyword path.** It stays
  unchanged. It already handles the case it can see, a PR that closes an
  issue, and preventive gating of manual closes is impossible at PR time.
- **Dispositioning the zombies it reports.** That is the W1.4 sweep (#1626),
  which runs `dotf spec audit` as its acceptance command. Until the sweep
  lands, `dotf doctor` in dotfiles is **red on purpose** (16 FAILs), just as
  W2.4 is designed to be red until W2.3. `dotf doctor` gates no CI context,
  so the red state blocks nothing.
- Normalising `hive`'s prose-linked specs (FEAT-015, HIVE-119). They live in
  another repo and belong to W1.4.
- kubelab's zombies (decision D-7 on #1625).
- Wiring the audit into CI or a schedule.

## Risks / open questions

- **Prose false positives.** Mitigated by requiring a label, rejecting
  `upstream`/`related`/`sister`, and filtering on the owner. Every measured
  shape is a test fixture. The fallback reports WARN, never ok, so it cannot
  become a second source of truth.
- **API cost and latency.** REST with a bounded pool, and only in the full
  doctor sweep, never in `--quick` (the SessionStart hook).
- **An unanswerable lookup read as clean.** This is the failure mode the audit
  exists to prevent. The audit exits non-zero, doctor never reports PASS, and
  both are tested.
- **Doctor red until W1.4.** This is intended and stated above.

## Acceptance criteria

- [x] **AC1** — `gorun ./internal/spec/... 'Archive.*Issue|IssueState'`
  passes with at least one test run. The tests cover every frontmatter form,
  every measured prose shape including the negative ones, frontmatter taking
  precedence over prose, the full classification table, and REST output
  parsing (open, closed, pull request, 404, other error).
- [x] **AC2** — A fixture spec whose issue is CLOSED makes the audit report
  FAIL and exit non-zero. A fixture whose lookup fails exits non-zero with
  "unanswerable", never with a clean result.
- [x] **AC3** — `dotf doctor 2>&1 | grep -i 'spec-issue-state'` shows the
  section. On today's tree it FAILs and lists every active spec whose issue is
  closed: 14 on 2026-09-23, GOV-004 included. The synthesis counted 16; CLI-078
  and CLI-080 left the set when their shared issue #1596 was reopened. A doctor unit test pins the mapping: ok →
  PASS, FAIL → FAIL, unanswerable → WARN, `gh` absent → Skip.
- [x] **AC4** — GOV-004 carries `issue:` frontmatter, and
  `TestIssueStateNoActiveSpecIsProseLinked` passes. The test fails when a
  prose-only fixture is added.
- [x] **AC5** — `dotf spec audit`, run in `hive`, reports HIVE-267 as a
  zombie, and resolves FEAT-015 and HIVE-119 through their prose links (both
  of those issues are closed too, so they are zombies the frontmatter-only
  count missed).

## References

- Epic: mlorentedev/dotfiles#1625 §3 W1.1, §5.2 AC-1.1. Addenda comment: fixture list.
- SDD-038 (archive-on-merge), `scripts/check-spec-gate.sh:130-320`.
- Prior reconcile: #1086 (45 → 30 active specs), the same failure class.
- Exit-code contract: `cli/internal/prtriage`, `dotf pr triage-queue`.

<!-- archived 2026-09-23 — PR: https://github.com/mlorentedev/dotfiles/pull/1630 -->
