---
tags: [spec, verification, templates]
created: "2026-05-13"
---

# Verification - POLISH-005-linux-idempotence-ci

## Evidence
 
- [x] Criterion 1 (Integration CI job) -> `tests/Dockerfile.integration` and `.github/workflows/ci.yml:integration` executes the test container
- [x] Criterion 2 (Run twice with snapshot+diff) -> `tests/verify-setup.bats` Section 12 test "POLISH-005: second setup-linux.sh run exits 0 cleanly with zero config diff"
- [x] Criterion 3 (Zero config diff) -> Verified in Docker container: sha256 checksums across `~/.dotfiles`, `~/.claude`, `~/.gemini`, `~/.config/opencode`, `~/.zsh`, `~/.bash`, `~/.ssh`, `~/.zshrc`, `~/.bashrc`, `~/.profile`, `~/.gitconfig`, `~/.tmux.conf` are 100% byte-identical.
- [x] Criterion 4 (No duplicate rc entries) -> `tests/verify-setup.bats` asserts exact count of 1 for path and function exports.
 
 ## Test status
 
 - Test suite: `docker run --rm dotfiles-integration-test` -> **63/63 pass (0 failures)**
 - Bats test suite: `bats tests/verify-setup.bats` -> clean pass
 - No regressions in existing test suite: yes (all 1359 Bats tests + Go test suite pass)
 
 ## Decisions made during implementation
 
 - **Fixed declarative convergence in `setup-linux.sh`**:
   1. `merge_claude_settings` was sorting `.permissions.allow` via `unique` only on subsequent merges, leaving initial bootstrap unsorted. Added normalization on initial bootstrap so both code paths produce identical json formatting.
   2. Third-party installers (like `opencode` install script) append paths to rc files during initial installation. Added a final `deploy_file` of `.zshrc`, `.bashrc`, `.profile` at the end of `setup-linux.sh` to immediately restore declarative purity on the first run.

## Promotion candidates

Before archiving, flag what (if anything) should be promoted to the vault. If all three are "no", archive in repo is the only persistence.

- [ ] Lesson for `<area>/90-lessons.md`? <yes / no - one line of what>
- [ ] ADR-worthy decision for `<area>/30-architecture/adr-XXX.md`? <yes / no - one line of what>
- [ ] New pattern candidate for `00_meta/patterns/`? Only if this recurs in >1 project. <yes / no - one line>

## Archive checklist

- [ ] `proposal.md` frontmatter set to `status: archived`
- [ ] Folder moved: `specs/POLISH-005-linux-idempotence-ci/` -> `specs/archive/POLISH-005-linux-idempotence-ci/`
- [ ] Backlog entry in vault `11-tasks.md` ticked with PR link
- [ ] Promotions above executed (if any)
