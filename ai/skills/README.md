# Skills moved to the vault (SDD-008)

Skills are no longer stored here. The single source of truth for every skill is
the vault:

- **Linux/macOS:** `~/Projects/knowledge/00_meta/skills/<name>/SKILL.md`
- **Windows:** `%USERPROFILE%\Projects\knowledge\00_meta\skills\<name>\SKILL.md`

## How it works

The cross-agent skill pipeline (`scripts/compile-harness.sh`) treats the vault as
the SSOT and renders each skill to its per-agent layout at deploy time:

1. `--refresh` (needs the vault) compiles every `00_meta/skills/<name>` into a
   committed record under `harness/skills/<name>/` (frontmatter validated). The
   records are the in-repo SSOT, so CI and machines without the vault still work.
2. `--deploy` (offline) renders each record to its per-agent path as a regular
   copy — never a symlink — de-symlinking any pre-existing vault symlink first
   (ending the BUG-100 fragility class):
   - claude   → `~/.claude/skills/<name>/`
   - opencode → `~/.config/opencode/commands/<name>.md`
   - agy      → `~/.gemini/skills/<name>/` + `~/.gemini/prompts/<name>.md`
   - copilot  → a catalog injected into `~/.copilot/copilot-instructions.md`
3. `--check` (offline) validates that every record still renders cleanly.

A skill ships only to the agents listed in its `targets:` frontmatter (absent =
all agents). `setup-linux.sh` and `setup-windows.ps1` run the deploy; the harness
manifest (`harness/manifest.json`) declares the deploy matrix.

To add or edit a skill, edit it in the vault and re-run setup. Do not add skill
directories here.

See `00_meta/patterns/pattern-cross-agent-skill-pipeline.md` in the vault.
