# Lesson 219 — A stale CLI refuses with the same exit status as a legitimate refusal

**Date:** 2026-08-21
**Context:** Wiring `scripts/compile-harness.sh` to `dotf harness resolve-tier`, the first consumer
of `harness/model-map.json` (HARNESS-076, #1161).

## What happened

The agent render needed to tell two failures apart, because they demand opposite responses:

| cause | right response |
|---|---|
| the map has no model for this tier/harness | **fail the render** — a routing error, and C15's case |
| the resolver is too old to answer at all | **warn and degrade** — a bootstrap state, not a bad map |

The obvious discriminator is the exit status. It does not work. Measured against the `dotf`
actually deployed on the dev box, which predates the subcommand (#1158):

```console
$ dotf harness resolve-tier top --harness claude --repo-root .
Error: unknown flag: --harness
$ echo $?
1
```

A genuine routing refusal also exits 1. **The old binary and the correct binary disagree about the
question, and agree about the answer.**

A second, subtler shape lurks behind it. Cobra runs a parent command's own `RunE` when it does not
recognise a subcommand, and `dotf harness`'s `RunE` prints help and returns nil. So a *differently*
stale binary — one where the flag happens to parse — answers **exit 0 with a help screen on
stdout**, and `model_id="$(...)"` captures the whole help text as the model id.

## The fix

Probe the **capability**, not the outcome. Ask whether the binary knows the subcommand, which is
the one question whose answer does not depend on arguments the old binary could not parse:

```sh
dotf_knows_resolve_tier() {
    dotf harness --help 2>/dev/null | grep -q '^[[:space:]]*resolve-tier[[:space:]]'
}
```

Then pin the string the probe greps for, from the side that generates it:

```go
func TestHarnessHelpListsResolveTier(t *testing.T) { /* asserts `dotf harness --help` lists it */ }
```

Without that test the probe is a one-way ratchet: rename the subcommand and the probe answers "too
old" forever, silently degrading every render — a guard that fails **open** and reports health.

## The lesson

**A version gate must ask what the tool can do, not what it just did.** An error status compresses
"I refuse" and "I do not understand the question" into the same byte, and only the first is a fact
about your data.

Two corollaries worth carrying:

- **A zero exit is not consent.** A CLI that prints help for an unrecognised subcommand reports
  success while answering a different question. Validate the *shape* of what came back — a model id
  is one non-empty token — rather than trusting the status alone.
- **Every probe that greps another component's output needs a test on the producing side.** The
  grep and the string it hunts for live in different languages and different directories here
  (shell in `scripts/`, cobra in `cli/`), which is exactly the distance across which a rename goes
  unnoticed.

This is a sibling of [lesson 214](lesson-214-a-declared-status-is-not-evidence-probe-the-syst.md) — a
declared status is not evidence — applied one level down, to the status a *tool* declares about
itself rather than the one a config declares about a system.

## Where it bit

`scripts/compile-harness.sh` (`dotf_knows_resolve_tier`, `resolve_model_tier`),
`cli/internal/cmd/harness_resolve_tier_test.go` (`TestHarnessHelpListsResolveTier`),
`tests/compile-harness.bats` (the stale-binary case reproduces the measured `unknown flag` + exit 1
shape, not an invented one).
