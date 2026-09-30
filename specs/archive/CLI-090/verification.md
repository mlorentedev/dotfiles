---
tags: [spec, verification, templates]
created: "2026-09-28"
---

# Verification - CLI-090

## Evidence

Map every acceptance criterion from `proposal.md` to concrete proof (commit hash, test name, or observed behavior).

- [x] Criterion 1 -> `tests/install-dotf.bats`: `a raw installer stream installs an explicit release outside a checkout`
- [x] Criterion 2 -> `tests/install-dotf.bats`: latest-release resolver tests; `tests/install-dotf-ps1.Tests.ps1`: `Get-DotfVersion` fallback test, and (added at archive time) the malformed-tag refusal
- [x] Criterion 3 -> existing checksum/atomic-swap regression cases in both installer test suites
- [x] Criterion 4 -> `README.md` documents native recovery commands and `dotf version` assertion

## Test status

- Test suite: WSL `bats tests/install-dotf.bats tests/install-bootstrap.bats tests/install-dotf-ps1.bats` -> 38 passed, 1 ShellCheck skip because it is not installed in WSL.
- Test suite: `Invoke-Pester -Path .\tests\install-dotf-ps1.Tests.ps1` -> 9 passed, 0 failed.
- Static validation: `bash -n scripts/install-dotf.sh`, `Invoke-ScriptAnalyzer -Path .\scripts\install-dotf.ps1 -Severity Error`, and `git diff --check` -> passed.
- Manual smoke test: raw local POSIX installer stream into a temporary WSL `$HOME` downloaded, checksum-verified, and ran `dotf version 0.60.0`.
- No regressions in exercised installer suites: yes.

## Review dispositions (round 1, PASS-WITH-GAPS)

| # | Finding | Disposition |
|---|---|---|
| 1 | PowerShell run-guard may exit 0 on a failed install (THEORETICAL Major) | Ticketed, #1890 |
| 2 | No end-to-end PowerShell recovery test (THEORETICAL Major) | Ticketed, #1890 |
| 3 | zsh drops the `$0` fallback, so `_DOTF_SCRIPT_DIR` is empty (REAL Minor) | Ticketed, #1890. The executed-under-zsh path was already a no-op, so nothing breaks end to end |
| 4 | f4's `dotf version` check is not platform-scoped (REAL Minor, contract set) | Ticketed, #1890. The contract is closed by the verdict |
| 5 | The recovery one-liner lacks the "verify before piping" note | Applied: `README.md` now carries it under the recovery command |
| 6 | Malformed-metadata error wrapped inside the lookup-failure error (THEORETICAL Minor) | Ticketed, #1890 |
| 7 | `sed`-based `tag_name` parsing (THEORETICAL Minor) | Ticketed, #1890 |

**Scope of the ticked boxes.** AC3 and features.json f4 are ticked against what is tested:
- AC3 holds for the POSIX installer. The PowerShell half, a non-zero process exit on failure, is unproven and possibly false (finding 1).
- f4 does not tell the POSIX `dotf version` line from the Windows one (finding 4).

Both boxes stay as they are because the review's contract digest pins `proposal.md`, `tasks.md` and `features.json`. Narrowing AC3 or scoping f4 now would invalidate the review that permitted this archive. Both corrections are items on #1890. PR-Agent raised the same two points on #1883.

The Windows run cited above tested `ffa74ae`. It also passed on the archive head `e4b7b1c` (`test-windows`); the Pester file is identical at both.

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

## Archive pass (2026-09-30)

Implementation merged in #1805 (`66a90b75`). Re-verified on `main` at archive time:

```
$ bats tests/install-dotf.bats tests/install-dotf-ps1.bats
1..30   all ok, exit 0
$ shellcheck scripts/install-dotf.sh
exit 0
```

- **The PowerShell half of AC2 had no test for malformed metadata.** The code
  throws on a non-semver tag, but only the POSIX side proved it. This pass adds
  `refuses release metadata whose tag is not a semver version` to
  `tests/install-dotf-ps1.Tests.ps1`. pwsh is not installed on this Linux box,
  so the case runs in CI's Windows Pester job. It passed there on #1883:
  `test-windows` (run 36738902605) reports `install-dotf-ps1.Tests.ps1 (10 tests)`
  and `Tests Passed: 87, Failed: 0`. The file had 9 cases before this pass.
- **`features.json` f4 grepped for the words "Recovery", "bootstrap" and
  "install-dotf" anywhere in the README.** Almost any README matches that. It now
  checks the two one-line recovery commands, the `dotf version` assertion and
  the checkout-bootstrap distinction. f5 covers the PowerShell resolver.
- The lint box was open because ShellCheck was missing in WSL. It passes on Linux.

## Archive checklist

- [ ] `proposal.md` frontmatter set to `status: archived`
- [ ] Folder moved: `specs/CLI-090/` -> `specs/archive/CLI-090/`
- [ ] Bitácora board ticket for this spec moved to Done / closed with PR link (ADR-018)
- [ ] Promotions above executed (if any)
