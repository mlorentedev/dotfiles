---
id: "lesson-413-a-check-that-needs-the-vault-belongs-where-the-vault-is"
type: lesson
status: active
title: "A check that needs the vault belongs in doctor, not in a test CI cannot run"
created: "2026-10-13"
---

# A check that needs the vault belongs in doctor, not in a test CI cannot run

## Context
Skills are written in the vault and copied into committed records under `harness/skills/` by
`compile-harness.sh --refresh`. On 2026-10-12 the vault's `teardown` skill changed and nobody
refreshed the record. A day later msi's checkout, refreshed by setup, showed the vault's version as
an uncommitted change in `harness/skills/teardown/SKILL.md`, which read as a stray edit to discard.
`main` was the stale side.

## The Trap
The comparison already existed: `tests/harness-doctrine-records.bats` runs `--refresh` against the
real vault and fails if anything changes. It had been failing since the vault edit. But it needs a
vault, so CI skips it, and on a laptop it is one red line among a full bats run. Doctor's own gate,
`compile-harness.sh --check`, validates the records offline and reads no vault by design. The one
check that could see the drift lived where nothing runs it routinely, so the first signal was a
dirty checkout on another machine, pointing the wrong way.

## The Solution
Put the comparison where the vault is: doctor runs on every machine that has one. Each committed
record carries `generated_from` (the vault path) and `generated_sha` (16 hex of the source's
sha256), so doctor hashes the source and compares, with no script and no refresh. It warns rather
than fails, because either side can be the older one, and the warning names both remedies: refresh
the records in a PR, or pull the vault. When a test needs state CI does not have, ask which
machine-level check owns that state, and move the assertion there.
