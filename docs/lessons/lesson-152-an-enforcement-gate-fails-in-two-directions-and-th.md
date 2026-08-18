---
id: lesson-152-an-enforcement-gate-fails-in-two-directions-and-th
type: lesson
status: active
created: "2026-08-06"
owner: manu
tags: [lesson, dotfiles]
---

# Lesson 152: An enforcement gate fails in two directions, and the cheap one is the refusal

**Context**: Adding the archive-on-merge half of the SDD Discipline Gate (#670) to `check-spec-gate.sh`, which runs under `set -euo pipefail`. The suite was written first and deliberately covered both directions: ten tests asserting the gate *fires* on a violation, ten asserting it stays silent on `Refs #N`, on a prose mention, on a cross-repo reference, on an empty PR body, on a spec with no `issue:` frontmatter.

**Problem**: The first implementation passed every "must fire" test and failed four of the "must not fire" ones — and both causes were `set -e` interactions on the *nothing-matched* path, which for a gate is the normal path. `grep` exits 1 when it finds no closing keyword, so under `pipefail` the capture aborted the entire script: every ordinary PR whose body said `Refs #N` would have been blocked by an exit code that had nothing to do with the rule. Separately, `[[ -n "$num" ]] && printf …` as the last command of a `while` body made the enclosing loop exit 1 whenever a spec carried no `issue:` field — 28 of 44 active specs — and the caller, correctly written to fail closed on an unreadable tree, read that as an unreadable tree. A gate that refuses valid work is not "safely strict"; it is broken in the direction that costs the most, because every author hits it and none of them can tell why.

**Solution**: Guard the no-match path explicitly (`grep … || true`, with the reason in a comment) and replace the `&&` idiom with an `if` block so the loop body cannot leak a falsy status. Then dogfood the finished gate against the repository itself rather than only fixtures: run it on the real branch with `Closes #670` (must fail, naming the spec) and with `Refs #670` (must pass). That surfaced a further gap no fixture had — reading active specs only at the base ref misses a PR that creates a spec and closes its issue in the same change, which is precisely the "created, shipped, never archived" pattern the gate exists to stop.

**Rule**: For anything that can block work, write the negative tests first and treat them as the primary suite — "does not fire when it shouldn't" is harder to get right than "fires when it should", and its failures are invisible until they are blocking someone. Under `set -euo pipefail`, audit every command whose *success* case is "found nothing": `grep`, a `[[ … ]] &&` as a loop body's last statement, an empty `read` loop. They are the ones that turn a clean pass into an abort. And a gate is only verified once it has been dogfooded on real data — fixtures encode the cases you already thought of, the repository encodes the ones you did not.
