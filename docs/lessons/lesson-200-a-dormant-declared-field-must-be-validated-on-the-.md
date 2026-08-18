---
id: lesson-200-a-dormant-declared-field-must-be-validated-on-the-
type: lesson
status: active
created: "2026-08-14"
owner: manu
tags: [lesson, dotfiles]
---

# Lesson 200: A dormant declared field must be validated on the same schedule it's written, not the schedule it activates on

**Context**: OPS-028 (#951, PR #957) added `bw.folder` to the secrets registry schema — ADR-028's already-ratified Bitwarden folder taxonomy, which the schema had never implemented. The registry's own convention (ADR-028 §2's addendum) pre-declares a secret's `bw:` block — item, field, and now folder — while `backend:` is still `age`, dormant until `dotf secrets migrate` flips it. The first implementation followed the existing code's own pattern for gating validation: `checkBwSources` (item/field completeness) only runs inside the `switch s.Backend { case "bw": ... }` arm, so the new `folder:` check was added there too, by the same reasoning.

**Problem**: an independent adversarial review (run on a non-Anthropic model per this repo's reviewer-pool rule, since Claude implements nearly every change here including this one) caught that gating folder validation on `backend == "bw"` meant every one of the 26 registry entries this PR populated — all still `backend: age` at merge time — had its `folder:` value parse clean with **zero validation**, because the check that would catch a typo never ran while backend was age. The gap wasn't cosmetic: `dotf secrets migrate` reads `s.BW.Folder` and hands it straight to `ResolveFolder`, which **creates** a Bitwarden folder for whatever string it's given if none matches — so a typo'd dormant value wouldn't fail loud at parse time, the safe place; it would silently create a stray folder in the live vault at migrate time, the exact drift the whole feature exists to prevent.

**Solution**: moved the folder check out of `checkBwSources` into the main per-secret validation loop, unconditional on `s.BW != nil` rather than `s.Backend == "bw"` — so it runs the moment the value is written to the registry, matching the moment `item`/`field` conceptually become "the" target even though they aren't live yet either. (Left `checkBwSources`'s own item/field checks as they were — narrowing that gate too was out of scope for this fix and not what the review flagged.)

**Rule**: when a schema pre-declares a field that activates later (a dormant `bw:` block, a feature flag's config, a "coming soon" API shape already being written to storage), validate it at write time — the moment a human or a pipeline can put a bad value in — not at activation time. Gating a check on "is this live yet" optimizes for the code path that runs least; the value sits wrong and silent through every state before it, and the state that finally reads it is usually the one where failing loud is most expensive (a live side effect, not a parse error). If existing code already gates a sibling check the same way, that's precedent, not proof it's correct — check what the gated-out state can actually reach before reusing the pattern.

**Tags**: `secrets`, `bitwarden`, `validation`, `spec-driven-development`, `adversarial-review`
