---
id: lesson-151-a-guard-can-be-green-because-its-assertion-never-r
type: lesson
status: active
created: "2026-08-05"
owner: manu
tags: [lesson, dotfiles]
---

# Lesson 151: A guard can be green because its assertion never ran

**Context**: The fix above shipped with two source-level assertions, one per platform, each verifying that the pi settings deploy is guarded on the destination being absent and carries neither `Compare-Object` nor `-Force`. Both were green on the first run, and the Linux one was genuinely correct.

**Problem**: The Windows assertion was green for two independent wrong reasons, neither visible from a passing run. First, `setup-windows.ps1` is CRLF (`.gitattributes`), so the `sed` range ending at `/^}$/` never closed — a CR sits before every newline — and the extracted "block" was the remaining 842 lines of the file rather than the intended 10. Second, `grep -qF '-Force'` parses a leading-dash pattern as **options** (`-F -o -r -c -e`), leaving `-e` without its argument; grep exits 2, so the `&&` that reports the failure never fired. The assertion could not have failed even with `-Force` present on the exact line it was written to catch — and it was, 27 times, in the unrelated code the runaway range had swallowed.

**Solution**: Strip CR before matching, pass the pattern as `grep -qF -e '<pattern>'`, and assert the extracted block is small — that size check is what converts a range that fails to close into a red test instead of a silently vacuous one. Confirmed by planting each defect in turn: re-adding `-Force` is now red, and was **green** before the repair.

**Rule**: A new guard is not done when it passes; it is done when you have watched it fail. Plant the exact defect it exists to catch and confirm red — a guard verified only in the green direction is indistinguishable from one that asserts nothing. Two mechanical traps make this cheap to get wrong: any pattern starting with `-` needs `grep -e` (or `--`), and any line-anchored `sed`/`grep` range over a CRLF file must strip CR first or the anchor silently never matches. When a test extracts a region of a file, assert the region's size too, so a range that runs to EOF fails loudly instead of quietly widening what the test claims to cover.
