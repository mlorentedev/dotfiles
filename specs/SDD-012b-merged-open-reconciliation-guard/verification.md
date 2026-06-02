# SDD-012b — Verification

Run from the repo root (or the worktree).

```bash
# Logic (CI-safe, fixture-based)
~/.local/bin/shellcheck scripts/check-backlog-merged.sh scripts/vault.sh scripts/vault-health.sh
~/.local/bin/bats tests/check-backlog-merged.bats          # AC1–AC4: 11/11
~/.local/bin/bats tests/check-backlog-integrity.bats       # no regression in the sibling

# Wiring (AC5)
./scripts/vault.sh check-merged ~/Projects/knowledge/10_projects/dotfiles/11-tasks.md --repo ~/Projects/dotfiles
grep -n 'check-backlog-merged' scripts/vault-health.sh     # advisory section present

# Live backlog (AC6) — expect exit 0 after the 2026-06-01 reconciliation
./scripts/check-backlog-merged.sh ~/Projects/knowledge/10_projects/dotfiles/11-tasks.md ; echo "exit=$?"
```

| AC | Check | Result at build (2026-06-01) |
|---|---|---|
| AC1 | stale-open flagged, names id | ✅ bats |
| AC2 | ticked / no-spec pass | ✅ bats |
| AC3 | full-id keyed; `WIN-002a` distinct | ✅ bats |
| AC4 | repo inference + skip + missing-file | ✅ bats |
| AC5 | `vault check-merged` dispatch + vault-health `warn` | ✅ dispatch exit 0; grep present |
| AC6 | live `11-tasks.md` exit 0 | ✅ exit 0 |

Self-test summary: shellcheck clean · 11/11 new bats · 6/6 sibling bats · dispatcher routes · live file clean.
