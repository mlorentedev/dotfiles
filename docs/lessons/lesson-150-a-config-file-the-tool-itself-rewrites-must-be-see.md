---
id: lesson-150-a-config-file-the-tool-itself-rewrites-must-be-see
type: lesson
status: active
created: "2026-08-05"
owner: manu
tags: [lesson, dotfiles]
---

# Lesson 150: A config file the tool itself rewrites must be seeded, not synced

**Context**: `setup-{linux,windows}` deploy three pi files side by side. `models.json` and `tui.json` are dotfiles-owned: the deploy copies them whenever source and destination differ, which is correct — the repo is the source of truth and any local edit is drift to be corrected. `ai/pi/settings.json` was given the same shape, and `ai/pi/README.md` plus `tests/pi-config.bats` have described it as seed-if-missing since AI-025.

**Problem**: pi rewrites `settings.json` at runtime — `lastChangelogVersion`, `theme`, and the model picked in the TUI — and `tests/pi-config.bats` *forbids* the committed copy from carrying `lastChangelogVersion` at all. The two files therefore can never be byte-identical once pi has run: `cmp -s` could only ever fail, the `already in sync` branch was unreachable, and every setup run silently reset the user's theme and default model. The bug had shipped on both platforms since AI-025 and surfaced only because a doc edit prompted someone to read what the code beside it actually did. Nothing was red: the integration container seeds a fresh `HOME`, so it exercises the first run and never the second, which is where the whole bug lives.

**Solution**: Guard the copy on the destination being absent, on both platforms (#756). Add source-level assertions that the shape cannot drift back, since the container cannot observe run two, and state that reason in the test so the next reader does not "improve" it into a behavioral test that silently covers nothing.

**Rule**: Before choosing a deploy policy, ask who writes the file at runtime. A file only dotfiles writes is *synced* (copy when different); a file the installed tool also writes is *seeded* (copy when absent) and every later run must leave it alone. A byte-comparison against a self-mutating destination is not a weak check, it is dead code — if the file is guaranteed to differ, the comparison has no true branch. And when a test cannot cover a code path (here, the second run of a bootstrap script), say why in the test body: an unexplained source-level assertion is the first thing a future reader deletes.
