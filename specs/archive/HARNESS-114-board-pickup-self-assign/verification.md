---
id: "HARNESS-114-board-pickup-self-assign"
type: verification
template_version: "1.0"
---

# HARNESS-114 — verification

## Automated

- `bats tests/install-git-hooks.bats tests/board-pickup.bats` → 15/15.
- `./scripts/check-bats-names.sh tests/` → clean (no non-ASCII / duplicate `@test` names).
- `shellcheck -S warning git-hooks/post-checkout git-hooks/lib/board-pickup.sh` → clean.

## Behavioural (bats, stubbed gh)

| Case | Expected |
|------|----------|
| open issue, current repo | self-assigns there, no fallback |
| current-repo issue missing | falls back to `knowledge` |
| closed issue | never assigned (open-state guard) |
| file checkout (flag=0) | immediate no-op, gh never called |
| branch without issue number | no-op, gh never called |
| gh fails entirely | exit 0 (fire-and-forget) |
| repo is `knowledge` itself | assigns once, no redundant fallback |

## Manual / live

- T7 (deferred — session rate limit): on the next real pickup, confirm `gh issue edit
  --add-assignee @me` on an OPEN Backlog issue triggers `bitacora-status.yml` → In Progress.
  The Action's assign→In-Progress behaviour is already verified under HARNESS-010 (#270).

## Result

PASS (automated + behavioural). One live e2e step deferred; see tasks T7.
