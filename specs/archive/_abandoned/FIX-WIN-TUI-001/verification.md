# Verification

- [x] Tested \pi\ and \opencode\ in a new PowerShell session.
  - Result: TUI boots perfectly without complaining about missing API keys, and without hanging or exiting silently.
- [x] Tested that \$env:NAN_API_KEY\ is empty after closing the apps.
  - Result: The \	ry/finally\ block correctly cleans up the environment.
- [x] Run \ats tests/pi-config.bats\
  - Result: Bats test suite passes.

## Promotion candidates

- [x] Lesson for the repo's `docs/lessons/`? no: lesson 270 already records what the review found (a security wrapper that breaks the tool it protects gets routed around); the open defects are tickets #2024, #2296 and #2909
- [x] ADR-worthy? no: whether Windows gets an exception to ADR-028 is decided in SEC-009 (#2296), not in this abandoned spec
- [x] New cross-project pattern? no: a single-project fix
