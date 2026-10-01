---
tags: [spec, verification, templates]
created: "2026-09-25"
---

# Verification - TOOL-023-pr-agent-incremental-push-review

## Evidence

Original acceptance criteria (`proposal.md`), landed in #1757 at `7ca394c` on `origin/main`
(`git log origin/main --oneline -- scripts/pr-agent-push-gate.sh` -> `7ca394c ci(pr-agent):
review a push incrementally, once three new commits follow the last review (#1757)`), then
hardened by the follow-up below (branch `fix/push-gate-trusted-baseline`, triage on #1757
comment 5851151835):

- [x] AC1 -> `7ca394c` / `bats tests/pr-agent-push-gate.bats`: run=false below 3 new
  non-merge commits, run=true at/past it, previous review found as PR-Agent does, merge
  commits ignored, bot push skipped, runs with no previous review or unreadable input.
- [x] AC2 -> `7ca394c` / `bats tests/pr-agent-config.bats`: gate wired on `synchronize`
  only, PR-Agent and the guard skip on `run=false`. Refined by follow-up f7 below:
  `push_commands` is no longer a bare `["/review -i"]` literal but a template driven by
  the gate's `mode` output.
- [x] AC3 -> `7ca394c` / `bats --filter 'marker' tests/pr-agent-config.bats` (5 cases,
  exit 0): the registry declares `## Incremental PR Reviewer Guide`, the guard accepts
  every declared marker.
- [x] AC4 -> `7ca394c`: `test -f docs/adr/adr-040-automatic-reviews-bounded-per-push.md`
  (present) && `grep -q 'draft' harness/enforced/pr-stewardship.md` (present).

