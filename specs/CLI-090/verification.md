---
tags: [spec, verification, templates]
created: "2026-09-28"
---

# Verification - CLI-090

## Evidence

Map every acceptance criterion from `proposal.md` to concrete proof (commit hash, test name, or observed behavior).

- [x] Criterion 1 -> `tests/install-dotf.bats`: `a raw installer stream installs an explicit release outside a checkout`
- [x] Criterion 2 -> `tests/install-dotf.bats`: latest-release resolver tests; `tests/install-dotf-ps1.Tests.ps1`: `Get-DotfVersion` fallback test
- [x] Criterion 3 -> existing checksum/atomic-swap regression cases in both installer test suites
- [x] Criterion 4 -> `README.md` documents native recovery commands and `dotf version` assertion

## Test status

- Test suite: WSL `bats tests/install-dotf.bats tests/install-bootstrap.bats tests/install-dotf-ps1.bats` -> 38 passed, 1 ShellCheck skip because it is not installed in WSL.
- Test suite: `Invoke-Pester -Path .\tests\install-dotf-ps1.Tests.ps1` -> 9 passed, 0 failed.
- Static validation: `bash -n scripts/install-dotf.sh`, `Invoke-ScriptAnalyzer -Path .\scripts\install-dotf.ps1 -Severity Error`, and `git diff --check` -> passed.
- Manual smoke test: raw local POSIX installer stream into a temporary WSL `$HOME` downloaded, checksum-verified, and ran `dotf version 0.60.0`.
- No regressions in exercised installer suites: yes.

## Decisions made during implementation

Brief log of non-obvious trade-offs or course corrections taken during the work. Routine choices belong in commit messages, not here.

- Release recovery is one OS-agnostic contract, not one cross-parser script:
  POSIX uses Bash and Windows uses PowerShell. Both resolve the latest release
  only when an explicit or checkout pin is unavailable.
- Raw recovery deliberately does not clone a checkout or run full setup. That
  preserves ADR-019's separate, fast-forward-only checkout update boundary.

## Promotion candidates

Answer each line `yes: <path>`, naming the file you promoted, or `no: <reason>`. `dotf spec archive` refuses a line left unanswered, a `no` without a reason, and a `yes` whose file does not exist; a `00_meta/` path is looked up in the vault.

- [x] Lesson for the repo's `docs/lessons/`? no: the behavior and its test fixtures document the checkout-free recovery contract directly.
- [x] ADR-worthy decision for the repo's `docs/adr/adr-XXX.md`? no: ADR-019 and ADR-036 already establish the update and release-channel boundaries; this change implements them.
- [x] New pattern candidate for `00_meta/patterns/`? no: this is repository-specific installer behavior, not a cross-project mechanism.

## Archive checklist

- [ ] `proposal.md` frontmatter set to `status: archived`
- [ ] Folder moved: `specs/CLI-090/` -> `specs/archive/CLI-090/`
- [ ] Bitácora board ticket for this spec moved to Done / closed with PR link (ADR-018)
- [ ] Promotions above executed (if any)
