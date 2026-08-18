---
id: lesson-185-mutation-testing-does-not-only-catch-tautologies-i
type: lesson
status: active
created: "2026-08-09"
owner: manu
tags: [lesson, dotfiles]
---

# Lesson 185: Mutation testing does not only catch tautologies — it finds the boundaries your fixtures never land on

**Context**: The vault-health golden corpus (#890) had 16 cases and 19 green tests, including cases named `orphans-warn` and `orphans-fail` for the two sides of a 30% threshold.

**Problem**: Mutating the threshold from 30 to 25 turned exactly one case red — and an unrelated one, which happened to sit at 30% for a different reason. The tier cases used 40% and 60%: comfortably inside their bands, so a 5-point move did not reclassify either. The suite looked like it tested the boundary because the cases were *named* for the bands, and naming is not coverage.

**Solution**: add fixtures that land *on* the edges — exactly 30% and 50% orphans, exactly 10 unresolved links, exactly 80% and 50% frontmatter coverage (integer division makes 3/10 and 5/10 exact). The same mutation then goes red on the case named for it.

**Rule**: Run the mutation you expect to be caught, not just one you expect to fail. A mutation that turns *fewer* cases red than you predicted, or turns the *wrong* ones red, is reporting a coverage gap rather than a passing guard — read which cases moved, never just the count. For any threshold, the defending fixture is the one whose value equals the threshold; a case on either side proves the branches exist and says nothing about where they divide.
