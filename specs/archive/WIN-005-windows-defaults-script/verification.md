---
tags: [spec, verification, templates]
created: "2026-05-13"
---

# Verification - WIN-005-windows-defaults-script

## Evidence

Map every acceptance criterion from `proposal.md` to concrete proof (commit hash, test name, or observed behavior).

- [x] AC1 (script exists, idempotent, no admin) -> `scripts/windows-defaults.ps1`; Pester "is idempotent: second run applies nothing" passes locally on a non-admin box (4/4, Pester 5.7.1, 2026-06-10)
- [x] AC2 (`-WithDefaults` switch invokes the deployed script) -> bats "setup-windows.ps1 declares the -WithDefaults switch" + "gates the invocation on the flag"; BUG-005 re-exec forwards the flag explicitly (the re-exec comment had warned `@args` stops sufficing the day a named param lands -- WIN-005 is that day, and a bats test now locks the forwarding)
- [x] AC3 (flag OFF by default) -> `[switch]` semantics + features.json f3 (no `$WithDefaults = $true` anywhere); opt-in hint logged when absent
- [x] AC4 (HKCU-only) -> structural bats "never references HKLM" + runtime guard (`$Root -notmatch '^HKCU:'` throws); Pester "rejects a root outside HKCU"
- [x] AC5 (tests verify invariant + idempotency) -> `tests/windows-defaults.bats` (20 tests) + `tests/windows-defaults.Tests.ps1` (4 tests, sandboxed under `HKCU:\Software\dotfiles-win005-test-<PID>`)
- [x] AC6 (README documents flag) -> Quick Start "Windows (PowerShell)" section, bats "README documents -WithDefaults"

## Test status

- `bats tests/windows-defaults.bats` -> 20/20 (includes PSScriptAnalyzer strict-catch variant + ParseFile via `tests/winpath.bash`)
- `Invoke-Pester tests/windows-defaults.Tests.ps1` -> 4/4 on this box (Windows 11, non-admin); all writes sandboxed, real HKCU untouched
- Full Windows bats subset (the 7 CI files + this one) re-run locally: no regressions
- Live `windows-latest` confirmation: PR #331 fully green on the FIRST run (run 27312926683; `test-windows` 5m56s) -- the Pester suite applied all defaults via real registry writes in the runner sandbox and confirmed `Applied 0` on the second pass. Zero CI iterations needed (WIN-004 lessons applied at authoring time)

## Decisions made during implementation

- `-Root` parameter as a test seam: the Pester suite redirects the whole tree under a throwaway sandbox key, so idempotency is verified with REAL registry writes without mutating the runner's (or a dev box's) actual defaults. The seam is itself constrained by the HKCU-only runtime guard.
- Win10/Win11 detection via `[System.Environment]::OSVersion.Version.Build` (>= 22000), NOT a registry read of the machine hive -- keeps the "no HKLM token anywhere" structural invariant honest.
- WIN-004 lessons applied at birth: PSScriptAnalyzer bats test uses the strict `catch { exit 1 }` form (never `exit 0`); Pester `-Skip` computed at discovery time; all pwsh-bound paths go through `_winpath`.
- Defaults list mirrors the proposal categories verbatim (15 entries); nothing added beyond it -- the list is user-taste territory and the proposal is the owner's decision record.

## Promotion candidates

Before archiving, flag what (if anything) should be promoted to the vault. If all three are "no", archive in repo is the only persistence.

- [ ] Lesson for repo `docs/lessons.md`? no -- mechanics already covered by the WIN-004 lesson (2026-06-10)
- [ ] ADR-worthy decision? no
- [ ] New pattern candidate for `00_meta/patterns/`? no (single-repo concern)

## Archive checklist

- [x] `proposal.md` frontmatter set to `status: archived`
- [x] Folder moved: `specs/WIN-005-windows-defaults-script/` -> `specs/archive/WIN-005-windows-defaults-script/`
- [x] GitHub issue #129 closed (built-in workflow moves it to Done on the bitácora board)
- [x] Promotions above executed (none flagged)
