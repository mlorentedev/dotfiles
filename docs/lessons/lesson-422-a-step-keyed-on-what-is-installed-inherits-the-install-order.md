---
id: "lesson-422-a-step-keyed-on-what-is-installed-inherits-the-install-order"
type: lesson
status: active
title: "A step keyed on what is installed inherits the install order"
created: "2026-10-14"
---

# A step keyed on what is installed inherits the install order

## Context
`agent-extensions` gained a herdr half that installs herdr's integration for every harness on PATH (#2449). Converge runs its native steps in a fixed order, and `legacy-setup` (the remaining `setup-linux.sh`) runs last.

## Problem
from-zero on Ubuntu was red: the first converge installed 4 herdr integrations, and the second installed a fifth. pi was installed by `setup-linux.sh`, so on a fresh box it appeared after `agent-extensions` had already looked for it. On macOS the job stayed green only because nothing installed pi at all. Every unit test passed, because each test fixed which harnesses were present.

## Solution
pi moved to the step that installs harnesses: `versions.conf` marks `PI_VERSION` under `# mise: latest`, so the `tools` step installs it before anything reads it, on Linux and macOS alike. `setup-linux.sh` lost its pi install, and a guard rejects an npm or vendor install of an agent there. The registry was not reordered: its rule is that the script runs last and each ported block becomes a native step before it. Moving `agent-extensions` after the script would have hidden the next harness someone leaves there.

## Why
A step that asks "what is installed?" depends on every step that installs, including ones outside its own code. The second-converge check is the test that sees that dependency, because only a fresh machine starts with nothing installed. When it fails on one OS and passes on another, ask what each OS installs and when.
