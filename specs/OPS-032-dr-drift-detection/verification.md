---
tags: [spec, verification, templates]
created: "2026-08-19"
---

# Verification - OPS-032-dr-drift-detection

## Evidence

Produced 2026-08-19, in this session.

- AC1/AC2/AC3 (the root's drift check) -> `go test ./internal/secrets/ -run 'Drift|Recipient'`; observed
  FAILING against a scratch registry pointing at the REAL key with a wrong declaration,
  and passing with the right one.
- AC4 (the manifest) -> live structural probe of the real escrow: `count=177`,
  `max_revision=2026-08-15T08:21:32.256Z`, `digest=0557ec4d…`.
- AC5 (drift seen) -> **demonstrated on the real vault before the tests were written.**
  `dotf doctor`, in the same run where the age check reported the escrow "present and
  fresh":

  ```text
  [WARN] DR escrow no longer describes the vault: 1 item(s) added since the escrow was
         written. The escrow describes the vault as of 2026-08-15T08:21:32.256Z.
  ```

- AC6 (the SKIPs) -> both observed live and unit-pinned, including an unreadable
  manifest reported as unreadable rather than as absent.
- AC7 (runbook) -> RECOVER step 1 verifies the restored key before the chain continues.
- AC8 -> `go vet` and `GOOS=windows go vet` both clean.

## Test status

- 17 Go packages, 0 FAIL. `golangci-lint` at the pinned 2.12.2 — 0 issues. bats — 0 failures.
- `dotf secrets verify` — 34 ok, 0 missing, 0 failed.

## Source equivalence, measured rather than assumed

The manifest digests the EXPORT's items; the freshness check digests the LIVE listing.
If those two sets differed systematically, identical vaults would produce permanent
false drift. Measured before wiring: 177 ids in the escrow, 178 live, **0 only in the
escrow, 1 only live, 177 in both** — one addition, no divergence.

## Audit round (pre-PR), and what it found

Run because the operator asked whether this work was introducing defects, after a peer
repo found three in ours. Three real problems in this change, all fixed before the PR:

1. **`dotf secrets backup` exited non-zero when only the MANIFEST failed**, while
   printing "the escrow itself is good" — a message and an exit code disagreeing about
   the same event, which is this repository's whole subject. Fixed by making the
   SIGNATURE carry the distinction (`path, manifestWarn, err`) rather than a sentinel
   error: a sentinel teaches one call site and the next re-imports the bug. The
   compiler then found the two call sites for us.
2. **A read error on the manifest was reported as absence**, sending the reader to run
   `backup` — which does not fix a permission error. The escrow check twenty lines above
   lectures about exactly this ("a stat error is NOT proof of absence").
3. **The recipient validator echoed the rejected value.** The likeliest way to reach that
   branch is pasting the PRIVATE key by mistake, and an error reaches scrollback and CI
   logs. It now names the shape and never the content, asserted by a test.

## Decisions made during implementation

- One reduction (`ManifestFromItems`), two producers. Two spellings of "what a manifest
  is over" would produce permanent false drift between two descriptions of one vault.
- A drift mismatch is WARN, not FAIL: it is expected after any mutation and one command
  fixes it. A section that goes red after every `rotate` is one people scroll past.
- The offline USB copy is deliberately NOT compared. It is not plugged in, and a check
  that cannot see its subject must not report on it.

## Promotion candidates

- **Instrument validity is not free even when you know about it.** Three mutations this
  session were invalid rather than informative — two failed to COMPILE and one edited
  documentation instead of code. Each looked exactly like a vacuous guard. Lesson 212
  says this; the session then demonstrated it three more times, which suggests the
  lesson needs a mechanical companion rather than more prose.
