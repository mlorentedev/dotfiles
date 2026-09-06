---
id: "HARNESS-114-board-pickup-self-assign"
type: tasks
template_version: "1.0"
---

# HARNESS-114 — tasks

- [x] T1 — `git-hooks/post-checkout` dispatcher: background the helper, then chain the repo-local hook.
- [x] T2 — `git-hooks/lib/board-pickup.sh`: parse `<prefix>/<issue>-<slug>`; flag/name guards; resolve current-repo → knowledge.
- [x] T3 — self-assign via `gh issue edit --add-assignee @me`, guarded to OPEN issues; fully fail-silent.
- [x] T4 — `scripts/install-git-hooks.sh`: `chmod +x post-checkout` on deploy.
- [x] T5 — tests: `tests/board-pickup.bats` (hermetic, stubbed gh) + deploy assertion in `tests/install-git-hooks.bats`.
- [x] T6 — branch convention in `pattern-git-workflow.md` (knowledge vault).
- [ ] T7 — live e2e: self-assign an open Backlog issue → confirm the `bitacora-status` Action flips it to In Progress (deferred: GitHub GraphQL rate limit hit during the session; the Action itself is HARNESS-010-verified).
