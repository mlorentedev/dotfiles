# dotfiles — Backlog

> Personal development environment: shell configs, AI integration, secrets management.

## P0 — High Priority

- [ ] **PowerShell CI validation**: PowerShell scripts (`setup-windows.ps1`, `init-project.ps1`, `profile.ps1`) not checked in CI. Add PSScriptAnalyzer or syntax validation step.
- [ ] **Consistent `set -euo pipefail`**: Not all shell scripts use strict error handling consistently. Audit and standardize.

## P1 — Medium Priority

- [ ] **Tool version pinning**: Java 21, Go 1.23, Python 3.12 hardcoded in scripts but no central version manifest. Add a `versions.conf` or similar for single-source-of-truth.
- [ ] **Health check script**: No way to verify all expected tools are installed at correct versions after setup. Add `scripts/healthcheck.sh`.
- [ ] **Automated setup testing**: CI now runs bats test suite (95 tests) + shellcheck. Remaining: integration test that runs `setup-linux.sh` in a container and verifies the result.
- [ ] **Secrets backup docs**: `backup-secrets-to-usb.sh` (69 LOC) has no usage guide in docs/.

## P2 — Low Priority / Nice-to-Have

- [ ] **Multi-profile support**: No work/personal/side-project profile separation. All configs are global.
- [ ] **Dotfiles diff/drift detection**: No way to detect when local configs have diverged from repo. Add `scripts/diff-check.sh`.
- [ ] **CHANGELOG.md**: No changelog tracking feature additions and fixes across commits.
- [ ] **Shell startup profiling**: No timing instrumentation for shell startup. Add opt-in profiling to identify slow plugins/scripts.

## Done

- [x] **CI bats test execution**: Replaced 5 inline tests with full bats suite (95 tests) + shellcheck for root scripts.
- [x] **Project-specific CLAUDE.md**: Populated `.claude/CLAUDE.md` with shell compat rules, verification commands, key file map, and workflows.
- [x] **tasks/lessons.md**: Created with 4 seed entries from shell hardening work.
- [x] **Windows MCP parity**: Added excalidraw MCP server registration to `setup-windows.ps1`.
