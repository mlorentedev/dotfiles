---
id: "WIN-015-doctor-batch-launcher"
type: spec
status: implementing # draft | implementing | verifying | archived
created: "2026-11-17"
issue: "mlorentedev/dotfiles#3572"   # repo#NNN — GitHub issue / Project item that tracks this spec
tags: [spec, proposal]
template_version: "1.0"
owner: manu
wip_override: "small doctor check split from the pi native launch fix, one PR (15 active, limit 10, 2026-11-17)"
---

# WIN-015-doctor-batch-launcher

## Why

<!-- from issue #3572: WIN-015: doctor reports an agent whose only Windows launcher is a batch file -->

On Windows `dotf` refuses to start an agent whose launcher is a batch file, because `cmd.exe` would re-parse the prompt (#3429, #3440). Since #3570 a launch starts the native executable mise holds behind such a shim and is refused only when there is none. Which of the two cases a machine is in is learned at the first launch, when a lane or a `dotf spec review` fails to start. #3569 asked for `dotf doctor` to say it ahead of time; that part was left out of #3570.

## What

`dotf doctor` gains a section, on Windows, with one line per agent of the lane launch table: the file a launch starts, or a warning that the only launcher is a batch file dotf refuses.

## Out of scope

- Any change to what is refused or to how a native executable is found: the check asks `lane.NativeTarget`, the function the launch asks.
- Installing a native build, or `mise-shim.exe` and PATH order: PLAT-001c (#2013).
- A `--fix`. There is nothing doctor can repair: a native build exists or it does not.
- Agents outside the launch tables.

## Risks / open questions

- The check asks mise once per agent whose launcher is a batch file (`mise which`, bounded at 15 s each by `lane.MiseNative`). Measured on this machine: one question, for pi. An agent with a native launcher on PATH costs nothing.
- A warning on every run for an agent that has no native build (opencode today). Accepted: it is true, it names why a lane will not start it, and it is a warning, so doctor's exit status does not change.
- The review launcher's runners are not read from `internal/spec`: its table is a `switch`, not a list. They are pi and agy, both in the lane table. A runner added there and not to the lane table would not be reported.

## Acceptance criteria

Observable outcomes. Each must be testable.

- [x] AC1: an agent PATH resolves to a native executable is reported OK, naming the file.
- [x] AC2: an agent PATH resolves to a batch file, with a native `.exe` behind it, is reported OK naming both files.
- [x] AC3: an agent whose only launcher is a batch file is a warning that names the file and `mise use -g <agent>`; what the tool manager answers is never named as a target unless `lane.NativeTarget` accepts it.
- [x] AC4: an agent that is not installed is skipped, and every agent of the lane launch table has a line.
- [x] AC5: off Windows the section is one skipped line, and neither PATH nor the tool manager is asked.
- [x] AC6: the check starts no process, and a System without the tool-manager seam never reaches the machine's mise.

## References

- Bitacora board: the GitHub issue / Project item tracking this spec (see the `issue:` frontmatter field)
- `specs/BUG-174-pi-native-launch-windows/`: the rule this reports
- `docs/lessons/lesson-611-a-tool-managers-path-for-a-command-is-not-a-file-windows-will-start.md`
