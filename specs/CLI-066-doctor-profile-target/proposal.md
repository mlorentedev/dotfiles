---
id: "CLI-066-doctor-profile-target"
type: spec
status: implementing # draft | implementing | verifying | archived
created: "2026-08-29"
issue: "mlorentedev/dotfiles#1364"   # repo#NNN — GitHub issue / Project item that tracks this spec
tags: [spec, proposal]
template_version: "1.0"
---

# CLI-066-doctor-profile-target

> **Naming**: file lives at `<repo>/specs/CLI-066-doctor-profile-target/proposal.md`. `CLI-066-doctor-profile-target` is `AREA-NNN-slug` (e.g. `TOOL-001-secret-drift`).

## Why

`dotf doctor` (CLI-064, #1353) decides whether the PowerShell profile is
corrupted from the file its own `findPowerShellProfile` enumerates — four
hardcoded roots under `~\Documents` and `~\OneDrive\Documents` — while `--fix`
runs `scripts/profile-heal.ps1`, which resolves `$PROFILE` itself and heals
**that**. On a box whose Documents folder is redirected anywhere else (a
corporate `User Shell Folders\Personal`, a second drive), doctor measures one
file, or none, and the heal rewrites another: detect and heal split. Raised by
the adversarial review of CLI-064 (Major, THEORETICAL — not reproduced;
`specs/archive/CLI-064-doctor-profile-heal/review.md`, finding 1), with three
Minors from the same review that this closes alongside.

## What

- **One source of truth for the target.** Doctor asks PowerShell for it —
  `pwsh -NoProfile -Command '$PROFILE'` through the bounded seam
  (`CommandOutputBounded`) — and measures that file. The four-root enumeration
  survives only as the fallback when `pwsh` is not on PATH (then the heal could
  not run anyway) and the row says which of the two answered.
- **The heal takes the path it must heal.** `profile-heal.ps1` gains
  `-ProfilePath`; doctor passes the file it measured, so `--fix` and the
  measurement agree by construction. Run by hand without the parameter, the
  script defaults to `$PROFILE` as before.
- **Thresholds linked, not duplicated.** A Go test reads the script and asserts
  its `-gt 1MB` and `-gt 1` literals beside doctor's `profileMaxBytes` and
  marker rule, so the two cannot drift silently (Minor 1).
- **The heal runs bounded** — `CommandOutputBounded`, like every other doctor
  probe that shells out (Minor 2).
- **What `--fix` does is said where it is offered:** the FAIL line and the
  script's synopsis state that the heal rewrites the profile from the SSOT and
  keeps everything else only in the backup it makes first (Minor 3, documented
  rather than built: preserving an outside-marker tail is a different feature,
  and BUG-020's profiles carried nothing worth preserving).

## Out of scope

- Preserving user content outside the dotfiles markers across a heal
  (documented; a follow-up if a real profile ever carries any).
- Windows PowerShell 5.1's `$PROFILE` (`WindowsPowerShell\`): setup deploys the
  pwsh 7 profile; doctor measures what `pwsh` reports, as before it measured
  the first existing candidate.
- The PowerShell-side `doctor.ps1 -Fix` twin — retired in CLI-064.

## Risks / open questions

- `pwsh` present but slow to answer (`-NoProfile` avoids the profile itself;
  measured cold start here is under 1 s). RESOLVED: bounded at 10 s; on a
  timeout doctor falls back to the enumeration and says so in the row.
- `$PROFILE` reported for a file that does not exist yet (fresh box, setup not
  run). RESOLVED: that is the missing-profile FAIL doctor already emits, now
  naming the path pwsh gave instead of four guesses.
- A `$PROFILE` answer with a trailing newline or CRLF. RESOLVED: trimmed, and
  the test feeds the CRLF form.

## Acceptance criteria

Observable outcomes. Each must be testable.

- [x] AC1 — with `pwsh` on PATH, doctor measures the file `pwsh -NoProfile -Command '$PROFILE'` names, even when it lies outside the four enumerated roots; the row names that path.
- [x] AC2 — without `pwsh`, doctor falls back to the enumeration and the row says the target was enumerated, not resolved.
- [x] AC3 — `--fix` invokes `pwsh -NoProfile -File <SCRIPTS_DIR>\profile-heal.ps1 -ProfilePath <the measured file>` through the bounded seam, and re-measures the same file.
- [x] AC4 — `profile-heal.ps1 -ProfilePath <file>` heals that file and only it; without the parameter it targets `$PROFILE` as before.
- [x] AC5 — a Go test fails if the script's size or marker threshold literals stop matching doctor's constants.
- [x] AC6 — on the Windows work box: `dotf doctor` names the pwsh-resolved profile path in its row, and `profile-heal.ps1 -ProfilePath` against a scratch corrupted copy heals the copy while the real profile is untouched.

## References

- Bitácora board: #1364. Prior spec: `specs/archive/CLI-064-doctor-profile-heal/` (review finding 1 and the three Minors).
- `cli/internal/doctor/checks_profile.go`, `scripts/profile-heal.ps1`, `tests/profile-heal-ps1.bats`.
