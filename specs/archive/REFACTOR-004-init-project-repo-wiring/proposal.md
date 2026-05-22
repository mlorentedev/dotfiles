---
id: "REFACTOR-004-init-project-repo-wiring"
type: spec
status: archived
created: "2026-05-21"
tags: [spec, refactor, init-project, cross-os, audit-005-followup]
template_version: "1.0"
---

# REFACTOR-004-init-project-repo-wiring

## Why

AUDIT-005 (`30-architecture/audit-005-scripts-classification`) surfaced that `init-repo-agents.sh`, `init-repo-standards.sh`, and `init-repo-github-defaults.sh` — created in 2026-05-14 via SDD-010 and SDD-013 — have **zero recorded invocations** in shell history despite being merged and tested. They are built-but-unused features: the spec lifecycle confirmed code exists, but the *value* (helping bootstrap any new repo with AGENTS.md + standards + GitHub defaults) never landed in user practice. The natural place for this value to materialise is the existing `init-project.{sh,ps1}` flow — when the user runs it to scaffold a personal coding project, these three helpers should run too.

## What

After this PR, `init-project.{sh,ps1}` invokes the three `init-repo-*` helpers automatically as part of the default initialization sequence, immediately after the `.gitignore` block. Each invocation is **non-fatal** (an individual helper failure logs a warning and continues; it does not abort the rest of the init). Users opt out of any of the three via new flags `--skip-agents`, `--skip-standards`, `--skip-github` (PowerShell equivalents: `-SkipAgents`, `-SkipStandards`, `-SkipGithub`). `init-repo-github-defaults` is **auto-skipped** when no `origin` remote is configured (typical for a brand-new local repo) — silent, with a single info log explaining the skip. The three standalone scripts themselves are **unchanged**; their independent CLI invocation continues to work for users who prefer to run them manually on an existing repo.

## Out of scope

- Modifying or deleting any of the three `init-repo-*.{sh,ps1}` scripts. They remain CLI-invokable standalone tools.
- Adding any *new* repo-bootstrap behaviour beyond invoking the existing three.
- Changing the `--work-sdk` early-exit path of `init-project.sh` (only the personal-project default flow gains the new wiring).
- Surfacing the new `--skip-*` flags in `setup-linux.sh` / `setup-windows.ps1`; the flags live in `init-project` and are user-facing only.

## Risks / open questions

- **gh auth state.** `init-repo-github-defaults` requires `gh auth status` to be green. If the user runs `init-project` without gh authenticated, the helper will fail. **Mitigation:** non-fatal invocation already covers this; the warning log explains the requirement. Also auto-skip when no `origin` remote (which captures the common "fresh local repo, no GitHub yet" case before gh is even relevant).
- **Vault path requirement.** `init-repo-agents` + `init-repo-standards` need `VAULT_PATH` (default `$HOME/Projects/knowledge`). If the vault doesn't exist on the host running `init-project`, the helpers fail. **Mitigation:** non-fatal invocation; the user sees a warning rather than a hard exit.
- **Argument forwarding.** `init-repo-github-defaults` takes `--repo <owner/name>` (NOT a path) and auto-derives from `origin`. The wiring must NOT pass the local pwd as `--repo` — invoke without that flag and let the script auto-derive.
- **PowerShell flag idiom.** PS uses `-Switch` not `--switch`. The flag-naming-parity bats assert must lock both spellings.

## Acceptance criteria

- [ ] `./init-project.sh foo python` with `VAULT_PATH` set and `gh` unauthenticated still completes successfully; the GH helper failure logs a warning, not a hard exit.
- [ ] `./init-project.sh --skip-agents foo python` does NOT invoke `init-repo-agents.sh`; the other two still run.
- [ ] `./init-project.sh --skip-standards --skip-github foo python` invokes only `init-repo-agents.sh`.
- [ ] When the new repo has no `origin` remote, `init-repo-github-defaults.sh` is auto-skipped with an info log (not a warning).
- [ ] PowerShell parity: `init-project.ps1 -Stack python -SkipAgents foo` behaves identically.
- [ ] Three new bats parity asserts in `tests/init-project-wiring.bats` (or appended to `tests/init-project.bats` if cleaner): one per skip-flag, plus one cross-OS flag-spelling assert.
- [ ] Full bats suite passes (no regressions on the existing 765 tests).
- [ ] The three `init-repo-*.{sh,ps1}` scripts remain bit-identical to their pre-PR state.

## References

- Vault: `[[10_projects/dotfiles/11-tasks#REFACTOR-004-init-project-repo-wiring]]`
- Audit parent: `[[30-architecture/audit-005-scripts-classification]]`
- Related ADR: none (additive refactor, no architectural change)
- Original specs the scripts came from: SDD-010 (init-repo-standards), SDD-013 (init-repo-agents); `init-repo-github-defaults` shipped post-merged-orphan-incident 2026-05-18 (vault `00_meta/patterns/github-branch-hygiene.md`)
