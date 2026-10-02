# Verification

- [x] Tested \pi\ and \opencode\ in a new PowerShell session.
  - Result: TUI boots perfectly without complaining about missing API keys, and without hanging or exiting silently.
- [x] Tested that \$env:NAN_API_KEY\ is empty after closing the apps.
  - Result: The \	ry/finally\ block correctly cleans up the environment.
- [x] Run \ats tests/pi-config.bats\
  - Result: Bats test suite passes.
