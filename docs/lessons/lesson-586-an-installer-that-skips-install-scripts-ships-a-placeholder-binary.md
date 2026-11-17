---
id: "lesson-586-an-installer-that-skips-install-scripts-ships-a-placeholder-binary"
type: lesson
status: active
title: "An installer that skips install scripts can ship a package whose binary is a placeholder"
created: "2026-11-12"
---

# An installer that skips install scripts can ship a package whose binary is a placeholder

## Context

#3329, PLAT-001c AC4. The four npm CLIs (bw, copilot, opencode, yarn) moved to mise's `npm:` backend. A from-zero converge on a clean Windows 11 Arm64 guest installed three of them and reported opencode as not running at its pin.

## Problem

mise runs `npm install -g <pkg> --prefix <dir> --ignore-scripts=true`. `opencode-ai` declares `bin: bin/opencode.exe`, a 479-byte placeholder that its `postinstall` (`node ./postinstall.mjs`) replaces with the platform binary, shipped in the optional dependency `opencode-windows-arm64` (143 MB). With the script skipped, the install succeeds, mise lists the tool, and the launcher points at a file Windows refuses to run ("not a valid application for this OS platform"). The first hypothesis, that the package had no windows-arm64 build, was wrong: the platform package exists at the pin, and running its binary directly answers `1.16.2`.

An install that exits 0 says the package unpacked, not that it runs. The pin check caught it only because it runs the tool and reads its version (lesson 337).

## Solution

Name the package that may run its own script, not the whole tree: `opencode = { version = "...", allow_builds = ["opencode-ai"] }` in the generated mise config, from `npmBuildScripts` in `cli/internal/tools/mise.go`. `npm_args = "--ignore-scripts=false"` also works and opens every dependency's scripts; it was measured and rejected. Both were measured on the guest with mise 2026.9.13 and Node 24.22.1 (npm 11.19.1): the placeholder becomes 143,426,440 bytes and `opencode --version` answers `1.16.2`, with the option on the aliased short name.

A package that needs a script to be usable is a property of the package, found by reading its `scripts` (`npm view <pkg> scripts bin`) before moving it to a backend that skips them, not by running it on a guest.
