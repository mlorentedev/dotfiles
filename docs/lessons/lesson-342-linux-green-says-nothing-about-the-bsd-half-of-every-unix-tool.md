---
id: "lesson-342-linux-green-says-nothing-about-the-bsd-half-of-every-unix-tool"
type: lesson
status: active
title: "Linux green says nothing about the BSD half of every Unix tool"
created: "2026-10-07"
---

# Linux green says nothing about the BSD half of every Unix tool

## Context
PLAT-001d (#2013, W10) ran the whole bats suite on a Mac for the first time. 88 of 1784 tests
failed, on a suite that had been green on Linux CI for months. Nobody had broken anything: every
one of them had been wrong on a Mac from the day it was written, and no job could say so.

## The Trap
Four of the causes did not fail loudly, which is why they lasted.

- **`grep -nP … 2>/dev/null`** in `check-bats-names.sh`. BSD grep has no `-P`, the error went to
  `/dev/null`, and "grep failed" read as "no findings". The lint that exists to catch silently
  dropped tests passed every file on a Mac, including the ones it was written about.
- **`wc` padding.** BSD `wc -l` prints `       5`. Arithmetic ignores it, so `[ "$n" -gt 3 ]`
  worked, and only the *printed* number was wrong: a vault-health report, a harness warning
  (`is 12 characters`) and 13 golden-file tests.
- **A logical path compared with a physical one.** macOS `$TMPDIR` is `/var/folders/…`, a symlink to
  `/private/var/…`. `git rev-parse --show-toplevel` and Go's `os.Getwd` answer with the physical
  path, so the memory-sink guard never recognised the vault, and the pre-commit store check never
  recognised the store: a guard that does not apply looks exactly like a guard that passed.
- **A detector that matched nothing.** `pgrep -a` means "include ancestors" on BSD and prints no
  command line, so the stray-GUI-process detector in `tests/setup_suite.bash` was inactive on a
  Mac and its own tests could only have failed if someone ran them there.

The rest were loud once looked at: `sha256sum` missing before macOS 26 (and then only in `/sbin`,
off the PATH an installer stream runs with), `chmod --reference`, `stat -c`, GNU-only `sed` syntax
in a test helper, a copied Apple-signed binary killed at exec, bash 3.2's `"${arr[@]}"` on an empty
array under `set -u`, and a Python 3.9 with no `tomllib` or PyYAML. Forty of the 88 were that last
kind, a missing tool, and each said so; the tests were right to fail.

## The Solution
Classify every failure into one of three, and do not mix them: a **script bug** (fix the script),
a **non-hermetic test** (fix the test), or **genuinely Linux-only** (skip, with the reason, on the
condition that makes it so, never on the OS name alone).

- Prefer the spelling that works on both over a per-OS branch: `[[:space:]]` for `\s`, `-E` for `-P`,
  `shasum -a 256` where `sha256sum` is absent, `ps -axo pid=,args=` for `pgrep -a`,
  `stat -c … || stat -f …`, `| tr -d ' '` after `wc`, `cd "$p" && pwd -P` before comparing paths.
- Shared test spellings live in `tests/lib/os.bash`. Where a test runs a CI step that only ever runs
  on ubuntu (`release-pr-body-refs.bats`), `require_gnu_sed` skips with that reason.
- Do not discard the error of the tool you ask a question with. `check-bats-names.sh` now treats
  grep exit 2 as a failure, and a test proves it with an unreadable file, no stub needed.
- Tag the tests whose answer depends on the OS (`# bats file_tags=os-sensitive`) and run that tier
  on a real macOS (bash 3.2) in CI; the full suite stays on Linux, and runs on macOS only on main.
  `tests/guard-bats-tags.bats` rejects a near-miss tag, because bats ignores one without a word, and
  `scripts/run-bats.sh` fails a tier that selects nothing, because bats passes an empty selection.
- bats scans every line of a test file for `@test` and `# bats …`, heredoc bodies included. A fixture
  that contains either is turned into part of the file that holds it; assemble it with `printf`.

A BSD tool is not an exotic target. It is the default on the machine the owner uses every day.
