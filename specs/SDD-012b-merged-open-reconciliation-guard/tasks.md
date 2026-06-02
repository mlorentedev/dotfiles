# SDD-012b — Tasks (TDD order)

- [x] Write `tests/check-backlog-merged.bats` fixtures (fake repo + `specs/archive/<id>/`) — failing first.
- [x] Implement `scripts/check-backlog-merged.sh`: parse open full-ids (awk), cross-reference `specs/archive/<id>/`, advisory exit 0/1/2.
- [x] Full-id keying + sub-id (`WIN-002a`) distinctness — bats AC3.
- [x] Repo inference (`<vault>/10_projects/<proj>` → `$HOME/Projects/<proj>`) + `--repo` override + missing-specs skip — bats AC4.
- [x] Wire `vault.sh` dispatcher: `check-merged|merged` case + usage + standalone list.
- [x] Wire `vault-health.sh` section 7: advisory `warn` over `10_projects/*/11-tasks.md`.
- [x] `shellcheck` clean (script + vault.sh + vault-health.sh); `bash -n` clean.
- [x] `bats tests/check-backlog-merged.bats` green (11/11); no regression in `check-backlog-integrity.bats`.
- [x] Run on live `10_projects/dotfiles/11-tasks.md` → exit 0 (AC6).
- [ ] Archive this spec post-merge → `specs/archive/SDD-012b-merged-open-reconciliation-guard/`; tick the vault `11-tasks.md` entry `[x]` with the PR link (dogfooding the guard).