Follow-up findings (merged in #1773, `94c9e6b`):

- [x] f5 (CWE-345) -> `bats --filter 'CWE-345|forged' tests/pr-agent-push-gate.bats`, exit
  0, 2/2 passing: a marker from a non-bot login is never a baseline, and one newer than a
  real `github-actions[bot]` review forces `mode=full`.
- [x] f6 (rebase blind spot) -> `bats --filter 'head_sha' tests/pr-agent-push-gate.bats`,
  exit 0, 5/5 passing: a `last_run.head_sha` in the state block counts commits by position
  (immune to a rebase preserving author dates), an absent sha forces `mode=full`, a state
  block without a sha falls back to the date count, and unreadable commits on that path
  still fail open.
- [x] f7 (mode selection) -> `bats --filter "runs /review -i only" tests/pr-agent-config.bats`
  and `actionlint .github/workflows/pr-agent.yml`, both exit 0: `push_commands` reads
  `steps.push_gate.outputs.mode` and defaults to the full command unless it is exactly
  `incremental`.

## Test status

- `~/.local/bin/bats tests/pr-agent-push-gate.bats` -> exit 0, 22/22 (each case run under
  both bash and zsh by the file's own `_gate` helper; 7 new cases add the CWE-345,
  head_sha-position, sha-absent, and missing-head_sha-fallback behavior).
- `~/.local/bin/bats tests/pr-agent-config.bats` -> exit 0, 48/48 (1 test rewritten to
  validate the `push_commands` template's two branches instead of one literal).
- `~/.local/bin/bats tests/pr-agent-push-gate.bats tests/pr-agent-config.bats` (combined,
  as CI runs them) -> exit 0, 70/70.
- `~/.local/bin/shellcheck scripts/pr-agent-push-gate.sh` -> exit 0, no findings.
- `bash -n scripts/pr-agent-push-gate.sh` and `zsh -n scripts/pr-agent-push-gate.sh` -> exit
  0 on both.
- `actionlint .github/workflows/pr-agent.yml` -> exit 0, no findings.
- No regressions: yes. All 15 pre-existing cases in `pr-agent-push-gate.bats` and all 47
  pre-existing cases in `pr-agent-config.bats` (bar the one rewritten for f7, which encodes
  the same intent against the new shape) pass unchanged.
- Deferred to the next PR with a live push (cannot be exercised offline): a below-threshold
  push on a real PR stays green with no review; `dotf pr triage-queue` correctly reads an
  incremental review comment once one exists in the wild; a PR-Agent timeout on a gated push
  still fails the "no review published" guard. None of these are new to this follow-up — the
  same three were already deferred by #1757 and remain open.

## Archive pass (2026-09-30)

Re-verified on `main`:

```
$ bats tests/pr-agent-push-gate.bats tests/pr-agent-config.bats
1..74   all ok
$ bats --filter <each features.json filter> ...
push 3, marker 6, CWE-345|forged 2, head_sha 5, "runs /review -i only" 1   (none empty)
```

Each `bats --filter` in `features.json` matches at least one case. A filter that matches
nothing runs zero tests and still exits 0.

**One of the three deferred live checks now has evidence.** On #1850, a one-commit push
(`0206468`) after a full review of `93139a1` produced this, in pr-agent run 36739357766:

```
run=false
reason=1 new commit(s) since the reviewed commit 93139a1620aec016a812acd55c878b10277ffab6, below the threshold of 3; comment /review to ask for one now
```

PR-Agent and the no-review guard were skipped, and the run concluded `success`. The
count came from the reviewed sha, which is f6's position path, not the date fallback.
The other two deferred checks have not occurred in the wild yet:
- `dotf pr triage-queue` reading an incremental review;
- a timeout on a gated push.

Both remain covered only offline.

The upstream finding is ticketed as #1889 for the owner.

## Decisions made during implementation

- The baseline computation now runs as two separate `jq` calls sharing one predicate
  (`JQ_IS_REVIEW`) — one restricted to `github-actions[bot]`, one restricted to everyone
  else — rather than one query with an author branch inside it, to keep each call
  independently readable and to match the file's existing style of one `jq` call per
  question asked.
- `mode` is only ever a third `decide()` line on a `run=true` verdict. A `run=false` verdict
  never starts PR-Agent, so a mode for it would describe a step that never runs; keeping it
  absent there also means the pre-existing "two key=value lines" test needed no change.
- The state block's JSON is extracted by splitting the body on the literal marker text and
  trimming the trailing `-->`, not by a brace-balancing regex: PR-Agent always appends the
  block last (`_get_markdown_text`), so everything after the marker up to the final `-->` is
  the block, including any nested `{`/`}` inside it (a real one nests a `findings` array).
- **Disclosed limitation, not fixed here (see `proposal.md`'s Follow-up section):**
  `last_run.head_sha` only ever appears in a FULL review's comment; PR-Agent v0.45.0 turns
  its finding-state machinery off entirely for an incremental run
  (`_review_finding_state_enabled` returns `False` when `self.incremental.is_incremental`,
  read from the pinned source at `f3b385e`; no real incremental comment exists yet on this
  repo to cross-check against). So once a PR's latest bot review is itself incremental, this
  gate is back on date-based counting — and the original rebase blind spot — until the next
  full review. Not widened here: doing so would mean trusting a field PR-Agent does not
  populate for that case, which is a guess, not a fix.
- **A fourth, narrower disclosure, found reviewing this fix rather than in the original
  triage:** SHA-position counting (f6) can disagree with PR-Agent's own date-based
  `get_commit_range` when every commit after the reviewed sha is authored at or before the
  review's `created_at` (a cherry-pick with a preserved old date, not a plain rebase, which
  lands in `mode=full` already via f6's absent-sha check). PR-Agent then computes an empty
  incremental range and publishes nothing — loudly caught by the pre-existing "no review
  published" guard (#1107), never silently. ADR-040 D2 is amended in this PR (was: "the gate
  mirrors its rule, so it never starts a run PR-Agent would then decline" — now states the
  exception and what catches it) rather than left to read as an absolute it no longer is.
- **Upstream finding, not filed by this agent (owner's call):** `get_previous_review`
  (`github_provider.py`) picks its baseline comment by body text alone, walking the comment
  list by position, with no author check — the same class of bug as f5, in the tool this
  gate mirrors. `mode=full` on a newer non-bot marker works around it locally; it does not
  fix it upstream.

## Promotion candidates

- [x] Lesson for the repo's `docs/lessons/`? no: this follow-up applies a pattern this repo
  already has lessons for (a check that predicts another component uses that component's own
  rule, lesson 303; a guard that skips is a guard that passes, lesson 287). No new defect
  class.
- [x] ADR-worthy decision for the repo's `docs/adr/adr-XXX.md`? no: ADR-040 already records
  the incremental-review decision; this refines its baseline-trust and rebase handling
  in-place rather than deciding something new.
- [x] New pattern candidate for `00_meta/patterns/`? no: specific to this one script.

## Archive checklist

- [ ] `proposal.md` frontmatter set to `status: archived`
- [ ] Folder moved: `specs/TOOL-023-pr-agent-incremental-push-review/` -> `specs/archive/TOOL-023-pr-agent-incremental-push-review/`
- [ ] Bitácora board ticket for this spec moved to Done / closed with PR link (ADR-018)
- [ ] Promotions above executed (if any)

Not archived by this PR, per instruction: this follow-up hardens a not-yet-closed spec and
leaves the archive decision to the owner after independent review.
