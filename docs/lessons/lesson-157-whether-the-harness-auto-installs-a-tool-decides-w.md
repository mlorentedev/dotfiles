---
id: lesson-157-whether-the-harness-auto-installs-a-tool-decides-w
type: lesson
status: active
created: "2026-08-07"
owner: manu
tags: [lesson, dotfiles]
---

# Lesson 157: Whether the harness auto-installs a tool decides whether its config deploy is conditional

**Context**: HARNESS-051 (`#771`) added a Copilot native-skill deploy target to the shared `skills.deploy[]` manifest pipeline (`scripts/compile-harness.sh` + `setup-windows.ps1`'s `Deploy-SkillRecord`), unconditionally — matching how opencode/agy/pi are already deployed to.

**Problem**: The integration container's `tests/verify-setup.bats` (BUG-001/PR#40, years old) asserts `~/.copilot` must not exist when Copilot is absent. The new deploy target broke that on its first real CI run. Two fixes were on the table: make the deploy unconditional and retire the old test (matching the other three agents), or gate the new target and keep the invariant. The deciding fact was not in either PR — it was in `setup-linux.sh` itself: opencode and agy are installed **by this repo's own setup script** (`curl | bash` / the official installer, unconditionally, on every run), so by the time skill-deploy runs they are guaranteed present — unconditional config deploy for them is safe by construction. Copilot has a written, deliberate policy to the contrary (`BUG-003`: "Linux side is detect-and-act -- no auto-install"), so it is not equivalent to the other three; it only *looks* like the same kind of target because it is declared in the same manifest array.

**Solution**: Added a manifest-declared `requires_command` field, read by both deploy engines, checked with `command -v` (bash) / `Get-Command` (PowerShell) before writing anything for that target. opencode/agy/pi carry no such field and remain unconditional.

**Rule**: In a manifest-driven multi-agent deploy pipeline, "does this repo's own setup install the tool" is the fact that decides whether that tool's config deploy should be unconditional or presence-gated — not symmetry with sibling entries in the same array, and not what the pipeline already does for other agents. Check the actual install step before assuming a new target should behave like its neighbors.
