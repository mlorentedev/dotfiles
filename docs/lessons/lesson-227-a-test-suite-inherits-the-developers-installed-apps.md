# Lesson 227 — A test suite inherits the developer's installed applications, and PATH is the door

**Date:** 2026-08-23
**Area:** tests / isolation / guards
**Severity:** medium — noisy and destructive-adjacent rather than silently wrong, but it drove a real desktop and touched live data

## What happened

A routine `bats tests/*.bats` left **18 Obsidian processes** running. Their argv
says what they were:

```
obsidian --no-sandbox dead-ends  --vault knowledge
obsidian --no-sandbox unresolved --vault knowledge
  --user-data-dir=/tmp/bats-run-gtlnlo/test/7/config/obsidian
```

`dead-ends` and `unresolved` are sections of `scripts/vault-health.sh`, which
reaches Obsidian as a bare command name. `--vault knowledge` is the developer's
**real** vault. The `bats-run` tmpdir in `--user-data-dir` is what proves a test
started them: Electron derives that path from the environment, and only a test
has a bats tmpdir there.

So a test ran a production script, the script resolved `obsidian` through PATH,
PATH found the developer's AppImage, and the GUI opened against real data —
eighteen times, silently, with the suite reporting green throughout.

## Why it survived

**The suite already knew about this hazard and had solved it once.**
`tests/golden/vault-health/lib.sh` replaces PATH with a sandbox and then asserts
no leak, with a comment naming the exact risk: *"any case could in principle
launch the actual GUI against the actual vault"*. That protection was correct,
thorough — and local to one file. Every other test inherited the developer's
PATH unchanged.

**Nothing failed.** The processes are a side effect, not an assertion, so no
test noticed. The only symptom was the human watching windows appear.

**The obvious fix is an instruction.** "Stub GUI tools in your tests" is exactly
the kind of rule that holds until someone writes a test without having read it —
the same shape as lesson 115, violated three times the same day it was reread.

## The fix

`tests/setup_suite.bash`, which bats loads once for the whole suite, so a new
test file inherits the protection without its author knowing it exists. Two
halves, because one cannot cover the other's gap:

**Interceptors on PATH** for every GUI binary installed on a developer machine
(`obsidian`, `orca`, `code`, `xdg-open` — the class, not just the one that bit).
They **refuse** rather than exit quietly: a silent success would convert "opens
a window" into "passes for the wrong reason", trading a visible failure for an
invisible one. Each refusal prints the remedy inline and logs the attempt with
`BATS_TEST_NAME`, so the next occurrence names its own culprit instead of
requiring process-table archaeology.

They exit **99, not 127**. 127 means *command not found*; this command was found
and declined. bats says so out loud (BW01), and the wrong code would teach every
reader of a failure the wrong cause.

**A post-suite stray detector**, because PATH governs a bare command name and
nothing else: an absolute path, an AppImage, walks straight past an interceptor.
`teardown_suite` looks for GUI processes carrying a bats tmpdir in
`--user-data-dir`, reports them, and terminates them.

**A test's own stub still wins**, being earlier on PATH. The guard sets the safe
default; it does not take the decision away — which is why the golden harness,
which replaces PATH wholesale, keeps working untouched.

## What the guard's own tests had to prove

`tests/guard-no-gui.bats` checks the interceptors resolve, refuse, name the
remedy, log the caller, and yield to a local stub. The detector is checked **in
both directions**, and the second is the one that matters:

| Fixture | Must be |
|---|---|
| `obsidian --user-data-dir=/tmp/bats-run-FAKE/…` | detected |
| `obsidian --user-data-dir=$HOME/.config/obsidian` | **ignored** |

A detector that swept up a human's open editor would be a worse bug than the one
it fixes. That is why the signature is `--user-data-dir` inside a bats tmpdir
and not the binary's name.

## A measurement trap worth carrying

The first reading of this incident reported three stray processes that did not
exist. `ps | grep --user-data-dir=...bats-run` matches **the shell running the
grep**, whose own argv contains the pattern. The same trap applies to `pgrep -f`.
The detector therefore matches by process NAME and filters the flag afterwards.

**A measurement that includes the measurer is not a measurement** — and it fails
in the alarming direction, which is the one that wastes time.

## The rule

**A test may never launch an application, and the suite must make that true
rather than ask for it.** Protection that lives in one test file protects one
test file.

## Relation to Lesson 224

[224](lesson-224-a-negated-assertion-is-exempt-from-set-e-so-it-cann.md) ends on
*falsify the check deliberately and confirm it goes red*. Applied here: the
interceptor is proven by a script written to reach for `obsidian`, and the
detector by a process built to look like a stray — plus one built to look like a
human's, to prove it is left alone.

## Evidence

- 18 Obsidian processes after a suite run, argv and `--user-data-dir` above (2026-08-23)
- `tests/golden/vault-health/lib.sh:45` — the same hazard, solved for one file
- Full suite with the guard: **1475 tests, exit 0, 0 failures, 0 attempts logged, 0 strays**
- `tests/guard-no-gui.bats` — 9 cases, including both directions of the detector
