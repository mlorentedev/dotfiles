---
id: lesson-204-a-check-that-cannot-fail-the-way-you-cite-it
type: lesson
status: active
created: "2026-08-15"
owner: manu
tags: [lesson, dotfiles]
---

# Lesson 204: A check that cannot fail the way you cite it

**Context**: HARNESS-072 (#963) adds an `enforced` harness region — text injected verbatim into every agent's instructions across every repo. The spec's own Risks section named the obvious failure: *"a region added to `enforced` but missing from a target's `inject` list silently misses that surface"* — the producer-updated / consumer-forgotten class that BUG-077 had been. It named the mitigation in the same breath: `compile-harness.sh --check` is the test, not a hand count. The acceptance criterion was written on that basis.

**Problem**: `--check` cannot detect that failure and never could. `do_check` builds the expected side of its diff from the target's *own* inject list — `mapfile -t ids < <(target_inject "$file")` — so an id missing from that list is missing from **both** sides of the comparison. The diff is clean, the target prints `[check] OK`, and the surface that never received the rule is indistinguishable from one that did. It is a consistency check (does the injected text match its record?) being cited as a coverage check (did the region reach the surfaces it should?). Running the new assertion against the tree as it stood produced two immediate hits on `pr-sizing`, doctrine-only by a decision argued at length in #830 and recorded nowhere a machine could read — a real exclusion that had survived on institutional memory alone.

**Solution**: a separate `check_coverage` pass over every enforced id × every surface: injected, or an `opt_out` entry naming that surface **with a reason** — an empty reason is still a gap. The decisive test asserts both halves on one run: the region diff reports `OK -> TARGET2.md` while coverage reports `GAP` on that same file. That also picks the shape — an orphan check ("is this id used anywhere?") would pass the partial case, and the partial case is the likelier mistake. Found by a second session reading this worktree from the outside, and verified against the source before being acted on.

**Rule**: when a spec names a command as the mitigation for a risk, open the command and find the line that would fail. A check earns its citation by the question it actually asks, and the question is usually narrower than its name suggests — `--check` also cannot see a committed record trailing its vault source, because it is offline by design (ADR-013), which is how six stale records sat clean until someone ran `--refresh`. This is the `pattern-verification-fails-toward-unproven` family in its cheapest form: not a check that ran and lied, but a check that was never capable of the answer and was trusted for it anyway. The tell is a mitigation you can state but not demonstrate red.

**Tags**: `harness`, `verification`, `spec-driven-development`, `ci`
