---
tags: [spec, tasks, env-contract, paths]
created: "2026-05-19"
---

# Tasks - REFACTOR-002-paths-in-env-contract

## Setup

- [x] Branch: `feat/REFACTOR-002-paths-in-env-contract` (off main).
- [x] Vault entry exists in `11-tasks.md`.

## Implementation (TDD)

### Phase 1 — bats parity test (red)

- [ ] New `tests/env-contract.bats` with cases for: 4 new entries in env-contract.json, exports in .zshrc / .bashrc / powershell/profile.ps1.

### Phase 2 — env-contract.json (green)

- [ ] Add 4 entries to `env_vars[]`: `SCRIPTS_DIR`, `GEMINI_HOME`, `COPILOT_HOME`, `OPENCODE_HOME`. Each `required: false`, `validation: "path_exists"`, with linux + windows defaults.
- [ ] `jq -e . env-contract.json` clean.

### Phase 3 — RC files (green)

- [ ] `.zshrc`: 4 export lines.
- [ ] `.bashrc`: 4 export lines (mirror of .zshrc).
- [ ] `powershell/profile.ps1`: 4 `$env:VAR` assignments (mirror with Windows path syntax).

### Phase 4 — Regression

- [ ] All new bats cases green.
- [ ] Full bats suite green (target: 659 + new cases).
- [ ] `shellcheck --severity=error` clean.
- [ ] Manual smoke: `doctor.sh --check --verbose` reports 4 new vars (after sourcing the new RC files).

## Closing

- [ ] verification.md filled.
- [ ] PR opened.
- [ ] Post-merge: tick REFACTOR-002 in vault `11-tasks.md` + archive spec.
