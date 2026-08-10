---
tags: [spec, verification, templates]
created: "2026-08-10"
---

# Verification - HARNESS-064-adversarial-review-trigger

## Evidence

Every command below was run in this session on `feat/adversarial-review-trigger`.

- [x] AC1 (trigger states its evidence) -> `tests/agents-md.bats` → "AGENTS.md carries the
      adversarial-review trigger with its evidence" — **proved red first** against the current
      `AGENTS.md` (`grep -qF '/adversarial-review' "$AGENTS_MD"' failed`), green after the paragraph
      landed
- [x] AC2 (literal "verification window") -> "AGENTS.md names the verification window, not just
      'before archiving'" — also captured failing first
- [x] AC3 (implementer cannot self-review) -> "AGENTS.md trigger forbids the implementer supplying
      their own review" — also captured failing first
- [x] AC4 (the pin exists and bites) -> the three tests above were run against the unmodified
      `AGENTS.md` and all three failed, each naming the missing string; a pin that had never been
      seen red would prove only that it was written
- [x] AC5 (skill carries the activation rule) -> `Agent-Side Activation Rule` + `When NOT to propose`
      + the `Proactive (verification window)` cross-reference all present in the render; the
      "When to use" list gains the HARNESS-064 bullet
- [x] AC6 (render regenerated, no drift) -> `./scripts/compile-harness.sh --refresh` then `--check`
      -> `no harness drift`

## Test status

- `bats tests/agents-md.bats` -> 18/18 ok (15 pre-existing + 3 new)
- Full suite via pre-commit (`./scripts/test.sh`) -> passed on every commit
- `./scripts/compile-harness.sh --check` -> `no harness drift`
- `jq -e .` on `features.json` -> valid; every `verification` command executed and exits 0
- `features.json` states left `pending` with empty `evidence` — only the harness may write `passing`

## Decisions made during implementation

- **The pin was proved red before the rule was written.** This spec exists because a rule with no
  checking layer does not fire; shipping a pin that had never been observed failing would have
  reproduced that defect one level up, in the test rather than the rule.
- **`--refresh` output was inspected before staging.** It touched exactly one render
  (`harness/skills/adversarial-review/SKILL.md`). Had it touched others, that would have meant a
  parallel session edited the vault without shipping its render, and those files would have been
  left out of this PR rather than swept in.
- **The trigger proposes and never acts.** An agent that both proposed and answered would produce
  the single-agent self-review the skill forbids under "When NOT to use", so the trigger text names
  the constraint explicitly ("cannot be the reviewer") rather than leaving it to the skill body —
  the trigger is the surface an agent reads first.
- **Placed between steps (6) and (7), not "before archiving".** Wording it against the archive would
  have re-encoded the very timing problem this spec removes: the archive is where the requirement is
  discovered *too late*.
- **Instruction-surface caps checked before growing `AGENTS.md`** (25603 chars). agy caps each rules
  file at 12000 and codex stops at 32 KiB, but both receive the compact doctrine payload (~2 KB),
  not the whole file — so this paragraph cannot breach either.

## Promotion candidates

- [ ] Lesson for the repo's `docs/lessons.md`? **not yet** — the transferable insight here ("a gate
      needs a trigger, not only an artifact") is the same family as the two lessons already written
      this session, and a third entry restating it would be noise. Revisit if the trigger fails to
      fire in practice, because *that* would be the novel finding.
- [ ] ADR-worthy decision? **no** — this implements the existing two-surface skill+trigger pattern
      (SDD-011), it does not choose a new architecture.
- [ ] New pattern candidate for `00_meta/patterns/`? **candidate, held** — "bind the artifact AND the
      moment": CLI-034 bound the artifact, this binds the moment, and the pair is what makes an
      enforcement layer usable rather than merely present. Promote only if it recurs in a second
      project, per the existing rule.

## Archive checklist

- [ ] `proposal.md` frontmatter set to `status: archived`
- [ ] Folder moved to `specs/archive/` (via `dotf spec archive HARNESS-064-adversarial-review-trigger --pr <url>`)
- [ ] **`review.md` produced by an INDEPENDENT session** — this spec proposes the review it now
      triggers; the implementer supplying it would be the exact self-review the trigger forbids
- [ ] Bitácora #879 moved to Done / closed with PR link (ADR-018)
- [ ] Promotions above executed
