---
id: "CLI-064-doctor-profile-heal"
type: spec
status: archived # draft | implementing | verifying | archived
created: "2026-08-28"
issue: "mlorentedev/dotfiles#531"   # repo#NNN — GitHub issue / Project item that tracks this spec
tags: [spec, proposal, doctor, windows, profile]
template_version: "1.0"
---

# CLI-064-doctor-profile-heal

## Why

CLI-018 retired the Windows `scripts/doctor.ps1` and consolidated every post-setup
diagnostic onto `dotf doctor`. The retired `doctor.ps1 -Fix` used to invoke
`scripts/profile-heal.ps1`, the BUG-020 repair for a PowerShell `$PROFILE` that
setup's marker splice had compounded to 26 MB / 689K lines of duplicated
sections. After the consolidation `checkProfileFiles` (`checks_profile.go`)
tested existence only: a 26 MB profile reported PASS, and nothing on the doctor
path could heal it. The corruption *source* was removed (BUG-022's index-based
splice), so auto-heal is convenience — doctor blessing a broken profile is the
defect. Issue #531.

## What

On Windows, `dotf doctor` FAILs the PowerShell profile when it shows either
BUG-020 signal — larger than 1 MB, or more than one `# >>> DOTFILES PROFILE >>>`
/ `# <<< DOTFILES PROFILE <<<` pair — naming `<SCRIPTS_DIR>\profile-heal.ps1` as
the remedy, with `SCRIPTS_DIR` resolved through the env contract (never a
literal path, so WIN-013's move of `~\scripts` does not break it). Under
`--fix` doctor runs that script through the `System` seam and re-reads the
profile: the fix is reported by consequence (the signals are gone), never by
the script's exit code. `profile-heal.ps1`'s own marker threshold is aligned to
the same rule (exactly one pair is healthy) so doctor never flags a profile the
script declines to heal.

## Out of scope

- Linux profiles (`.bashrc`/`.zshrc`) — the splice defect is Windows-only.
- The parser-error heuristic (`profile-heal.ps1`'s third signal) — it needs a
  PowerShell parser and stays in the script.
- Pruning the profile's non-dotfiles content.

## Risks / open questions

- A OneDrive-redirected Documents folder: `findPowerShellProfile` already
  searches both roots; the heal path is passed absolute.
- A heal that runs but leaves a signal in place (partial rewrite) is reported as
  a FAIL with the script named, not as a fix — verified by consequence.

## Acceptance criteria

- [x] AC1 — a Windows profile over 1 MB or with more than one marker pair FAILs
  with a message naming the heal script's contract path; a healthy one PASSes.
- [x] AC2 — the heal path is `<SCRIPTS_DIR>\profile-heal.ps1` read from the
  contract (`SCRIPTS_DIR` env, then the contract default), never a literal.
- [x] AC3 — under `--fix` the heal runs through the `System` seam and doctor
  reports Fix only when the re-read profile carries no signal; a heal that
  changes nothing stays a FAIL.
- [x] AC4 — `profile-heal.ps1` and doctor agree on the marker threshold (>1).
- [x] AC5 — non-Windows behaviour is byte-identical (existence checks only).

## References

- Bitácora board: #531
- BUG-020 (profile compounding), BUG-022 (index-based splice), CLI-018 (doctor consolidation)
- `scripts/profile-heal.ps1`, `cli/internal/doctor/checks_profile.go`

<!-- archived 2026-08-28 — PR: https://github.com/mlorentedev/dotfiles/pull/1353 -->
