---
spec: "AI-030-pi-packages-manifest"
verdict: "PASS-WITH-GAPS"
reviewed_sha: "fb88359c3177728896f936c6a047a6ebda290c70"
reviewer: "nan/glm5.3-flash"
date: "2026-09-30"
---

## Adversarial review

**Scope**: AI-030-pi-packages-manifest (round 2, after the round-1 FAIL's contract amendment)
**Sources**: `specs/AI-030-pi-packages-manifest/{proposal,tasks,verification,features.json}`; `git diff 180116b0a1dee71b57b8ec64ff40813e711e1fb1...HEAD` (launcher-resolved base; the spec's own work is `9c44d7a7` → HEAD plus the HARNESS-139 re-point `e4b39f55`/`4dad5ead` and the round-1 follow-ups `354539fa`…`fb88359c`).

### Spec and task alignment

- The round-1 FAIL was caused by a spec-vs-code drift: the proposal declared removal out of
  scope while `dotf pi packages apply` (HARNESS-139) removes undeclared packages. The
  amendment of 2026-10-01 closes it: AC12 now makes removal a criterion, the "Out of scope"
  contradiction is gone, and the Risks section states the accepted cost (owner, 2026-10-01).
  Contract files were edited before this round, not during it.
- Round-1 F-03 (stale `1..16` / retired `verify-reconcile.sh` shown as current) — applied:
  verification.md now shows `1..14` and labels the old block historical. F-06 (README
  install-only) — applied: `ai/pi/packages.json` `$comment` and `ai/pi/README.md` describe the
  bidirectional reconcile.
- Tasks `[x]` claims spot-checked against the diff: the shell reconcile loops are gone from
  both twins (bats test 10 "neither twin carries the reconcile loop any more"), the Go
  command exists at `cli/internal/pi/{packages,apply}.go` + `cli/internal/cmd/pi.go`, the
  doctor check at `cli/internal/doctor/checks_pi_extensions*.go`. No `[x]` was found without
  diff evidence.

### Verification performed this session (evidence, not assertions)

- `bats tests/pi-packages.bats` → `1..14`, all ok, fresh at `fb88359c`.
- `go test ./internal/pi ./internal/cmd -count=1` → ok; `go test ./internal/doctor/ -run TestPiExtensions` → ok (6 named cases).
- **Mutation 1** — replaced `npm:pi-effort@0.0.8` with `npm:pi-effort` in the manifest: the
  pin guard test failed with `these sources carry no pinned version: npm:pi-effort`. Reverted.
- **Mutation 2** — emptied an entry's `why`: "every entry says what it is for" failed. Reverted.
- **End-to-end behaviour** — built `cli/cmd/dotf` and ran `pi packages check` against a
  synthetic agent dir: an undeclared live `npm:ghost-pkg@1.0.0` plans `remove`; a live
  `npm:pi-effort@9.9.9` (declared at another version) is kept while the declared source is
  installed — exactly the documented identity semantics; malformed live settings → error
  (exit 1); empty manifest → `refusing to read that as "remove everything"` (exit 1).
- `shellcheck setup-linux.sh` → 15 findings, **0 in the new block (lines 820–869)**;
  `bash -n` and `zsh -n` both pass; `setup-windows.ps1` has 0 non-ASCII lines (AC10).
- AC3 verified: no `packages` key in `ai/pi/settings.json`; neither twin writes the array.
- No `[AGENT-DRAFT]`/`[AGENT-SUGGESTION]` tags in the spec artifacts.

### Findings

| Severity | Reality | Area | Finding | Evidence | Test (named, or UNTESTED) | Fix location (code / tests / spec / vault) |
|----------|---------|------|---------|----------|---------------------------|---------------------------------------------|
| Minor | REAL | Windows parity | `setup-windows.ps1` has no `~/.local/bin/dotf.exe` fallback for the reconcile call, unlike the Linux twin's PATH-then-`~/.local/bin` probe. Consequence is bounded: an explicit `Write-Warn`, the run continues, and the next run with dotf on PATH converges — so I judge it below round-1's Major, but it is a real asymmetry | code read, `setup-windows.ps1:1281` vs `setup-linux.sh:838-858`; open ticket #1925 | UNTESTED | code (tracked as #1925 — carry into a follow-up, do not edit the contract) |
| Minor | REAL | reconcile completeness | `LiveSources` silently drops a live entry that is neither a string nor an object with `source`, so such an entry can neither be removed nor counted | code read, `packages.go` LiveSources; open ticket #1926 | UNTESTED | code (tracked as #1926) |
| Minor | REAL | spec record | `tasks.md` claims shellcheck "20 findings before, 20 after"; the current shellcheck reports 15 (version drift between then and now). The load-bearing half of the claim — none in the new block — verified true | `shellcheck -f gcc` count this session | UNTESTED | none — historical statement in a contract file; editing tasks.md would invalidate this review, so record it here and leave it |
| Minor | REAL | spec record | proposal "What" and tasks say "nine entries"; the manifest now declares ten (`@gtrabanco/pi-nan-provider` joined via #1789) | manifest read | UNTESTED | none — same contract-set rule as above; the count claim is historical, the guard (≥1 entry, pinned, why) is what binds |
| Question | SPECULATIVE | Windows `--pi` | The Windows apply call passes no `--pi`, so `resolvePi` falls back to `~/.local/bin/pi` then PATH. On Windows `pi` is a real binary (the locked-vault shell function is a Linux construct), so this should be fine — confirm pi resolves on a real Windows box | code read only, no Windows repro available | UNTESTED | tests (fold into #1925's fix if it does not resolve) |

AC11's real-machine evidence (the `pi -p` exit 1 → 0 flip and the live `dotf doctor` FAIL/FIX
sequence of 2026-08-26) is UNVERIFIED this round — it is machine state, the implementing
session measured it, and re-running the repair against the live agent is the owner's call.
The check's logic is covered by the 6 named doctor unit tests, which pass.

### Test traceability

AC1 (bats 1/4/5), AC2 (bats 2 + 3, mutation-proven this session), AC3 (bats 6 + 7), AC4/AC5
(`TestNewPlan`, `TestApplyConvergesAndASecondRunCallsNothing`), AC6 (`TestLiveSourcesReadsBothEntryForms`,
`TestIdentityIgnoresTheVersion`), AC7 (`TestPiPackagesApplyWithoutPiWarnsAndExitsZero`,
`TestPiPackagesApplyWithoutNpmWarnsAndExitsZero`, `TestPiPackagesApplySkipIsFirstAndLoud`),
AC8 (`TestLoadManifestRefusesWhatItCannotRead`, `TestPiPackagesCheckRefusesAnUnreadableManifest`),
AC9 (bats 9), AC10 (bats 8 + the 0-non-ASCII count), AC11 (`TestPiExtensions*`, 6 cases),
AC12 (`TestPiPackagesApplyRemovesThenInstalls` + `TestLoadManifestRefusesWhatItCannotRead`,
and reproduced end-to-end this session with a synthetic agent dir). No criterion rests on the
implementer's word alone. Negative paths exist for every dangerous input I could think of:
unpinned source, duplicate source, empty why, empty manifest, malformed manifest, malformed
live settings, missing pi, missing npm, missing dotf, retire-path traversal (rejected in
`validate`).

### Evaluator rubric

| Dimension | Grade (A-D) | Rationale (one line) |
|-----------|-------------|----------------------|
| Correctness        | A | All 12 ACs verified including negative and abuse paths; removal/malformed/empty reproduced end-to-end on a fresh build |
| Verification       | B | features.json commands are reproducible and I re-ran their substance; the real-machine AC11 evidence is historical and machine-bound |
| Scope              | B | The spec's own work matches the amended proposal exactly; the launcher's diff base spans other merged specs' work, which is scope of the review, not of this change |
| Reliability        | A | Degrades loudly on every missing tool, never aborts setup, idempotent second run, retire never deletes, failed calls counted not fatal |
| Maintainability    | A | Small functions, CC well under 10, comments carry the WHY (why not settings.json, why $PI_BIN, why not "any undeclared extension") |
| Handoff-readiness  | A | Round-1 dispositions recorded with tickets, lesson-231 exists, promotion candidates answered, README/manifest `$comment` self-documenting |

### Verdict

**PASS WITH GAPS** — no Blockers, no REAL Majors, rubric all B/A; four REAL Minors, each
carrying a disposition (two ticketed as #1925 and #1926, two recorded above as historical
claims that the contract-staleness rule forbids fixing now).

### Recommended next steps

For the implementer to disposition in `verification.md` (the contract set is closed under a
passing verdict — an edit to `proposal.md`, `tasks.md` or `features.json` would invalidate
this review):

- Disposition #1925 and #1926 (apply or carry into a follow-up PR); the Windows `--pi`
  question above folds into #1925's fix.
- Leave the shellcheck-count and nine-vs-ten claims as recorded; they are historical and the
  binding guards are live and mutation-proven.
- The manual smoke test against real `pi` remains deliberately unperformed, as verification.md
  states — that is the owner's call, and `dotf spec archive` may proceed without it.

**Archive advisable**: yes — `dotf spec archive` can run against this review.
