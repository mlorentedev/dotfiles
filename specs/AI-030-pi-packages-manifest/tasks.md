---
tags: [spec, tasks, templates]
created: "2026-08-25"
---

# Tasks - AI-030-pi-packages-manifest

> **Order of record.** The implementation landed in `2c20332` before this spec
> folder existed. The Discipline Gate trigger was detected while measuring the
> diff for the PR, not while scoping — 94 executable lines across the two setup
> scripts, plus a new deployed config schema. The spec was then written against
> the work rather than ahead of it, and saying so is the point: a tasks list
> back-dated to look like TDD would make this folder a worse record than no
> folder. Tests and implementation *were* written together; the spec was not.

## Setup

- [x] Branch created from main: `feat/pi-packages-manifest`
- [x] `proposal.md` is complete and acceptance criteria are testable
- [x] No open questions left blocking implementation in `proposal.md`
      (one is recorded as deliberately open — whether pi auto-installs
      user-scoped packages at startup — and the design does not depend on it)

## Implementation

- [x] [AC1] [AC2] Declare `ai/pi/packages.json`: nine entries, each pinned to an
      explicit version, each carrying a `why`
- [x] [AC2] Guard that refuses an unpinned source, **and** a second test proving
      that guard rejects `npm:pi-effort` and accepts
      `npm:@ayulab/pi-rewind@0.4.6` — an assertion whose regex accepts
      everything is not an assertion (#1203)
- [x] [AC1] Guards for valid JSON, no duplicate `source`, non-empty `why`
- [x] [AC3] Guard that `ai/pi/settings.json` declares no `packages` key, and
      that neither setup script writes the array directly
- [x] [AC4] [AC5] [AC8] Reconcile block in `setup-linux.sh`: read the manifest
      with `jq -er`, read the live array, install the set difference through
      `$PI_BIN`
- [x] [AC6] Read both the string and object entry forms from the live array
- [x] [AC7] Degrade to a warning when pi or jq is absent; never abort setup
- [x] [AC9] Guard that the Linux path installs through `$PI_BIN`, never the
      `pi` shell function
- [x] [AC10] Mirror the block in `setup-windows.ps1`, with parity guards
- [x] [AC5] [AC6] Verify behaviour by driving the real block, extracted from
      `setup-linux.sh` by anchor, against a stubbed `pi` — four scenarios

## Closing

- [x] Every acceptance criterion is covered by at least one test or a recorded
      scenario run
- [x] Every acceptance criterion has an entry in `features.json` with a
      non-vacuous verification command
- [x] Lint passes (`shellcheck setup-linux.sh`: 20 findings before, 20 after,
      none in the new block)
- [x] Both shells parse (`bash -n`, `zsh -n`)
- [x] PowerShell adds no non-ASCII (10 non-ASCII lines before, 10 after)
- [x] No unrelated changes in the diff
- [x] `verification.md` filled in
- [ ] PR opened referencing this spec folder
- [ ] Independent adversarial review before archive (`dotf spec review`) — the
      implementing session cannot be the reviewer

## Machine-readable features

`features.json` is the harness-facing contract. The agent may not write
`"state": "passing"`; only the harness may, after running `verification` and
capturing exit code 0.
