# Lesson 230 — A config that parses is not a config the consumer reads

**Date:** 2026-08-24
**Context:** CLI-042 PR E (#1190) — wiring the NaN credential into hive's daemon.

## What happened

I wrote a systemd drop-in to hand the hive daemon its credential, and nine bats
assertions to prove it. All nine passed:

- does the `ExecStart` invoke `dotf secrets run`?
- is the injection scoped with `--only`?
- is the `ExecStart` list reset before the override?
- does it use `%h` rather than a literal home?
- is `StartLimitIntervalSec` under `[Unit]` and `RestartSec` under `[Service]`?

Every one of those was true. The drop-in was also useless: it injected
`NAN_API_KEY`, and hive's worker reads `HIVE_WORKER_API_KEY` and
`HIVE_WORKER_BASE_URL`. It ignores `NAN_API_KEY` entirely.

Deployed as written, the daemon would have restarted, systemd would have
reported `active (running)`, the test suite would have stayed green, and the
worker would have gone on serving nothing — the exact state the criterion
existed to end.

## Why the tests could not catch it

They asserted the **shape** of the configuration and never its **effect**. Every
question above is about the artifact: its syntax, its directives, its internal
consistency. None of them is about the consumer.

That gap is invisible from inside the artifact, because a config file has no
opinion about whether anyone reads it. `Environment=BANANA=1` is a perfectly
valid systemd directive. A schema check, a linter, and a unit test on the
rendered text will all pass on a variable no process on earth consumes.

## What broke the tie

Asking the consumer. hive ships a `worker_status` tool that **probes** the
provider rather than inferring it from configuration, and it answered:

```
## Provider
- Configured: no — set HIVE_WORKER_BASE_URL
```

One question to the thing being configured, and the whole test suite's verdict
was overturned. Reading `/proc/<pid>/environ` for variable NAMES (never values)
confirmed it independently: zero credential-shaped variables in a daemon that
had been "healthy" for 2h40m.

## The rule

**When you configure a consumer you do not own, one assertion must come from the
consumer.** Everything else — syntax, directives, deployment, idempotence — is
worth testing and proves only that you wrote the file you meant to write.

Concretely, before believing a config change works:

1. Find the consumer's own status/probe surface and ask it. Most daemons worth
   configuring have one.
2. Failing that, read the process's actual environment for variable NAMES — never
   values, which belong in no transcript.
3. Failing both, name the gap in `verification.md` as owed rather than closing
   the criterion.

## The near-miss worth naming

hive had already learned this lesson and I did not read it. `worker_status`'s
own documentation says:

> The old output ... reported two providers by *configuration*: it said "Ollama:
> offline / OpenRouter: no API key" for an unknown length of time while every
> caller treated the worker as a working capability. A status surface that cannot
> distinguish "configured" from "answers" is how a dead backend stays invisible.

Upstream rebuilt that tool precisely so configuration could not masquerade as
capability — and I then wrote nine tests that asserted configuration and called
it done. The tool that would have told me in one call was already installed.

## Related

- Lesson 229 — an empty secret is not an error. Same family: the failure is
  silent and points away from its cause.
- CLI-042 AC9 is the guard emitted for this: `dotf doctor` now fails when hive
  probes present but is missing either half of its worker contract, so the
  configured-but-dead state cannot go unnoticed for an unknown length of time
  again.
- The AC9 check is itself a **proxy** and says so in its own doc comment: it
  reads the unit rather than probing the worker, because reaching `worker_status`
  from `dotf doctor` would put an MCP client inside a diagnostic. A proxy whose
  limit is written down is honest; one whose limit is implied is this lesson
  repeating.
