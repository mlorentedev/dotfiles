# Lesson 250 — Restoring a file's content is not restoring the file, and the mode is the part nobody diffs

**Date:** 2026-09-01
**Context:** #1411 shipped `setup-linux.sh` at `100644`. `install.sh:52` is `exec ./setup-linux.sh "$@"` — the bootstrap entry point — so a fresh clone failed at its first step with exit 126. Every already-installed machine was unaffected, which is why it reached `main` unnoticed. Found by a peer session reading a merge diff, not by any check.
**Category:** verification, guards, mutation-testing, ci

## What happened

The bit was not lost by editing the file. It was lost by **verifying the edit**.
This repo's convention is to mutation-test a guard: break the thing, watch the
test go red, restore it. The mutation was written the obvious way:

```sh
grep -v 'has("attribution")' setup-linux.sh > /tmp/x && mv /tmp/x setup-linux.sh
```

The redirection creates a **new** file at the umask — `644` — and `mv` installs
it over the original. The executable bit is gone before the test even runs.

Then the restore, also written the obvious way:

```sh
cp /tmp/scratch/sl.bak setup-linux.sh
```

`cp` onto an **existing** file writes through it and keeps the *destination's*
mode. The destination was already `644`. So the content came back byte-for-byte
— `git diff` was empty, the content check passed, the mutation was declared
verified — and the mode did not come back at all.

That asymmetry is the whole lesson. The restore *looked* total because the only
property anyone compared was content.

## Why nothing caught it

Four layers ran and none could observe it:

- **`git diff`** shows `old mode` / `new mode` only in the header, and the
  commit was reviewed for its content.
- **pre-commit** ran the full suite: passed. No test asserted the mode.
- **CI integration** runs the script — via `tests/Dockerfile.integration:83`,
  `bash setup-linux.sh`. An explicit interpreter works at **any** mode, so the
  job exercises the script without ever exercising the invocation that broke.
- **`ci.yml`** otherwise only shellchecks it, which is mode-blind.

The contrast was inside the repo the entire time: `install.sh` was still
`100755`. Nothing compared the two.

## The rule

**A mutation that rewrites a file rewrites its metadata, and `cp` will not give
it back.** When mutating a file in place, either mutate content only —

```sh
printf '%s\n' "$mutated" > file.sh    # redirection into the EXISTING file keeps its mode
```

— or restore with `git checkout -- file.sh`, which restores the index's mode
along with its content. `cp backup file` is the form that silently loses it.

And the general form, which is this repo's recurring shape in a new place:
**"I restored it" is a claim about the property you compared, not about the
file.** Same structure as lesson 247 (a check that passes because of an accident
is indistinguishable from one that passes because of correctness) and lesson 235
(*"I cannot reproduce it" is a statement about the instrument*).

## The guard

`tests/executable-bit.bats` asserts that every script invoked as `./x.sh`
carries `100755` **in git's index** — not on the filesystem, because the index
is what a fresh clone materialises and what a commit records; a working tree can
be executable while the thing that ships is not.

Two details were deliberate:

- **The list is derived from the invokers** (`install.sh`, `README.md`, `docs/`),
  never hand-maintained. A hand-written list cannot catch the entry point nobody
  thought to add to it — the same reasoning already recorded in
  `tests/claude-settings-template.bats:36`.
- **A second test asserts the derivation is non-empty.** Verified by mutation:
  with a regex that matches nothing, the main test **passes vacuously** and only
  the meta-test fails. A guard that passes on the broken thing is the defect the
  guard exists to prevent, and a derived list makes that failure mode reachable
  in a way a literal list does not.

Blanket "every `.sh` is executable" would have been the wrong guard: of 44
tracked shell files, four are libraries that are sourced rather than executed
(`.zsh/functions.sh`, `git-hooks/lib/board-pickup.sh`, two `tests/golden/*/lib.sh`).
The narrow rule is the true one.
