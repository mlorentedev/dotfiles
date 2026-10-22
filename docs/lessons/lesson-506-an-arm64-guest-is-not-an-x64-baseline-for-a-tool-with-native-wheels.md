---
id: "lesson-506-an-arm64-guest-is-not-an-x64-baseline-for-a-tool-with-native-wheels"
type: lesson
status: active
title: "An Arm64 guest is not an x64 baseline for a tool with native wheels"
created: "2026-10-21"
---

# An Arm64 guest is not an x64 baseline for a tool with native wheels

## Context

AI-028 was triaged against a clean Windows 11 guest (`dotf vm windows`,
PLAT-003) to measure how hive gets installed on a fresh Windows machine. The
converge ended green, `dotf doctor` reported hive missing, and the first reading
was "no fresh Windows machine gets hive", which informed the decision to resume
the spec (#791).

## Problem

The guest is Arm64. `hive-vault` needs `cryptography>=42`, and cryptography
shipped `win_arm64` wheels only in 46.0.0 to 46.0.3; 50.0.2, the latest, has
none. uv therefore built it from source and failed at `link.exe not found`.
`setup-windows.ps1` pipes that install to `Out-Null`, so the log showed only the
consequence (no hive), never the cause. On x64 the wheel exists and the same
install most likely succeeds, leaving the uv copy hive ADR-019 rules out: a
different defect. An install-model question had turned into a toolchain question,
and the converge stayed green either way.

## Solution

Print what the script discards before reading a missing tool as a design
finding: a throwaway branch that only echoed the install's output, `$env:Path`
and the expected paths found the cause in one guest run (hive#515). State the
architecture next to every guest measurement, and treat a result for a Python
tool with native dependencies as holding for that architecture only. `uv pip
compile --python-platform aarch64-pc-windows-msvc --only-binary <pkg>` shows,
without a guest, which version still has a wheel.
