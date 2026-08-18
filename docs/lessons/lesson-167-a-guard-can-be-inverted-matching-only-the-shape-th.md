---
id: lesson-167-a-guard-can-be-inverted-matching-only-the-shape-th
type: lesson
status: active
created: "2026-08-08"
owner: manu
tags: [lesson, dotfiles]
---

# Lesson 167: A guard can be inverted: matching only the shape that is always a false positive, and blind to the shape that is always a true positive

**Context**: `#769` reported that `dotf spec archive` refused specs that merely *quote* the `[AGENT-DRAFT]` / `[AGENT-SUGGESTION]` markers — inside a code span, or on a completed `- [x]` line. The obvious reading is "the matcher is too broad".

**Problem**: The matcher was also too *narrow*, and that half was worse. The pattern required `]` immediately after the keyword, but `/spec fill` emits the suffixed form (`[AGENT-DRAFT — review before archive]`) documented in the skill. That form had never matched, so the archive lock **had never once fired on a marker emitted the way the tooling emits it**. Proof was in the tree, not in reasoning: `specs/archive/CLI-002-repo-structure/proposal.md` sits in the archive today still carrying a live suffixed marker. Meanwhile the one shape the pattern did match — the bare form — is the shape this repo writes when *documenting* the markers rather than using them. The guard matched exactly what is always a false positive and was blind to exactly what is always a true positive. The reported symptom was the benign half.

**Solution**: Widen the pattern to both emitted shapes and exclude the quoted contexts (fences, code spans, ticked items), with one predicate shared by every call site — a second consumer, the session-start injector, had drifted to its own cruder scan and disagreed with the archiver about what "unresolved" even meant.

**Rule**: When a matcher produces false positives, check the false-negative direction in the same pass — a pattern written from memory of the format rather than from the emitter is as likely to be wrong about what it *misses* as about what it catches. Establish the canonical form from the **producer** (the emitter, the schema, the skill that writes it), never from the matcher under suspicion; the matcher cannot be both the artefact under review and the reference. And test the claim against the corpus: one grep over the archive turned "the lock may not fire" into a named file.
