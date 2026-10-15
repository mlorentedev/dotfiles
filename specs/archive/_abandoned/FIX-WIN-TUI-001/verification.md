# Verification

- [x] Tested \pi\ and \opencode\ in a new PowerShell session.
  - Result: TUI boots perfectly without complaining about missing API keys, and without hanging or exiting silently.
- [x] Tested that \$env:NAN_API_KEY\ is empty after closing the apps.
  - Result: The \	ry/finally\ block correctly cleans up the environment.
- [x] Run \ats tests/pi-config.bats\
  - Result: Bats test suite passes.

## Abandoned

The 2026-10-13 archive review (`review.md`) returned FAIL. The owner then chose ConPTY behind `interactiveChildSupported()` (#1650, recorded on #2019 and #2296), which removes the Base64 wrapper this proposal describes. Archiving this spec after ConPTY lands would archive the opposite of what shipped, so it is abandoned and the work moves to a new spec under #1650. Acceptance there needs the Windows box.

## Promotion candidates

- [x] Lesson for `docs/lessons/`? no: lesson 270 already states the rule this wrapper broke (a security wrapper must not drop its property to restore the tool).
- [x] ADR-worthy decision for `docs/adr/`? no: the ConPTY decision keeps ADR-028 intact, so there is no exception to record; it lives on #2019 and #1650.
- [x] New pattern candidate for `00_meta/patterns/`? no: single-repo Windows PTY detail, not a cross-project pattern.
