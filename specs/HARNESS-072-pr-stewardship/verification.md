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

## Archive checklist

- [ ] Adversarial review passes (`dotf spec review HARNESS-072-pr-stewardship`)
- [ ] `proposal.md` frontmatter set to `status: archived`
- [ ] Folder moved to specs/archive/HARNESS-072-pr-stewardship/ (plain text: the
      path does not exist yet, and a backticked one is a live claim the doc-path
      guard checks)
- [ ] Bitácora #963 closed with the PR link (ADR-018)
- [ ] Promotion above executed
