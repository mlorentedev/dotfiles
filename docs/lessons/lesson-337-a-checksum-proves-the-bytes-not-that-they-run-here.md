---
id: "lesson-337-a-checksum-proves-the-bytes-not-that-they-run-here"
type: lesson
status: active
title: "A checksum proves the bytes, not that they run here"
created: "2026-10-05"
---

# A checksum proves the bytes, not that they run here

## Context
The first bootstrap audit on a macOS arm64 machine (PLAT-001, #2013) found that `setup-linux.sh`
downloads age, eza, jq, gh and shellcheck from release URLs with `linux-amd64` hard-coded. The
catalog installer (`dotf tools install`) had the same weakness in a quieter form. It verified
every asset's sha256 against the release's checksums file and then placed it. The catalog entry
picks the asset per OS, and a wrong template is a one-line typo.

## The Trap
Both paths logged success for a binary that could not execute. The checksum gate passed, because
the bytes really were the release's: the release's *Linux* build. The binary landed in
`~/.local/bin`, which is first on PATH, so it shadowed a working copy installed some other way.
`command -v` was then satisfied, so no later run repaired it. The failure only appeared on first
use (`exec format error`), far from the install that caused it. With eza, `alias ls=eza` broke
`ls` in every shell. Measured on darwin/arm64: released `dotf` 0.64.0 installed
`sops-v3.13.1.linux.arm64` and exited 0.

npm and uv have the same shape. They exit 0 when their global bin dir is not on PATH, or when an
older copy earlier on PATH still answers, and the installer read that as "installed".

## The Solution
Treat "it executes and reports at least the pinned version" as the post-condition of an install,
and check it before the result can do harm. `dotf tools install` now stages a release binary under
its command name in the temp dir and runs `--version` after the checksum gate. Only a binary that
reports a version at or above the pin is placed. After `npm install -g` or `uv tool install`, it
probes the tool on PATH and fails, naming the tool, when nothing runs or an older copy answers.

The general rule is the one in `.claude/CLAUDE.md`: a step whose failure path reports success needs
a post-condition. Integrity checks (checksum, signature) answer "are these the right bytes"; only
running the thing answers "does it work on this OS and arch".
