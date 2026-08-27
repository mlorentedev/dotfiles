---
tags: [spec, verification, templates]
created: "2026-08-27"
---

# Verification - AI-032-pi-settings-field-sync

## Evidence

- [x] AC1 (Linux converges) -> `setup-linux.sh` field-sync block +
      `tests/pi-config.bats` "setup-linux.sh pi enabledModels sync updates the model
      list and preserves theme/defaultModel/lastChangelogVersion" (executes the real
      block via `eval` against temp-file fixtures, first run)
- [x] AC2 (other fields untouched) -> same test, same assertion: `jq -e '... and .theme
      == "light" and .defaultModel == "nan/mimo-v2.5" and .lastChangelogVersion ==
      "0.50.0"'` after the sync
- [x] AC3 (idempotent second run) -> same test, second `eval` of the identical block,
      asserts no further change
- [x] AC4 (Windows parity) -> `setup-windows.ps1` field-sync block +
      `tests/pi-config.bats` "setup-windows.ps1 pi enabledModels sync block guards
      both files, preserves other fields, avoids the pipe/ConvertTo-Json
      array-collapse bug" (structural: bats cannot execute pwsh, matching this repo's
      existing PowerShell-parity test pattern)
- [x] AC5 (real block, not reimplemented) -> both new tests extract the actual script
      text (`sed`/`extract_block`) rather than duplicating the logic inline

## Test status

- `bats tests/pi-config.bats` -> 18/18 ok (2 new: #17, #18)
- `bats tests/*.bats` (full suite) -> 1521 results, 1519 ok + 2 skips (`pwsh` not
  available in this environment), exit 0
- `shellcheck setup-linux.sh` -> 19 pre-existing info-level notices, 0 in the new block
- `bash -n setup-linux.sh` -> syntax OK
- ASCII check (`LC_ALL=C grep -n '[^\x00-\x7F]'`) on the new `setup-windows.ps1` block
  -> 0 matches
- Manual smoke test: hand-ran the exact jq sequence against synthetic fixtures (stale
  4-model dst, 6-model src, user `theme`/`defaultModel`/`lastChangelogVersion` set) —
  first run synced and preserved user fields, second run reported "already in sync"
  with no write
- No regressions in existing test suite: yes

## Decisions made during implementation

- Chose to fix this in `setup-linux.sh`/`setup-windows.ps1` rather than as a `dotf
  doctor --fix` check, despite #1256's `checkModelPins` (merged concurrently) landing
  right next to this and deliberately declining to write. The two are not the same
  class of change: `checkModelPins` detects and would repair INVALID/stale entries
  (a judgment call — is the user relying on that entry?), while this adds a
  repo-curated, always-valid entry (never destructive, never surprising). That
  distinction is why the existing `packages.json` reconcile already runs
  unconditionally in `setup-linux.sh` rather than behind a doctor `--fix` flag, and
  this change follows the same precedent rather than the pin-drift check's.
- `-InputObject`, not a pipe, for every `ConvertTo-Json` call comparing arrays —
  see `docs/lessons/lesson-233-...` for the single-element array collapse bug this
  avoids.
- Diff crossed the spec-gate's 50-LOC threshold (129 LOC across 3 files) after the
  work was already implemented and verified; this spec folder was created
  retroactively against issue #1247, mirroring how AI-033 handled the same situation
  minutes earlier in the same session.

## Promotion candidates

Before archiving, flag what (if anything) should be promoted to the vault. If all three are "no", archive in repo is the only persistence.

- [x] Lesson for the repo's `docs/lessons/`? Yes —
      `docs/lessons/lesson-233-piping-a-single-element-array-into-convertto-json-un.md`,
      the `ConvertTo-Json -InputObject` vs. pipe array-collapse gotcha.
- [ ] ADR-worthy decision for the repo's `docs/adr/adr-XXX.md`? No — implementation
      detail within an existing deploy contract, not a new architectural decision.
- [ ] New pattern candidate for `00_meta/patterns/`? No — single-repo, single-file
      pattern (PowerShell JSON round-tripping); not yet observed recurring elsewhere.

## Archive checklist

- [ ] `proposal.md` frontmatter set to `status: archived`
- [ ] Folder moved: `specs/AI-032-pi-settings-field-sync/` -> `specs/archive/AI-032-pi-settings-field-sync/`
- [ ] Bitácora board ticket for this spec moved to Done / closed with PR link (ADR-018)
- [ ] Promotions above executed (if any)
