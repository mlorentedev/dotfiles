# Lesson 223 — A test updated to keep passing stops being a guard

**Date:** 2026-08-23
**Area:** guards / deploy / harness
**Severity:** medium — a permanent red check whose printed remedy cannot clear it

## What happened

`dotf doctor` reported, persistently:

```text
[Harness + skill drift]
  [FAIL] harness/skill drift (run: compile-harness.sh --refresh, then re-deploy)
```

Running the printed remedy from the repo exits **0** — `[check] OK: no harness drift`.
So the operator follows the instruction, sees green, re-runs the doctor, and is
still red, with nothing in the message pointing at the cause.

The cause needed three facts, none of which is visible from the failure text:

1. `harness/manifest.json` declares three injection targets: `AGENTS.md`,
   `ai/claude/CLAUDE.md`, and — added by `#1176` — `ai/orca/ORCA.md`.
2. `setup-linux.sh` mirrored a **hardcoded pair** into `~/.dotfiles`. The third
   target never got a copy line, so the deploy dir never had the file.
3. `checkCompileHarnessDrift` (`cli/internal/doctor/checks_deploy.go:611`) runs
   the script **from `$DOTFILES_DIR`**, not from the repo. The two scripts are
   byte-identical; only the root differs. The mirror's run evaluated a target
   whose file was absent and failed on the marker count.

## The part worth carrying forward

There was already a guard for exactly this. `tests/compile-harness-rootresolve.bats`
assembles a non-git copy of "exactly what `--check` reads" and asserts it passes.
It was written *because* this failure mode had happened before — its header says
so: *"setup copied none of the latter three, so `--check` exited 2 and section 12
reported a false drift FAIL."*

And `#1176` **edited that test** to add `ai/orca/ORCA.md` to its hand-copied list.
The commit message records it plainly: *"include orca in rootresolve test"*.

So the test went on passing, by doing by hand precisely the copy that the deploy
did not do. The guard was adjusted to accommodate the gap instead of exposing it.
Nothing was hidden and nothing was careless — the test needed that file to run at
all, and adding it is the obvious local fix. That is what makes the shape worth
naming: **the change that keeps a guard green is not always the change that keeps
it a guard.**

The structural cause underneath: the target list existed in **four** places — the
manifest, `setup-linux.sh`'s copy block, `compile-harness-rootresolve.bats`'s
assembly block, and `setup-linux.bats`, which asserted each literal `safe_copy`
line by `grep -qF`. The fourth only surfaced when the fix removed those lines and
CI went red on it, which is its own small lesson: **a test that asserts the
presence of a hardcoded list can only check that the entries someone remembered
are there, never that the list is complete.** It was green throughout, on a list
missing the entry that mattered.

A list restated four times can only diverge, and it diverged silently in the copy
that had no assertion attached.

## Rule

- When a change makes an existing test need new setup, ask what the test was
  protecting **before** adding the setup. If the new fixture line is doing work
  the production path is supposed to do, the test is now describing a bug rather
  than catching one.
- **Derive lists, never restate them.** The fix reads targets from the manifest in
  both the deploy block and the test, so the next entry is mirrored by
  construction. There is no place left for the list to disagree with itself.
- A guard that builds its own copy of the world can only test logic, never
  delivery. Assert the outcome where it actually lands too — here,
  `verify-setup.bats` now asserts every manifest target exists in the real
  `$DOTFILES_DIR` after a real setup run, which is the half no unit test reaches.
- **When a check fails, verify the remedy it prints actually clears it.** A red
  check with an unactionable message trains the operator to ignore the check,
  which costs more than the original defect. "Target absent" and "target present
  but disagrees" need different messages; collapsing them is what produced this.

## What the new guard found on its first run

Worth recording, because it is the argument for writing the outcome assertion at
all. The integration guard went red on its very first CI run — not on the bug it
was written for, but on a latent one underneath it. In one setup run, 22 seconds
apart and in the same process:

```text
[SUCCESS] jq installed
[WARNING] Claude Code CLI, npx, or jq not found, skipping MCP server registration
```

Both lines are true. `jq` is downloaded to `$HOME/.local/bin`, which the rc files
put on `PATH` and the running setup process does not have. So `command -v jq`
is false immediately after a successful install, and every step gated on it —
MCP server registration among them — is silently skipped on precisely the fresh
machine that needs it. The success message reports the *download*; the later
check asks about the *lookup*. **"Installed" has to mean the same thing the next
check asks about, or the two will disagree and both will look right.** Tracked as
`#1202`; worked around here by resolving the binary path explicitly.

## Evidence

```console
$ bash ~/.dotfiles/scripts/compile-harness.sh --check
[ERROR] /home/manu/.dotfiles/ai/orca/ORCA.md: need exactly 1 BEGIN + 1 END HARNESS marker (found /)
[check] FAIL: harness drift detected

$ ./scripts/compile-harness.sh --check
[check] OK: no harness drift

$ diff scripts/compile-harness.sh ~/.dotfiles/scripts/compile-harness.sh   # identical

# after the fix, mirroring derived from the manifest:
$ dotf doctor
[Harness + skill drift]
  (5 checks, all ok)
Results: 147 passed, 0 failed, 4 warned, 6 skipped
```

Tracked as `#1200`.
