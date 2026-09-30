---
id: "HARNESS-168-agy-windows-no-sandbox"
type: spec
status: verifying # draft | implementing | verifying | archived
created: "2026-09-29"
issue: "mlorentedev/dotfiles#1838"   # repo#NNN — GitHub issue / Project item that tracks this spec
tags: [spec, proposal]
template_version: "1.0"
---

# HARNESS-168: Run agy reviews without elevation on Windows

## Why

<!-- from issue #1838: HARNESS-168: avoid agy sandbox elevation on Windows -->

`dotf spec review` forces `agy --sandbox` on every operating system. Antigravity
1.2.13 implements that flag with a Windows AppContainer; on a standard work
account every `run_command` triggers UAC elevation that cannot be approved, so
the independent reviewer times out without a verdict. Direct `agy` works because
the deployed settings leave terminal sandboxing disabled.

## What

On Windows, the agy review launcher omits `--sandbox` and runs under the
caller's existing non-elevated user token. It retains the pinned reviewer model,
workspace, deadline, stream output and unattended permission flag. Linux and
macOS continue to force sandbox isolation.

## Out of scope

- Elevating agy or changing Windows UAC policy.
- Removing sandbox isolation from non-Windows review runs.
- Replacing `--dangerously-skip-permissions` with a new permission policy.
- Changing NaN/pi reviewer behavior.

## Risks / open questions

- Windows loses AppContainer isolation for this headless review path. The
  remaining bounds are the standard-user token, allow-listed reviewer model,
  isolated worktree and process deadline.
- Platform behavior must be tested through an explicit OS seam; tests that use
  the host `runtime.GOOS` cannot prove both branches.

## Acceptance criteria

- [x] Windows agy reviewer argv omits `--sandbox` while retaining
  `--dangerously-skip-permissions`, `--add-dir`, model pin and timeout.
- [x] Non-Windows agy reviewer argv continues to include `--sandbox`.
- [x] Pi reviewer argv remains unchanged and carries no agy-specific flags.
- [x] The Windows command builder contains no `--sandbox`, and a real agy probe
  can run
  `git` without UAC elevation.

## References

- Bitácora board: the GitHub issue / Project item tracking this spec (see the `issue:` frontmatter field)
- Antigravity sandbox docs: `https://antigravity.google/docs/cli/sandbox`
- Related incident: WIN-014 review transcript, 2026-09-29
- Related safety issue: #1649
