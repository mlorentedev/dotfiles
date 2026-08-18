---
tags: [spec, verification, templates]
created: "2026-08-15"
---

# Verification - HARNESS-072-pr-stewardship

## Evidence

| AC | Proof | Where |
|---|---|---|
| AC1 | `harness/enforced/pr-stewardship.md` written by `--refresh` from `pattern-change-lifecycle.md#pr-stewardship` | `334c1a7`, vault `2e351f1a` |
| AC2 | positive grep for *"the disposition, not the waiting"* on all five surfaces: `AGENTS.md`, `ai/claude/CLAUDE.md`, and after `--deploy` `~/.claude/CLAUDE.md`, `~/.gemini/GEMINI.md`, `~/.codex/AGENTS.md` — 1 hit each. Caps hold: GEMINI 6503/12000, codex 6503/32768 | `334c1a7` + session run |
| AC3 | region states *"What binds is the disposition, not the waiting"* and *"its instruction wins"* | `harness/enforced/pr-stewardship.md` |
| AC4 | region states *"The trigger is the archive gate and nothing wider"*; no spec-folder trigger | `harness/enforced/pr-stewardship.md` |
| AC5 | region states *"A notice that no review ran leaves the PR unreviewed"* | `harness/enforced/pr-stewardship.md` |
| AC6 | skill description now triggers on *"whichever lands later"*; body already `gh`-only | vault `35d93f0a`, record `fdbe0b4` |
| AC7 | `./scripts/compile-harness.sh --check` → exit 0, "no harness drift" | session run |
| AC8 | `bats tests/compile-harness.bats -f 'HARNESS-072'` → 3/3; guard observed **red before green** twice | `bf2eda9` |

## Test status

- `~/.local/bin/bats tests/compile-harness.bats` → **47/47 ok**, 0 failures.
- `~/.local/bin/shellcheck scripts/compile-harness.sh` → clean. `bash -n` → clean.
- `./scripts/compile-harness.sh --check` → exit 0.
- All eight `features.json` verification commands executed this session → 8/8 exit 0.

**Red observed before green, on real state and not a fixture:**

```
$ ./scripts/compile-harness.sh --check      # before declaring pr-sizing's exclusion
[GAP] enforced region "pr-sizing" reaches neither surface "AGENTS.md" nor an opt_out for it
[GAP] enforced region "pr-sizing" reaches neither surface "ai/claude/CLAUDE.md" nor an opt_out for it

$ ./scripts/compile-harness.sh --check      # with pr-stewardship wired to AGENTS.md only
[GAP] enforced region "pr-stewardship" reaches neither surface "ai/claude/CLAUDE.md" nor an opt_out for it
```

The second is the case that justifies the guard's shape: an orphan check would
have passed it, because `pr-stewardship` was in use on another surface.

## Decisions made during implementation

- **The obligation is the disposition; the timed watch is only a default.** The
  first draft made the watch the rule, which would have overridden a standing
  user preference in another project ("do not poll CI, I notify") by a route the
  user never chose. A project signal now *satisfies* the region instead of
  contradicting it. Rejected a hardening of this opt-out ("only a signal written
  in the project's instructions counts") — it would have ranked a file above a
  live instruction from the user, a worse bug than the lawyering it prevented.
- **The adversarial-review trigger stays at the archive gate.** "Touches
  `specs/<id>/`" caught nearly every docs PR in a repo whose `tasks.md` is ticked
  as work proceeds.
- **AC2's original verification was vacuous and is the reason AC8 exists.**
  `--check` renders its expected side from the target's own `inject` list, so a
  region missing from that list is missing from both sides of the diff and the
  surface reports `OK`. The spec had named that check as the mitigation for
  precisely the risk it cannot see. Found by a second session reviewing this
  worktree from the outside; verified against the source before acting on it.
- **The coverage guard was taken in this PR rather than ticketed.**
  `feedback_incident_to_guard` requires the assertion in the same PR, and the
  fix-or-ticket escape does not apply to a spec still in `draft`.
- **`--check` is offline by design (ADR-013) and cannot see a record trailing its
  vault source.** Six records were found stale only by running `--refresh`. Same
  shape as the coverage gap — a check answering a narrower question than the one
  asked of it — but not fixed here: CI has no vault, on purpose.

## Promotion candidates

- [x] Lesson for the repo's `docs/lessons.md`? **yes** — "a check that cannot
      fail the way you cite it": `--check` was named as the mitigation for a risk
      it is structurally blind to. Generalises past this spec.
- [ ] ADR-worthy decision for the repo's `docs/adr/`? no — the manifest gains a
      field, not an architecture.
- [ ] New pattern candidate for `00_meta/patterns/`? no — the cross-project half
      already exists as `pattern-verification-fails-toward-unproven`; this is one
      more instance of it, not a new pattern.

## Round-3 review findings — disposition

Verdict `PASS WITH GAPS`: 0 blockers, 0 majors, 4 minors. Each one is applied or
ticketed; none is left floating.

| # | Reality | Finding | Disposition |
|---|---------|---------|-------------|
| 1 | REAL | `features.json` f2 verifies AC2 only on the committed targets, never on the rendered doctrine payloads at the per-machine `.gemini` / `.codex` homes | **Ticketed, #1035.** Deliberately not fixed here: the fix edits `features.json`, a contract file, which would invalidate the `reviewed_sha` of the review that asked for it. It is also the wrong home — those payloads are per-machine state that CI cannot see (ADR-013's offline model), so the check belongs in `dotf doctor`. Current state re-verified by hand: 1 hit on all five surfaces |
| 2 | THEORETICAL | The region's "the default mechanism is to stay" reads as a contradiction of the standing "Hand the PR over; don't watch CI" rule | **Applied.** A bridging paragraph in the region names the second rule as the *escape being exercised*, not a counterexample: it makes the human the one who reports a red build, which is the signal that keeps the timed window from opening. Vault edit + `--refresh` |
| 3 | THEORETICAL | The skill's "two minutes" and the region's "ten minutes" look like rival timers | **Applied.** The skill now states the two minutes sit inside the ten-minute window — first look vs. window close, one phase and its deadline |
| 4 | SPECULATIVE | The skill's "by default once a PR has come back" implies an automatic invocation that does not exist | **Applied.** The skill's preamble now says it is a judgement the agent makes, and points at the `pr-stewardship` obligation as what actually binds |

Findings 2-4 land in the vault and its generated records, none of which is a
contract file, so the review's `reviewed_sha` still describes what it reviewed.

A note on how that was nearly lost. Rebasing this branch onto main to pick up
0.43.0 made the staleness gate report all three contract files as changed, while
`git diff` against the same sha showed them byte-identical: the gate asks an
ancestry question about content, so a rewrite that changes no bytes reads as
tampering. Unblocked by restoring the pre-rebase ancestry, and ticketed as
**#1036** — the same failure direction as the squash-merge lesson written this
session, and not a defect of this spec.

## Archive checklist

- [x] Adversarial review passes (`dotf spec review HARNESS-072-pr-stewardship`)
- [x] `proposal.md` frontmatter set to `status: archived`
- [x] Folder moved to specs/archive/HARNESS-072-pr-stewardship/ (left in plain
      text: it was written before the move, when a backticked path would have
      been a live claim the doc-path guard rejects)
- [ ] Bitácora #963 closed with the PR link (ADR-018)
- [x] Promotion above executed (the lesson landed as *"A check that
      cannot fail the way you cite it"*)
