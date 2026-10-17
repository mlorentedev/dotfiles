---
id: "lesson-466-a-substring-parity-assertion-passes-for-a-prefix-of-the-value"
type: lesson
status: active
title: "A substring parity assertion passes for a prefix of the value"
created: "2026-10-17"
---

# A substring parity assertion passes for a prefix of the value


**Context:** PLAT-003 adds `dotf lint powershell`, a local run of CI's `lint-powershell` job. A Go test holds the two together by asserting that the job's script contains each parameter the command uses: `-Severity Error,Warning`, `-Settings .PSScriptAnalyzerSettings.psd1`, `-Include *.ps1, *.psm1`.

**Problem:** The first version used `strings.Contains`. A mutation that narrowed the command to `-Severity Error` still passed, because `-Severity Error` is a substring of `-Severity Error,Warning`. The same holds for any value whose narrowing is a prefix: a settings file without its extension, an `-Include` list with its last entry dropped. The test was green for exactly the drift it existed to catch: the local lint going quieter than CI.

**Solution:** Match each expected parameter followed by a token boundary (`` regexp.QuoteMeta(want) + `(\s|\)|$)` ``), and run the mutations before trusting the test: narrow each constant once and watch it fail. Parse the workflow with a YAML decoder rather than cutting it by line range, so a moved or renamed job cannot widen or empty what the test reads.

**Why:** A containment check answers "is this text somewhere in there", and a narrower setting is usually a prefix of the wider one. A guard that cannot tell the two apart passes on the regression it names. Mutate the guard once before calling it a guard.
