# Lesson 228 — A calendar date is not a timestamp, and UTC is the wrong zone for one

**Date:** 2026-08-23
**Area:** cli / mem / time handling
**Severity:** medium — silently misfiles durable records, and can overwrite one

## What happened

`dotf mem session-end` writes the durable session record whose filename *is* a
date:

```
<vault>/10_projects/<project>/sessions/<date>-<project>-claude.md
```

The caller handed it UTC:

```go
_, _ = mem.SessionEnd(payload, vault.ResolveVault(), time.Now().UTC())
```

On `America/Denver` (UTC-6) that means every session ending after 18:00 local is
filed under **tomorrow**. On 2026-08-23 five records written between 18:21 and
18:47 MDT all landed as `2026-08-24-*`:

```
10_projects/{kubelab,resume,ts-bridge,web,dotfiles}/sessions/2026-08-24-*-claude.md
```

The `ts-bridge` record made the split visible from inside the file: its body
read `Updated: 2026-08-13` under a filename claiming `2026-08-24`.

## Why it is worse than a cosmetic wrong label

The record path is append-only with a same-day collision rule (`-2`, `-3`, …
per `handoff/SKILL.md` §1b). That rule is enforced by the *agent* writing a
handoff, not by the Go writer, which does a plain `os.WriteFile`.

So an evening session filed under tomorrow sits exactly where **tomorrow
morning's** session will be written — and the morning run overwrites it with no
error and no diff, because the file is not in git until the next vault commit.
A wrong date is recoverable. A silently clobbered session journal is not.

## The general shape

`time.Now().UTC()` is close to a reflex, and for an *instant* it is usually
right — logs, sync cursors, `RFC3339` fields all want a fixed zone.

The tell that this was not an instant is one line down:

```go
date := now.Format("2006-01-02")
```

**The moment you format away the clock, you are no longer storing a point in
time — you are naming a day in someone's life.** A day belongs to the observer's
timezone. Once the hours are discarded, UTC is not "the precise choice"; it is a
different day for a predictable slice of every 24 hours.

Rule of thumb: **instants are UTC, calendar dates are local.** If the value ever
reaches a filename, a heading, a `created:` field, or anything a human reads as
"when I did this", it is a calendar date.

## Why it survived four months

It was not a considered policy. Blame puts the `.UTC()` in `dd95039`
(CLI-025, #569), the commit that assembled the session-start adapter — and the
same file already calls plain `time.Now()` for session-start (`mem.go:129`,
`:161`). One noun in the pair was normalised and the other was not, which is
the signature of an incidental keystroke rather than a decision.

It stayed invisible because the failure is **timezone- and time-of-day-gated**:
CI runs in UTC, where the bug cannot reproduce, and a developer only sees it
after 18:00 local. Neither the tests nor the pipeline was ever in a position to
observe it.

## Guards added

Two tests, both verified to fail against the unfixed code before being kept:

- `TestSessionEnd_UsesLocalCalendarDate` — hands `SessionEnd` an 18:30 time in a
  `-0600` zone and asserts the record is filed under that local date. It opens
  by asserting the fixture is actually timezone-sensitive, so it cannot decay
  into a test that would pass in any zone.
- `TestSessionEnd_FrontmatterKeyOrder` — pins `id`/`type`/`status` as the first
  three keys and the presence of `created`/`owner`.

The second exists because the same writer was also emitting frontmatter missing
`status`, `created` and `owner` — an error-level finding in both `vault_health`
and `vault-validate.py` §1 that had produced four of the vault's five open
frontmatter failures. Nothing in the CLI validated its own output against the
Frontmatter Law it was writing for.

## Takeaways

1. `Format("2006-01-02")` on a UTC time is a code smell. Ask whose day it is.
2. A writer that emits into a schema should have a test asserting the schema,
   not rely on a downstream validator in another repo to notice.
3. When a bug can only reproduce outside CI's environment, the regression test
   has to *construct* that environment — here, an explicit `time.FixedZone`
   rather than whatever the runner's local zone happens to be.
