---
id: "lesson-552-a-hosted-runner-ships-the-runtime-a-clean-windows-lacks"
type: lesson
status: active
title: "A hosted runner ships the runtime a clean Windows lacks"
created: "2026-11-05"
---

# A hosted runner ships the runtime a clean Windows lacks


**Context:** #3174 added mise to the Windows tool catalog. Earlier measurements had mise 2026.9.13 installing every marked pin on Windows, and the PR's `test-windows` job installed mise on windows-latest and ran its sync over all 40 marked pins.

**Problem:** On a clean Windows 11 Arm64 guest, the same catalog entry failed with `mise-v2026.9.13-windows-arm64.exe does not run`. The binary was the right architecture (PE machine `0xAA64`) and passed its checksum. It exited `0xC0000135`, STATUS_DLL_NOT_FOUND, because it imports `vcruntime140.dll` and a clean Windows ships no Visual C++ redistributable. The GitHub-hosted Windows image installs that redistributable, so every runner measurement passed. After `winget install Microsoft.VCRedist.2015+.arm64`, `mise --version` printed `2026.9.13 windows-arm64`.

**Solution:** Prove a Windows binary from the catalog on the `dotf vm windows` clean guest, not only on a hosted runner. A binary that links the MSVC runtime dynamically needs the redistributable installed before it, from the catalog, for the box's architecture. A nonzero exit with no output is a load failure: read the code before suspecting the asset. `0xC0000135` names a missing DLL, and the strings `*.dll` in the PE name the candidates.

**Why:** A hosted runner image is a developer box with hundreds of packages installed. The things it ships are exactly the dependencies a test on it can never find missing. A from-zero claim needs a from-zero machine. The runner tests our code; it does not test the box's prerequisites.
