---
id: lesson-168-two-enforcement-gates-each-correct-alone-can-compo
type: lesson
status: active
created: "2026-08-08"
owner: manu
tags: [lesson, dotfiles]
---

# Lesson 168: Two enforcement gates, each correct alone, can compose into a state no change can satisfy

**Context**: `check-spec-gate.sh` runs a Discipline Gate (a large diff must touch an **active** `specs/<id>/`) and an archive-on-merge check (a PR closing an issue must **archive** that issue's spec). `#397` had hardened the first against archive-moves; `#767` later made archive-moves mandatory.

**Problem**: For any PR both over the LOC threshold and closing its own issue, archiving satisfies the second and breaks the first, and not archiving does the reverse. Neither half is wrong in isolation, and each was reviewed in isolation — the contradiction was created by composition, months apart, and only surfaced when a PR happened to be both large and closing. The practical cost is worse than the block: the escape hatches become the normal path, so both gates erode.

**Solution**: Count a spec archived *in fulfilment of* archive-on-merge as the Discipline Gate's spec touch — reachable only through the `issue:` frontmatter of a spec whose issue the PR closes, so a gratuitous archive-move still earns nothing and `#397` stands. The ticket's own proposed fix (count `specs/archive/<id>/` as a touch) does not work, and measuring said so: the gate accumulates LOC against a floor of 10, and a real archive move renders as **4 LOC** — three pure renames at `0 0` plus the `status:` rewrite. Counting its lines would have left the gate exactly as unsatisfiable while looking fixed.

**Rule**: Two gates over the same artifact are a system, not two features; whenever one is added or tightened, enumerate the states the *other* now forbids. The tell is an escape hatch being used routinely — that is a design report, not a workflow preference. And before implementing a proposed fix, measure how the artifact it keys on actually renders (`git show --numstat` on a real instance), because a fix aimed at the wrong mechanism can pass review, ship, and change nothing.
