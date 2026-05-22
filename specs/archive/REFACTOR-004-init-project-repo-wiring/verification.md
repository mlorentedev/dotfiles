---
tags: [spec, verification, refactor-004]
created: "2026-05-21"
---

# Verification — REFACTOR-004-init-project-repo-wiring

## Acceptance criteria mapping

| # | Criterion | Evidence |
|---|-----------|----------|
| 1 | `./init-project.sh foo python` with `VAULT_PATH` set and `gh` unauthenticated still completes successfully; the GH helper failure logs a warning, not a hard exit | Existing bats test 3 (`init-project.sh creates project structure`) and test 4 (zsh variant) both invoke the full default flow with `MOCK_HOME` (no vault templates, no origin remote). Both PASS after the wiring change — confirms `\|\| log_error ... continuing` catches the helper failures without aborting. |
| 2 | `--skip-agents` flag skips `init-repo-agents` only | Bats test 5 (`arg parser recognizes --skip-agents`) + bats test 10 (`helper failures are non-fatal`) + Linux conditional `[ "$SKIP_AGENTS" = "0" ]` lock the contract. |
| 3 | `--skip-standards --skip-github` invokes only `init-repo-agents` | Symmetric to (2): bats tests 6, 7 lock both flag parsers; the corresponding conditionals are independent. |
| 4 | When no `origin` remote, github-defaults auto-skips with info log (not warning) | Bats test 9 (`github-defaults invocation guarded by origin remote check`) locks the `git config --get remote.origin.url` check. Inspection of the inserted block confirms `log_info "No origin remote yet..."` (info, not warn). |
| 5 | PowerShell parity: `init-project.ps1 -SkipAgents foo` behaves identically | Bats tests 11, 12, 13 lock the PS switches + helper-invocation parity + origin-check parity. |
| 6 | New bats parity asserts present | 9 new asserts in `tests/init-project.bats` (tests 5–13). |
| 7 | Full bats suite passes | `bats tests/` -> **774/774 PASS** (was 765/765 pre-PR). 0 regressions. |
| 8 | The 3 `init-repo-*.{sh,ps1}` scripts remain bit-identical | `git diff main -- scripts/init-repo-{agents,standards,github-defaults}.{sh,ps1}` should be empty. Confirmed: no changes to those files in this branch. |

## Commands run

```bash
# Linux syntax + lint
bash -n scripts/init-project.sh                                              # OK
~/.local/bin/shellcheck scripts/init-project.sh                              # OK (no output)

# Windows encoding check (PSScriptAnalyzer non-ASCII rule)
file scripts/init-project.ps1                                                # ASCII text, with CRLF line terminators
grep -P '[^\x00-\x7F]' scripts/init-project.ps1                              # (empty -- no non-ASCII)

# Bats focused
~/.local/bin/bats tests/init-project.bats                                    # 13/13 PASS

# Full bats suite
~/.local/bin/bats tests/                                                     # 774/774 PASS
```

## Post-merge actions

- [ ] Archive this spec folder via PR `chore(REFACTOR-004): archive spec post-merge`
- [ ] Mark REFACTOR-004 as `[x]` in vault `11-tasks.md`
- [ ] Real-world validation: next time `init-project.{sh,ps1}` runs to bootstrap a new repo, the 3 helpers should fire and (if vault + gh are available) materialize AGENTS.md, docs/standards.md, and GitHub defaults automatically. AUDIT-005's "built-but-unused" finding then resolves itself empirically.
