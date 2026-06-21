---
tags: [spec, verification]
created: "2026-06-21"
---

# Verification - CLI-020-dotf-init-windows-repoint

## Evidence

- [x] **profile `project-init` calls `dotf init`** → `powershell/profile.ps1` body is `dotf init $ProjectName --stack $Stack`; bats `profile.ps1 has project-init function` (powershell-profile.bats) + `profile.ps1 valid PowerShell syntax` pass.
- [x] **setup-windows.ps1 cleans up orphans, does not deploy** → new bats `setup-windows.ps1 removes retired init .ps1 orphans, does not deploy them (CLI-020)` passes; `setup-windows.ps1 passes PSScriptAnalyzer` + `valid PowerShell syntax` pass.
- [x] **3 `.ps1` + 2 bats deleted; ci.yml updated** → `git rm` of all five; `ci.yml` PSScriptAnalyzer list (`:55`) + bats run (`:253`) no longer reference them.
- [x] **Parity verified (the pre-deletion gate)** → isolated `VAULT_PATH=$tmp dotf init <tmp> --stack go --skip-github` created `<VAULT>/10_projects/<repo>/{00-context.md,10-roadmap.md,memory/MEMORY.md}` and ran structure/agents/ci/stack/git/pre-commit/vault; `--skip-github` honored. Superset of the retired `.ps1` (which seeded a stale `11-tasks.md`).
- [x] **Guard-grep clean** → no live ref to `init-(project|repo-agents|repo-github-defaults).ps1` outside the intentional cleanup list, the updated bats, and historical records.

## Test status

- Test suite: `bats tests/setup-windows.bats tests/powershell-profile.bats` → **all pass, exit 0** (incl. PSScriptAnalyzer + PowerShell-syntax checks for both edited scripts).
- Manual smoke test: isolated `dotf init` run on Windows with a throwaway `VAULT_PATH` (then cleaned up) — vault entry + full scaffold produced; confirmed the Windows memory junction is (re)created by `claude-session-start.ps1` `Ensure-MemoryJunction`, so delegating it is safe.
- No regressions: yes — full Windows bats suite green; only the deploy→cleanup assertion intentionally changed.

## Decisions made during implementation

- **No Go change.** Parity gate came back green; `dotf init` is at/above parity on Windows, so CLI-020 stayed a pure repoint+delete.
- **Junction delegated, not ported.** `dotf init`'s Go `linkMemory` is non-Windows by design; the eager junction the `.ps1` created is recreated every session by `Ensure-MemoryJunction`. The transient gap (no junction until the first Claude session) is harmless — a junction's only consumer is a session.
- **`agents-spec-section.md` left untouched** despite a stale `init-repo-agents.ps1`/#380 mention: it is a vault-SSOT, drift-tested template (`cli/internal/initrepo/drift_test.go:32`). Editing only the embedded copy would trip the drift guard → fold into **#461** (template re-vendor).

## Promotion candidates

- [x] Lesson for `docs/lessons.md`? **yes** — "first real strangler-fig deletion: a parity gate must cover *every* behavior (vault-path resolution, seeded files, OS-specific side effects like the Windows junction); a deliberately-different Go design still counts as parity when a downstream consumer reconstructs the omitted effect." (capture at archive)
- [ ] ADR-worthy? no — executes ADR-020/021, no new decision.
- [ ] New cross-project pattern? no — repo-specific.

## Archive checklist

- [ ] `proposal.md` frontmatter `status: archived`
- [ ] Folder moved to `specs/archive/CLI-020-dotf-init-windows-repoint/`
- [ ] Backlog: close #489 (PR link) — bitácora auto-moves to Done
- [ ] Promotions above executed (the `docs/lessons.md` entry)
