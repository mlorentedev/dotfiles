---
tags: [spec, verification, templates]
created: "2026-08-16"
---

# Verification - BUG-082-status-poisons-item-reads

## Evidence

| AC | Proof |
|---|---|
| AC1 | `secrets run -- true` ×10 → **10/10 pass** (same binary shape before the fix: 2/10) |
| AC2 | `secrets verify` ×6, 33 secrets each = **198 resolutions, 0 failures** |
| AC3 | `TestSelectBWBackend_NeverCallsStatus` (unlocked / locked / unauthenticated) |
| AC4 | `fakeBWServe` refuses `/object/*`, `/list/object/*`, `/sync` unless unlocked |

### The characterisation that produced the fix

Probe once, then 10 item GETs:

| probe | before | after | |
|---|---|---|---|
| `GET /status` | 0/10 | **10/10** | **poisons** |
| `GET /list/object/folders` | 0/10 | 0/10 | clean |
| `GET /list/object/items?search=` | 0/10 | 0/10 | clean |
| `POST /sync` | 0/10 | 0/10 | clean |

Recovery window ≈ 0.5s.

### Hypotheses falsified, and why recording them matters

Each looked plausible and two were proposed by different sessions as the answer:

| hypothesis | test | result |
|---|---|---|
| connection reuse | `DisableKeepAlives`, 360 requests | 35.0% vs 32.8% — no improvement |
| concurrency | 24 requests at 1/2/4/8-way | 96/96 clean |
| gzip | `DisableCompression` | 30.6% either way |
| rate / spacing | item-only at 0/25/100/300ms | 360/360 clean |

The discriminating variable was whether a `/status` call had happened in the preceding
half-second — i.e. **the observation was causing the failure.** Three sessions measured
this bug the same day and got 1/5, 3/10 and 12/12-clean; all three were correct, and all
three were sampling one window from different distances.

### Precision: reliable, not invariant

The claim needs stating carefully, because counter-evidence exists and a future reader
will find it. The 2026-08-15 handoff records an adversarial review that completed cleanly
at 02:22 against this same poisoned code, with no change on `main`.

The measurements already carried the nuance:

| measurement | failure rate |
|---|---|
| one `/status`, then 10 item GETs, isolated | **10/10** |
| `/status` before *each* of 100 item GETs | **58/100** |
| item GETs with no `/status` at all | **0/360** |

So: a `/status` call **very reliably** poisons the reads that immediately follow, but not
invariably. With 33 secrets to resolve and a `Status()` before each, even the 58% floor
makes a successful `dotf secrets run` vanishingly unlikely — which is what was observed —
while leaving a lucky run possible, which is what the 02:22 review was.

What IS invariant across every measurement: **no failure was ever observed without a
preceding `/status`** (0/360, at four spacings). That is the property the fix relies on
and the one the guard asserts. Stating it as "deterministic" would have been an
overstatement that the next reader could have falsified in one counter-example, and
discarded a correct diagnosis along with it.

### Live before/after

```
FIXED build   : secrets run  PASS=10 FAIL=0  of 10
before (#1007): secrets run  PASS=2  FAIL=8  of 10

6 consecutive verify runs, fixed build:
run1..run6: 33 ok, 0 missing, 0 failed
```

One `verify` run mid-session reported `5 ok, 28 failed` — traced to the *previous* loop
having run the old binary, which poisoned the daemon for the command that followed.
Recorded because it is confirmation of the mechanism, not noise: the old binary damages
the next process's reads, not only its own.

## Test status

- `go build ./... && go vet ./... && go test ./...` — every package ok
- `golangci-lint run` (pinned 2.12.2) — **0 issues**
- No regressions: the full `internal/cmd` and `internal/secrets` suites pass unchanged

## Decisions made during implementation

- **Removed the trigger rather than retrying the symptom.** A bounded retry on HTTP 500
  would have produced passing commands and hidden the cause, which is how this bug
  survived weeks of observation in the first place.
- **The guard asserts a negative** (`selection never calls /status`) rather than a
  positive (`selection calls folders`). The invariant is about what must not happen, and
  it stays true if the probe endpoint is ever changed again.
- **The locked-daemon response was not verified live**, because locking the shared daemon
  would have disrupted two other sessions mid-work. Handled structurally instead: any
  non-success selects the shellout.

## Promotion candidates

- [x] Lesson for `docs/lessons.md`? **Yes** — *the observation was the cause*. A liveness
      probe that breaks the operation it authorises is invisible to every measurement,
      because every measurement includes the probe. Generalises past this daemon: it is
      the shape of any health check with a side effect.
- [ ] ADR-worthy? No.
- [ ] Pattern for `00_meta/`? Not yet — revisit if a second instance appears elsewhere.
