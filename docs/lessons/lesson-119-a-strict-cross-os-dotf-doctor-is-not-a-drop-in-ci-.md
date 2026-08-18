---
id: lesson-119-a-strict-cross-os-dotf-doctor-is-not-a-drop-in-ci-
type: lesson
status: active
created: "2026-06-21"
owner: manu
tags: [lesson, dotfiles]
---

# Lesson 119: A strict cross-OS `dotf doctor` is not a drop-in CI gate for a lenient platform-specific healthcheck

**Context**: CLI-018 retired Windows `healthcheck.ps1`. The `test-windows` CI job's "Run healthcheck.ps1" step was repointed to `dotf doctor` followed by `exit $LASTEXITCODE`.

**Problem**: The step false-red'd. `dotf doctor` (the cross-OS Go diagnostic) FAILs on a partial CI runner — `HOME` unset, `wget`/`terraform`/`java` are env-contract *required* binaries, and opencode/pi/git-hooks-dispatcher are absent — so it exit-codes 1. The retired `healthcheck.ps1` had been Windows-aware and lenient about exactly those checks, so it exited 0 in the same environment. Renaming the step silently swapped a lenient checker for a strict one against an environment that never satisfied the strict one — same intent, different exit semantics.

**Solution**: Removed the live gate from `test-windows` rather than tuning the Go diagnostic's Windows-awareness (a behaviour change, out of scope for a delete+repoint PR). This mirrors Linux, which runs **no** live diagnostic gate — `dotf doctor` is covered by `go test` + structural bats, and `setup-windows.ps1` still runs end-to-end (its own *non-fatal* post-setup `dotf doctor` prints health without gating).

**Rule**: When a CI gate's underlying tool changes from a lenient platform-specific script to a strict cross-OS one, re-derive what the gate can actually assert in the CI environment — a partial runner will not satisfy a full-install diagnostic, so gating on its exit code false-reds every PR. Check what the *other* OS's CI does (parity) before inventing a gate; a diagnostic built for humans post-setup is normally validated in CI by unit + structural tests, not by gating on its live exit code.
