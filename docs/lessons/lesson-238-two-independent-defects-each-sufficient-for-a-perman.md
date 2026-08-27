# Lesson 238 — Two independent defects, each sufficient for a permanent red, hide behind one symptom

**Date:** 2026-08-27
**Context:** WIN-008 (#1289) — `[FAIL] stale: .copilot/copilot-instructions.md has drifted` on every Windows setup, with a remedy (`compile-harness.sh --deploy`) that does not exist on Windows.
**Category:** guards, line endings, windows, diagnosis

## What happened

The deployed file measured 5708 bytes with 67 CR / 67 LF; the repo source 1902
bytes with 0 CR. After stripping the harness regions the two differed only by
the carriage returns. So the cause looked like one thing: a writer emitting CRLF.

It was two things, and either alone guaranteed the FAIL:

1. **Writer.** `Set-Content -Value $list -Encoding UTF8` joins a list with the
   platform newline and rewrote the *whole* file CRLF, not just the injected
   region. The same call shape wrote every rendered `SKILL.md` and command file.
2. **Comparator.** `stripHarnessRegions` split on `"\n"` and closed a skipped
   region with `l == endMarker`. A `\r`-suffixed END marker never matched, so
   the strip swallowed the rest of the file — the comparison was not "off by 67
   carriage returns", it was comparing a truncated document. And independently,
   every remaining line kept its trailing `\r`.

Fixing the writer alone would have cleared the FAIL and left the comparator
one stray `\r` away from the same runaway skip. Fixing the comparator alone
would have cleared it and left every deployed `.md` violating the
`.gitattributes` contract the repo declares (`*.md eol=lf`).

## The lesson

When a guard is permanently red with a remedy that cannot clear it, keep
looking after the first cause. A symptom that survives a correct fix is the
second defect announcing itself — cheaper to find now, with the measurement
already in hand, than after the first fix ships and the FAIL "comes back".

Normalise what is not content at the comparator (line endings), and fix the
writer to honour the declared contract; guard both halves separately, because
they fail separately.

## Guard

Go: `TestStripHarnessRegions/CRLF_input…` and
`TestCheckInstructionDrift_CRLFDeployedCopyIsNotDrift` (mutation-checked: both
fail without the normalisation). PowerShell: `Write-Utf8LfFile` pinned by
`tests/console-encoding.Tests.ps1`, and the copilot native-skills smoke — which
was invoked nowhere before — now asserts 0 CR bytes and no BOM on the deployed
copy after a real `Deploy-SkillRecord` run.
