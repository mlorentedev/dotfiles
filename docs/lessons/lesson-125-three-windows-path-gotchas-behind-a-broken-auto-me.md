---
id: lesson-125-three-windows-path-gotchas-behind-a-broken-auto-me
type: lesson
status: active
created: "2026-06-25"
owner: manu
tags: [lesson, dotfiles]
---

# Lesson 125: Three Windows path gotchas behind a "broken" auto-memory junction (Go 1.26)

**Context**: HARNESS-040 (#551) wired `dotf doctor --fix` to the merged `memlink` primitive to detect+repair the Claude auto-memory↔vault junction. Implementing it on Windows surfaced three non-obvious cross-OS facts the POSIX-first shell code had silently papered over.

**Problem**: (1) **Encoding** — Claude's per-project key under `~/.claude/projects/<key>` maps *every* path separator AND the drive colon to `-` (`C:\Users\me\proj` → `C--Users-me-proj`), but the ported `encodeProjectPath` only replaced `/`. On Windows it computed the wrong key, so the junction was created at (or looked for at) the wrong path — the latent root cause of the "junction never created here". (2) **Link detection** — a `mklink /J` junction surfaces via `os.Lstat` as `ModeIrregular`, **not** `ModeSymlink`, on Go 1.26 (verified empirically). The old `isLink` checked only `ModeSymlink`, so it never recognized a junction; `Ensure`'s "already linked" no-op only worked by accidentally falling through to its `dirNotEmpty` branch. (3) **cmd quoting** — `exec.Command("cmd","/c","mklink","/J",target,src)` relies on Go's `EscapeArg`, which quotes args with **spaces** (so `C:\Users\First Last\...` works) but not a bare **comma**; cmd then splits the path on the comma and mklink fails silently. A comma slipped in via a `t.TempDir()` path derived from a subtest name containing `(PASS, no dup)`.

**Solution**: Put the encoding in the shared `memlink` primitive (`ClaudeProjectKey` maps `/ \ :` → `-`; `ClaudeMemoryTarget` joins the full path) so the session-start adapter and doctor compute an identical target on every OS, and deleted the local `mem.encodeProjectPath`. Widened `isLink` to `ModeSymlink|ModeIrregular`. Named test subtests without `,`/`()` so they don't poison `t.TempDir()` paths; ticketed the real cmd-quoting robustness fix (#575) rather than rabbit-holing on `cmd /s /c` quoting in this PR.

**Rule**: When code that creates/inspects filesystem links is "ported from shell", re-derive the Windows facts from scratch — separators, the drive colon, junction-vs-symlink mode bits, and cmd argument quoting are all places POSIX intuition is wrong. Keep one OS-aware encoding as SSOT shared by every caller; never let two callers re-encode a path independently. And keep test names free of shell/cmd metacharacters — `t.TempDir()` embeds the test name, so a comma or paren in a subtest name becomes a real path component.
