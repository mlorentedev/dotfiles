---
id: lesson-208-the-health-probe-was-the-illness-a-liveness-check-
type: lesson
status: active
created: "2026-08-15"
owner: manu
tags: [lesson, dotfiles]
---

# Lesson 208: The health probe was the illness: a liveness check that breaks the operation it authorises

**Context**: BUG-082 (#988) — `dotf secrets verify` and `run` failed against the `bw serve` daemon with `bw serve returned no parseable envelope: invalid character 'I'`. It had been open for weeks, described as intermittent, and three agent sessions measured it on the same day.

**Problem**: two defects, and the second is why the first survived so long.

The read path asked the daemon `GET /status` before every field read, to decide whether it was unlocked. That status call is what broke the read: for roughly half a second afterwards, `GET /object/item/<id>` returns HTTP 500. Resolving 33 secrets meant 33 status calls, each poisoning the read that immediately followed it. **The probe broke the very call it was authorising**, so the more carefully the code checked before reading, the more reliably it failed.

That made the bug invisible to measurement, because every measurement contained the probe. The three sessions got 1/5, 3/10 and 12/12-clean, and all three were right — they were sampling one 0.5s window from different distances. Three plausible hypotheses were proposed and all three were wrong, each falsified only by a controlled experiment: connection reuse (`DisableKeepAlives`: 35.0% vs 32.8% over 360 requests), concurrency (24 requests at 1/2/4/8-way parallelism: 96/96 clean), and request rate (item-only loops at 0/25/100/300 ms: 360/360 clean).

The second defect is why nobody got that far for weeks. `call()` decoded the response straight off the wire and reported only `no parseable envelope: invalid character 'I'`. The `'I'` was the first byte of `Internal Server Error` — an HTTP 500 with a 21-byte plain-text body. The status code was never read, so a server error was reported as a parser error, and everyone looked at the parser.

**Solution**: stop asking for a status string and ask the question that is actually needed — *will you serve a read?* — via `GET /list/object/folders`, measured clean alongside `/sync` and `/list/object/items?search=`. Any non-success selects the CLI shellout, so correctness does not depend on knowing exactly how a locked daemon refuses. `secrets run` went from 2/10 to 10/10, and six `verify` runs resolved 198 secrets with zero failures. Separately, the decode error now names `(HTTP 500, 21 bytes)` — status and size only, never body bytes, since a 200 that merely failed to parse can still be a credential. Upstream is a `switchMap`/`ReplaySubject` disposal race (bitwarden/clients#20951); the deterministic trigger was reported there, and the affected-versions list was extended by measurement rather than inherited.

**Rule**: before trusting a health check, ask whether observing changes the thing observed. A probe with a side effect is the hardest bug shape to see, because every symptom points at the subject and every measurement carries the instrument — and the usual response, checking *more* carefully before acting, makes it worse. Two habits follow. First, prefer probes that are the operation in miniature over probes that ask about it: "will you serve a read" is answerable by trying, where a status string is a claim about a different subject that may not predict the one you care about. Second, **never let a transport error be re-reported as a parse error** — read the status code before the body, and put it in the message. A diagnostic that discards the diagnosis costs weeks, and it is a bug in its own right, not merely a missing nicety.

A last note on claims. The first write-up called the poisoning "deterministic" on the strength of a 10/10 measurement, and a peer produced a review that had completed cleanly against the same code. The measurements had said so all along — 10/10 in one trial, 58/100 in another. The durable claim is the narrower one: *no failure was ever observed without a preceding `/status`* (0/360, four spacings). A claim that survives its own counter-examples is worth more than a stronger-sounding one that a single lucky run can discredit — along with the correct diagnosis attached to it.

**Tags**: `observability`, `verification`, `secrets`, `false-positive`, `bitwarden`, `debugging`
