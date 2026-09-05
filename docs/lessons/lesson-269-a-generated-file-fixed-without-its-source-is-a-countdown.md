# Lesson 269 — A generated file fixed without its source is a countdown, and the drift guard says OK either way

**Date:** 2026-09-05
**Context:** Two reversions in two days (`new-ticket`, `crystallize`), found by two different sessions, neither found by the guard whose job it is

## What happened

`harness/skills/**` and `harness/agents/**` are **generated** from the knowledge vault. Each record
carries its provenance in frontmatter:

```yaml
generated: true
generated_from: 00_meta/skills/crystallize/SKILL.md
generated_sha: d6d6cf0c738eab7d
```

`scripts/compile-harness.sh --refresh` re-renders every one of them from the vault. Twice in two
days, someone fixed a defect by editing the **generated** file, merged it, and had the fix silently
reverted the next time anybody ran `--refresh`.

### Instance 1 — `new-ticket`

A fix sat uncommitted in the main checkout for roughly 20 hours before anyone noticed it had been
undone. Nothing reported it. It was found because a human happened to read `git status` in a
checkout they were passing through.

### Instance 2 — `crystallize`

PR #1490 removed a reference to `knowledge-crystallize.sh`, a script deleted in #1276, from
`harness/skills/crystallize/SKILL.md`. It merged. Six hours later a `--refresh` in a fresh worktree
put the deleted script's name straight back:

```diff
-- Run: `dotf vault crystallize` (add `--all` to stamp every project)
++ Run: `knowledge-crystallize.sh` (or `dotf vault crystallize`)
```

**`--refresh` was not malfunctioning. It was propagating a stale SSOT correctly.** The vault's
`00_meta/skills/crystallize/SKILL.md:67` still carried the old line, because #1490 had fixed the
render and never touched the source. The generated file was right for six hours and wrong forever
after.

## The part that makes this a guard bug, not a discipline problem

`compile-harness.sh --check` is the repository's drift guard for exactly these artifacts. It runs
in CI. Agents cite it as evidence the tree matches the vault. Measured on a clean tree at `ca81157`:

| state | `crystallize/SKILL.md` md5 | contains the deleted script | `--check` |
|---|---|---|---|
| A — as committed, #1490's fix in place | `1bb56a86` | no | `[check] OK -> record crystallize`, **rc=0** |
| B — after `--refresh` reverted it | `d84f1c33` | **yes** | `[check] OK -> record crystallize`, **rc=0** |

**Both pass.** `--check` returns rc=0 on a tree that `--refresh` — the same script, one flag apart —
would itself rewrite. Not confined to skill records: on the same run it printed `[check] OK ->
AGENTS.md` while `--refresh` added two doctrine paragraphs to that file.

So the two reversions were not missed through carelessness. The instrument that exists to catch them
was answering a different question, confidently, in the affirmative. Filed as **#1502**, separately
from the reversion incident (**#1493**) — an incident and a broken instrument get fixed at different
speeds and should not share a ticket.

## The blast radius nobody sees until they look

One `--refresh` on a clean tree moved **seven** files in **three different directions**:

```
 M AGENTS.md                             forward  — doctrine the repo lacks, owned by another branch
 M ai/claude/CLAUDE.md                   forward  — same
 M ai/orca/ORCA.md                       forward  — same
 M harness/enforced/no-auto-merge.md     forward  — same
 M harness/agents/shipper/AGENT.md       MINE     — the one-line change I intended
 M harness/skills/crystallize/SKILL.md   BACKWARD — reverts #1490
 M harness/skills/new-ticket/SKILL.md    stamp    — content identical, generated_sha differs
```

Every line of `--refresh`'s output reads `[refresh] skill record: …` regardless. **The command
distinguishes none of these, and the direction is the only thing that matters.** A narrow intent —
one field on one persona — produces a seven-file review in which six files belong to other people
and one of them is a regression.

## Content-identical is not the same property as clean-after-refresh

A subtle residue, and both fixes hit it. After repairing the vault source, `--refresh` leaves the
record's **content** identical — but the file still shows as modified, because `generated_sha` was
computed from the *old* source. Two different claims:

- *"the round-trip is content-identical"* — what both fixes achieved
- *"`git status` is clean after `--refresh`"* — what neither achieved until the stamp was carried

The second is the one that stops the next person seeing noise and learning to ignore it. The first
sounds like it implies the second and does not.

## The lesson

**Editing a render is never a fix. It is a fix with a timer on it, and the timer is however long
until the next person runs the generator.**

The failure is attractive because the render is where you found the defect, the render is what the
consumer reads, and editing it makes the symptom go away immediately — including in CI, including
in review. Nothing in the loop distinguishes it from a real fix. In both instances here the change
was correct, reviewed, and merged; it simply had no durability.

This is the same shape as lesson 092 (*editing a committed render without its source of truth*) and
lesson 009 (*always edit the repo copy, never the deployed system*), which is worth stating plainly:
**this class has now been documented three times and recurred anyway.** Documentation is not a
mechanism. #1502 is the mechanism.

## What to do instead

- **Read the frontmatter before editing any file under `harness/`.** `generated: true` and
  `generated_from:` are there precisely to answer this, and they are the first eight lines of the
  file. Both instances edited a file whose second line said not to.
- **Fix the vault, then `--refresh`, then commit both the content and the stamp.** The stamp is what
  makes the round-trip idempotent for everyone else.
- **After any `--refresh`, check the direction of every file it touched — not just the count.**
  `git diff -- <file>` per file. Seven moved files with one intended change is normal; six of them
  being someone else's is also normal. Reverting the ones that are not yours is the routine step,
  and it needs the diff, because forward and backward look the same in `git status`.
- **Do not read a green `--check` as "my tree matches the vault".** Until #1502 lands it does not
  test that. Run `--refresh` in a scratch clone and diff if you need the answer.
- **Prefer `--refresh` in a fresh worktree off `main`,** so unrelated drift shows up as *someone
  else's uncommitted change* rather than mixing into your own diff.

## Relation to the neighbouring lessons

Lesson 268 is the immediate neighbour and the same failure at the tooling boundary: *"no drift"* and
*"I did not check for drift"* print identically. The table above is that sentence with numbers
attached — `--check` emits the identical `OK` line for a correct file and a reverted one, so its
output carries no information about the thing it names.

Lesson 267's requirement — a mutation harness must prove the mutation **landed** — is the same
demand made of a guard: prove the check **ran against the thing it names**. A guard that would pass
whatever the file contained has not checked the file, and no amount of green tells you which case
you are in.

Lessons 220 and 214 name the parent class: *a thing verified by a proxy is not verified*, and *a
declared status is not evidence — probe the system*. `--check`'s `OK` is a proxy for "the render
matches the source", and the two instances here are what it costs when the proxy is not the thing.
