---
id: lesson-178-the-sandbox-is-not-the-territory-a-dry-run-that-ca
type: lesson
status: active
created: "2026-08-09"
owner: manu
tags: [lesson, dotfiles]
---

# Lesson 178: The sandbox is not the territory: a dry run that can reach production, and a fixture that cannot know what production contains

**Context**: Migrating 17 auto-memory `MEMORY.md` files (#864). Two safety steps were built before touching live data: a dry run into a sandbox copy of `~/.claude/projects`, and a fixture suite covering every shape the corpus was known to hold.

**Problem**: Both leaked, in opposite directions. **The dry run could write to production.** The sandbox was built with `cp -a`, which *preserves* symlinks — and every `projects/*/memory` is a symlink into the knowledge vault, so `--fix` against the "sandbox" would have written straight through to the real files. It did not, purely because an unrelated `head` in the pipeline closed stdout, SIGPIPE killed the script under `pipefail`, and it died before the fix step. The isolation was never tested; luck substituted for it. **The fixtures could not see a defect they had no way to contain.** The live run then emitted `shape not recognised` for a file the vault showed as successfully migrated: two project keys, `youtube-toolkit` and `yt-metrics-cli`, symlink to *one* vault directory — a rename whose old key was never removed. The scan listed the file twice, migrated it through the first alias, re-read it through the second as already-plain, and reported that as a shape failure. No fixture could have caught it, because a fixture only contains the aliases you already know about.

**Solution**: For the dry run, `cp -aL` to dereference, plus an assertion that *no symlink survives into the sandbox* — isolation proven, not assumed. For the alias, deduplicate by `filepath.EvalSymlinks` before scanning, and distinguish `ErrNotWrapped` during the fix loop (report "already plain" rather than "unrecognised shape"). No mutation probe was written for the alias case: the production run **is** the evidence that the assertion fails without the fix.

**Rule**: A dry run's isolation is a claim that needs its own assertion — if a sandbox can be reached by a symlink, a mount, or a config default, then it is production with extra steps, and the run that "did no harm" may only have crashed early. Test the isolation, not the migration. And keep the pairing straight: fixtures prove the transform on shapes you chose, production proves the *inventory* you did not. Run against reality before writing, and treat the first live run as a discovery step rather than a confirmation step — this session found two real defects that way (the alias here, and the 37-row noise wall in the adjacency check the day before), neither reachable from a fixture.
