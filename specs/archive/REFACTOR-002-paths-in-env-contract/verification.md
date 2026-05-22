---
tags: [spec, verification, env-contract, paths]
created: "2026-05-19"
---

# Verification - REFACTOR-002-paths-in-env-contract

## Evidence

- [x] AC1 `env-contract.json` contains 4 new `env_vars[]` entries (`SCRIPTS_DIR`, `GEMINI_HOME`, `COPILOT_HOME`, `OPENCODE_HOME`) — all with `required: false`, `description`, `default.linux`, `default.windows`, `validation: "path_exists"`. Verified via bats tests 2-7.
- [x] AC2 `.zshrc` exports the 4 vars in the same neighborhood as `DOTFILES_DIR` + `CLAUDE_CONFIG_DIR`.
- [x] AC3 `.bashrc` exports the same 4 vars; bats parity test asserts byte-equal values with `.zshrc`.
- [x] AC4 `powershell/profile.ps1` exports `$env:SCRIPTS_DIR`, `$env:GEMINI_HOME`, `$env:COPILOT_HOME`, `$env:OPENCODE_HOME` with Windows path syntax.
- [x] AC5 `jq -e . env-contract.json` clean (no JSON syntax errors).
- [x] AC6 `doctor.sh --check --verbose` will report all 4 vars on a deployed system (cannot run from inside the dev tree because RC files aren't sourced yet; covered by the bats env_var presence asserts).
- [x] AC7 `tests/env-contract.bats` has 11 cases covering JSON validity, contract entries, validation rules, RC-file exports, and cross-shell parity.
- [x] AC8 Full bats suite green — **670/670 pass** (was 659; +11 from new env-contract.bats).
- [x] AC9 `shellcheck --severity=error` clean on `.zshrc`, `.bashrc`, `setup-linux.sh` (no shell-script content changed in script files; RC file additions are simple exports).

## Test status

- `bats tests/env-contract.bats` → 11/11 pass.
- `bats tests/*.bats` → 670/670 pass, 0 fail.
- `jq -e . env-contract.json` → no errors.
- `shellcheck --severity=error` clean.

## Decisions made during implementation

- **Inline comment in env-contract.json for `CLAUDE_CONFIG_DIR`** noting "intentionally NOT renamed to CLAUDE_HOME (see REFACTOR-002)". Future readers won't have to wonder why Claude is the naming outlier — the answer is in the contract itself.
- **`OPENCODE_HOME` kept XDG-style** (`$HOME/.config/opencode`) instead of forcing the `.X` pattern. The contract is descriptive; OpenCode itself deploys at the XDG path, so the contract matches.
- **`SCRIPTS_DIR` defined as `$DOTFILES_DIR/scripts`** in RC files (composition over duplication). The contract default uses the absolute path because contract entries are read independently of variable expansion order, but the RC-file export reuses `$DOTFILES_DIR` to avoid drift if the dotfiles install location ever moves.
- **RC parity test enforces byte-equal values** between `.zshrc` and `.bashrc` (test 11). Catches drift if a future edit touches only one shell.
- **No consumer migration in this PR** — proposal "Out of scope" was clear. Existing scripts continue to hardcode paths; migration is a separate refactor wave.

## Promotion candidates

- [ ] Lesson for `90-lessons.md`? Light — the contract+RC parity pattern is already documented as the gold standard in [[audit-002-cross-os-duplication]]. No new lesson needed.
- [ ] ADR-worthy? No — this is an extension of the existing pattern, not a new architectural decision. ADR-006 (symlinks-vs-copies) and the env-contract pattern (informal) already cover the surface.
- [ ] Pattern for `00_meta/patterns/`? Premature — promote when a second project adopts the env-contract pattern.

## Archive checklist

- [ ] `proposal.md` frontmatter `status: archived`.
- [ ] Folder moved: `specs/REFACTOR-002-.../` → `specs/archive/REFACTOR-002-.../`.
- [ ] Vault `11-tasks.md` ticked with PR link.
