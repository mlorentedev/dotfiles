---
tags: [spec, tasks, vault, safety-net]
created: "2026-05-19"
---

# Tasks - SDD-006-vault-integrity-check

## Setup

- [x] Branch: `feat/SDD-006-vault-integrity-check` (off main).
- [x] Vault entry in `11-tasks.md`.
- [x] Manually repaired 4 prior `[BS-n]` corruption instances in `11-tasks.md` (the symptom that surfaced the need).

## Implementation

- [x] `scripts/check-md-escapes.sh`: standalone scanner, accepts file/dir, exit 0/1/2 contract, set -euo pipefail, shellcheck clean.
- [x] `tests/check-md-escapes.bats`: 9 cases including the dotfiles self-test (catches future corruption in this repo at PR time).
- [x] Lesson captured in vault `dotfiles/90-lessons.md`: "Incident → guard pattern (red-team thyself)".

## Verification

- [x] Self-test passes: dotfiles repo markdown is clean.
- [x] Full bats suite green.
- [x] shellcheck --severity=error clean on new script.
- [ ] Fill `verification.md`.
- [ ] PR opened.

## Closing

- [ ] Tick SDD-006 in vault `11-tasks.md` post-merge with PR link.
- [ ] Archive spec to `specs/archive/SDD-006-vault-integrity-check/` post-merge.
