---
id: lesson-212-an-invalid-instrument-is-indistinguishable-from-an
type: lesson
status: active
created: "2026-08-19"
owner: manu
tags: [lesson, dotfiles, guards, verification]
---

# Lesson 212: An invalid instrument is indistinguishable from an absent guard

**Context**: two nights (2026-08-17/18) spent closing the review loop — #1033, #1042, #1045, #1047, #1052, #1054, #1065, #1072, #1073. Nine defects that looked unrelated and were one shape.

**Problem**: every one of them was *a signal a check consumes quietly stops answering the question it is asked, and the check keeps reporting green*. That is `pattern-verification-fails-toward-unproven`. What these nine added is the corollary that bites hardest in practice, and it is about the act of testing rather than the thing tested.

**When a mutation test comes back "nothing failed", there are two explanations and they look identical**: the guard is vacuous, or the mutation never landed. Three times in one session the second was true and was read as the first:

- A mutation edited the *documentation table* that describes `EXEMPT_SUITES` instead of the variable. Nothing failed. The guard looked vacuous; it was fine.
- Two "liveness samples" came back byte-identical, read as a stalled process. Foreground `sleep` is blocked in that harness, so both samples were taken at the same instant.
- An isolation control ran `git stash`, which had nothing to stash, and therefore measured the *new* guard while reporting that it measured `main`'s. It concluded "no gap" about a gap that was real.

The same shape appears without any mutation at all, which is what makes it a class rather than a habit:

- **A guard that asserts the configuration text, not the behaviour.** `pr-agent: excluding release PRs from review is paired with a gate exemption` checks that two strings are declared. Both are. The reviewer reviews release PRs anyway, because the setting is loaded and never consulted (#1073). Green, and the invariant it names is false.
- **A guard whose list is hand-written.** *"The script names no reviewer of its own"* refuted three names someone thought of, while the file named a fourth added by a later PR (#1033). It now derives the list from the registry, which is the only place that knows it.
- **An instruction in prose where a marker was needed.** The reviewer is told to open every review with a compliance section. It does so once in sixteen (#1072). Nothing notices a miss, because a review missing the section looks exactly like a review that had nothing to report.

**Solution**: three rules, in the order they bind.

1. **Confirming the mutation landed is part of the mutation.** Print the mutated line, or grep the file, *before* running the suite. `"mutation present? -> 1 occurrence"` in the transcript is the difference between evidence and a story. Green after a mutation is not information until the mutation is proven present.
2. **Assert the effect, never the declaration.** "The setting is present" and "the behaviour happens" are different claims, and only the second is what anyone wanted. Where the effect cannot be asserted yet, say so in the test's own comment rather than asserting the cheaper thing silently — and do not add the effect assertion until the effect exists, or it is simply red.
3. **Anything a check consumes gets a machine marker, not prose.** Prose is reworded, translated, A/B tested, and dropped under context pressure. This repository already knew that for `declined_markers`, which are matched on vendor machine output for exactly this reason; the nine defects above are what it costs to learn it once per surface. When a marker does drift, the failure mode must be *escalate*, never *pass silently* — `unclassified -> a human looks` rather than `unclassified -> assume fine`.

**Cross-reference**: `pattern-verification-fails-toward-unproven` (vault, `00_meta/patterns`), extended 2026-08-18 with three further arrivals. Lesson 210 is the same family seen from the git side: `git branch --merged` answering an ancestry question about content.
