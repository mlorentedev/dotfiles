---
tags: [spec, tasks, templates]
created: "2026-08-09"
---

# Tasks - BUG-062-crystallize-refuse-yaml

> TDD order. One task = one focused commit.
>
> `[P]` = independent of another unchecked task; `[AC<n>]` = satisfies acceptance criterion `<n>`.

## Setup

- [x] Branch created from main: `fix/crystallize-refuse-yaml-wrapped`
- [x] `proposal.md` is complete and acceptance criteria are testable
- [x] No open questions left in `proposal.md` "Risks / open questions"

## Implementation

- [x] [AC1] [AC2] Failing bats suite: block-scalar fixtures at four- and six-space
      indent, asserting non-zero exit, byte-identity and YAML parseability
- [x] [AC3] Failing case: plain markdown still stamps and keeps the handoff last
- [x] [AC1] `is_yaml_block_scalar` + refusal at the top of `process_project` (sh)
- [x] [AC4] `--all` counts a refusal as skipped; single-project mode exits 1
- [x] [AC5] Same guard in the `.ps1` twin (`Test-YamlBlockScalar` + `throw`),
      with its `--All` catch incrementing `$skipped`
- [x] [P] [AC5] Structural bats assertions for the PowerShell twin
- [x] Re-run issue #857's own repro against the guarded script

## Closing

- [x] Every acceptance criterion is covered by at least one test
- [x] `features.json` entries with non-vacuous verification commands
- [x] `shellcheck` passes; `bash -n` and `zsh -n` pass
- [x] Full `bats tests/*.bats` shows no new failures
- [x] `verification.md` filled in with output produced this session
- [ ] PR opened referencing this spec folder, body carries `Refs #857` — **not** a
      closing keyword: #857 closes when #490's port makes the shape work
