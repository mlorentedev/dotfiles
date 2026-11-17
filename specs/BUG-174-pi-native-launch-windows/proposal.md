---
id: "BUG-174-pi-native-launch-windows"
type: spec
status: implementing # draft | implementing | verifying | archived
created: "2026-11-16"
issue: "mlorentedev/dotfiles#3569"   # repo#NNN — GitHub issue / Project item that tracks this spec
tags: [spec, proposal]
template_version: "1.0"
owner: manu
wip_override: "five of the seven adversarial reviewers cannot be launched on Windows until this lands, so the finished specs waiting for review cannot be archived from this machine (13 active, limit 10, 2026-11-16)"
---

# BUG-174-pi-native-launch-windows

> **Naming**: file lives at `<repo>/specs/BUG-174-pi-native-launch-windows/proposal.md`. `BUG-174-pi-native-launch-windows` is `AREA-NNN-slug` (e.g. `TOOL-001-secret-drift`).

## Why

<!-- from issue #3569: BUG-174: pi cannot be launched by dotf on Windows, only batch shims are on PATH -->

On Windows `dotf` refuses to start an agent whose launcher is a `.cmd` or `.bat` file, because `cmd.exe` would re-parse the free-text prompt (#3442, #3506). The refusal asks for a native executable and nothing supplies one: the first `pi` on PATH is a batch shim (npm's, or mise's own when `mise-shim.exe` is not beside mise), while mise already holds a native `pi.exe`. Since 2026-11-12 no pi-runner reviewer can be launched from a Windows machine, which is five of the seven members of `harness/reviewer-pool.json`.

## What

Where Windows would run a launch target through `cmd.exe`, `dotf lane exec` and `dotf secrets run` ask mise for the executable of that name (`mise which`) and start it when it is a native `.exe`. Without one the launch is refused with the same error as before. The check that allows a launch and the launch itself use one resolution.

## Out of scope

- Installing mise with `mise-shim.exe` so its shims are native: PLAT-001c (#2013). It would not fix this alone, since `~/.pi/agent/bin/pi.cmd` is ahead of mise's shims on PATH.
- Retiring the npm pi block in `setup-windows.ps1` and reordering PATH: PLAT-001c AC4 follow-up.
- Agents mise holds only as npm packages (opencode): mise has no `.exe` for them, so they stay refused.
- Any change to what counts as a batch shim, or to the refusal's wording.
- `dotf doctor` reporting a batch-only agent launcher before the first launch: #3569 asks for it, and it is its own check with its own tests, so it has its own ticket (#3572).

## Risks / open questions

- mise's answer is input, not a fact. Resolved: only a bare command name is asked about, with `--` before it; the answer must be an absolute path to a regular `.exe`; mise itself must be a native executable; the question is bounded at 15 s.
- `mise which` reads the configuration of the working directory, so a trusted project `mise.toml` can pin another pi. That is what a mise shim resolves there too, so the launch starts the tool the shell would.
- The pi mise installs (1.5.2) is not the copy the shell starts (`~/.pi/agent/bin`, npm 1.5.0). Measured 2026-11-16: both read `~/.pi/agent` and list the same extension packages and model patterns.
- `dotf spec review` starts `dotf` from PATH, so the fix reaches reviews only once a release carries it.

## Acceptance criteria

Observable outcomes. Each must be testable.

- [ ] AC1: a launch whose target is a batch shim starts the native `.exe` mise holds for that command name; the shim never runs and the arguments arrive byte for byte.
- [ ] AC2: with no native `.exe` behind the shim (unknown to mise, another batch file, an extensionless npm stub, a missing file, a directory, a relative path) the launch is refused with the batch-shim error.
- [ ] AC3: a target that is already native is started as PATH resolves it, and mise is not asked.
- [ ] AC4: a caller that names a file (a path, or a name with an extension) is never given another file in its place.
- [ ] AC5: `dotf secrets run` starts the file its check allowed, and a command a multi-call binary would start (`busybox <cmd>`) keeps the plain refusal.

## References

- Bitácora board: the GitHub issue / Project item tracking this spec (see the `issue:` frontmatter field)
- #3440, #3442, #3506: the refusal this builds on
- `specs/PLAT-001c-tools-via-mise/tasks.md`: "has no `mise-shim.exe` beside it, so mise falls back to file shims"
- `docs/lessons/lesson-611-a-tool-managers-path-for-a-command-is-not-a-file-windows-will-start.md`
