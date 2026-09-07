---
id: lesson-277
type: lesson
status: active
created: "2026-09-06"
owner: manu
tags: [lesson, handoff, diagnosis, testing, evidence, session-continuity]
---

# 277 — A diagnosis repeated often enough becomes indistinguishable from evidence

## What happened

`tests/install-dotf.bats` has been red locally and green in CI for months. A
session at some point wrote this into its handoff:

> **#1409 (BUG-771)** — bats red on clean `main` locally, green in CI: fixture
> isolation, not a regression.

That line was carried forward through roughly **ten** archived handoffs. By the
time it reached this session it was sitting in `MEMORY.md`, in the always-loaded
index, phrased as settled fact. So when six tests went red on a clean tree, I
did what every session before me had done: I recognised it, matched it against
the known note, and **told my user it was the known local-only noise** — before
verifying anything.

A peer session then bisected it properly. It took about ten minutes:

```
failures WITH    the dev dotf on PATH: 6
failures WITHOUT the dev dotf on PATH: 0
```

The cause was not fixture isolation and never had been. `scripts/install-dotf.sh`
resolves the **ambient** `dotf` off `PATH`:

```sh
_dotf_current="$(dotf version 2>&1 | grep -oE '[0-9]+\.[0-9]+\.[0-9]+|dev' | head -n1)"
if [ "$_dotf_current" = "dev" ]; then
    log_info "dotf is a source build (dev); leaving it in place"
    return 0            # <- installs nothing, returns success
fi
```

A source build reports `dev`, so on any dev box the installer takes its
deliberate leave-it-alone branch and returns 0 **without installing**. The tests
then fail on `[ -x "$DEST/dotf" ]` rather than on the status — which is precisely
why the failure looked like fixture damage rather than like a short-circuit. CI
is green because CI has no source-built `dotf` on `PATH`.

The test even tried to guard against this and could not have worked:

```sh
VERSION="9.9.9"   # no real dotf reports this -> idempotence stays off
```

That targets the **version** gate. The `dev` gate sits four lines earlier and
returns before the version is ever compared, so no value of `VERSION` reaches it.
The guard was not incomplete; it was **unreachable** in exactly the case that
failed.

## Why it happened

Not carelessness, and that is the point worth keeping. Every session behaved
rationally.

**The handoff format has no field for confidence.** A measurement and a guess are
written in the same voice, in the same place, with the same authority. "Fixture
isolation, not a regression" reads identically whether someone bisected it or
glanced at it and moved on. Nothing in the note distinguishes the two, and the
session that wrote it is gone.

**Each session's cheapest move is to trust the previous handoff.** That is what a
handoff is *for*. Re-deriving every inherited claim would make continuity
worthless. So the rational act, repeated, is also the act that launders a guess
into a fact — and repetition adds apparent weight while adding no evidence. By
the tenth handoff the claim had a provenance chain and still had zero
observations behind it.

**The tell was there and it is generic.** The diagnosis named a *category*, not a
mechanism. "Fixture isolation" identifies no file, no line, and no command that
would confirm or refute it. A real diagnosis is falsifiable and cheap to re-run;
this one could only be agreed with. **Any inherited cause you cannot immediately
turn into a command is a guess wearing a fact's clothes.**

**The second harm is worse than the first.** Six permanently-red tests are a place
a seventh, real failure hides in plain sight. And because the suite is unusable
locally, nobody runs it locally, so CI becomes the first place anything is caught
at all. The cost was never "six tests are red" — it was the whole local signal.

## The rule

**A handoff claim that names a cause must carry the command that established it,
or be marked as unverified.** One line either way:

```markdown
- #1409 — bats red locally, green in CI. Cause: fixture isolation.
  Established by: <command> on <date>.          <- a measurement

- #1409 — bats red locally, green in CI. GUESSED: probably fixture isolation.
  Never bisected.                                <- honest, and stays honest
```

The second form is not weaker than the first. It is the one that gets fixed,
because it advertises the work that has not been done. The first form, written
without the evidence line, is the one that survives ten sessions.

Three corollaries:

- **Inheriting a diagnosis is inheriting a claim, not a result.** Before repeating
  it to a human, cost it out: confirming this one took a 30-second `PATH` swap
  against ~10 sessions of tax. That ratio is typical, because a real cause is
  usually cheap to demonstrate — which is what makes it real.
- **Repetition is not corroboration.** N sessions agreeing is one session's guess
  with N-1 copies. Provenance chains feel like evidence and are not; if you cannot
  find the observation at the root, there isn't one.
- **State inherited claims as inherited.** "The handoff says X, unverified" costs
  four words and keeps the uncertainty attached to the claim as it travels. I said
  "these are the known local-only failures" and dropped it, which is how the next
  session would have received it as fact from *me* as well.

## Worked example

**#1409** is the full trace: the wrong diagnosis, its ten-handoff survival, the
real cause (two gates in series, the second unreachable), and the dead ends ruled
out — a different `VERSION` cannot work, and a `DOTF_VERSION` override is closed
off by `install-dotf.sh`'s own comment arguing #1305.

Kept deliberately separate from **#1469** (`~/.local/bin/dotf` is a dev source
build). Fixing #1469 would make the six tests pass **by accident** while leaving
the `dev` branch untested — converting a visible failure into an invisible one,
which is strictly worse than today.

## A sibling shape, same night

A second session made the mirror-image error within the hour: it searched
`.pre-commit-config.yaml` and `.github/workflows/*.yml` for anything reading a
lesson number, found nothing, and stated in a GitHub comment that **no guard
existed** — while having run `bats tests/*.bats` repeatedly in that same session.
`tests/` is where this repo keeps its guards, and the guard was there. It
retracted publicly.

Different mechanism, same family, and worth naming together: **a non-observation
treated as an observation.** Mine was a claim inherited and never checked; that
one was an absence concluded from a place never searched. Both produce a
confident sentence with nothing behind it, and in both the checking step is
precisely what manufactured the confidence.

That is the sharpest form of the rule. **A check that cannot fail is worse than no
check**, because no check leaves you uncertain and a vacuous one leaves you sure.

## Corollary: a right finding lends its credibility to a wrong remedy

Twice in one night, from two different reviewers, on two different changes:

- PR-Agent reported that a vault source was not updated alongside its record.
  **The finding pointed at something real** — nothing verifies the two copies
  agree, now filed as #1545. **The stated fact was false.**
- An adversarial reviewer found that a dictated `--tier` skips the validation a
  persona-declared tier gets, so an invalid one fails later with a thinner
  error. **Real.** Its proposed fix was to reuse the persona path's error
  message — which blames *a persona's record* for a value a human typed. **The
  remedy was wrong.**

The failure is not that the reviewer was wrong. It is that **a correct finding
lends its credibility to whatever remedy is attached to it**, and the remedy is
where a reviewer is furthest from the code: it has read the diff, not run it, and
it is proposing a change it will not have to make work.

Both get applied verbatim if you are tired, and applying them is worse than
ignoring the finding, because the wrong fix closes the ticket.

**Grade the finding and the remedy separately.** Accepting one is not accepting
the other. In the `--tier` case the right fix was in a third place neither the
reviewer nor the original diff had named — `ResolveChain`, where both paths
already meet — which is the usual shape once you stop taking the suggested
location as part of the finding.
