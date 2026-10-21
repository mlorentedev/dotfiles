---
id: "lesson-473-a-leg-that-puts-the-binary-on-path-first-certifies-a-machine-the-installer-never-makes"
type: lesson
status: active
title: "A leg that puts the binary on PATH first certifies a machine the installer never makes"
created: "2026-10-17"
---

# A leg that puts the binary on PATH first certifies a machine the installer never makes


**Context:** `from-zero.yml` and the `integration` container both run the bootstrap from zero and both were green. PLAT-003's first live `dotf vm linux` converged `main` in a clean Ubuntu 24.04 guest that shares nothing with the host.

**Problem:** The guest's first converge failed at `records-harness` (`dotf is absent or predates it`), and 12 steps were skipped. `install.sh` execs converge by the binary's absolute path but never puts `~/.local/bin` on PATH, and `compile-harness.sh` looks `dotf` up on PATH. Neither CI leg could see this. `from-zero.yml` appends `$HOME/.local/bin` to `GITHUB_PATH` before `./install.sh`, so the PR's own build is the one used. `setup-linux.sh` exports the same directory before anything runs. Each leg had prepared, for its own good reason, the one condition the installer was supposed to establish (#2688).

**Solution:** At least one leg must run the bootstrap the way a person does, with the environment a fresh machine has and nothing pre-positioned: `dotf vm linux`, which copies in a git archive of a committed ref and runs `./install.sh` with the guest's own PATH. When a CI leg has to stage the artifact under test (to test the tree's binary rather than the release), name what that staging hides in a comment, so the gap is a known one.

**Why:** A verification leg certifies the machine it ran on, and setup steps are part of that machine. A step that makes the code under test findable also answers the question "does the installer make it findable?", and always answers yes.
