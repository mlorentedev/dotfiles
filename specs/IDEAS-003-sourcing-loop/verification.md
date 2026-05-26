---
tags: [spec, verification, ideas-003]
created: "2026-05-25"
---

# Verification - IDEAS-003-sourcing-loop

> Status: skeleton. Populated by the implementation PR on the feature branch.

## Evidence

| # | Criterion | Evidence |
|---|---|---|
| 1 | `.zshrc` uses brace-expanded source loop | _pending_ |
| 2 | `.bashrc` uses brace-expanded source loop | _pending_ |
| 3 | Loop iteration guarded by `[ -r ] && [ -f ]` | _pending_ |
| 4 | Trailing `unset __src_f` — no namespace leak | _pending_ |
| 5 | Bats parity: env state identical pre/post | _pending_ |
| 6 | Shell startup ±10% of baseline (bash + zsh) | _pending_ |
| 7 | Drift detector exit 0 post-deploy | _pending_ |
| 8 | No regressions in cumulative bats suite | _pending_ |

## Test status

- Test suite: `bats tests/sourcing-loop.bats` → _pending_
- Performance: `profile-shell` pre/post deltas → _pending_
- Drift: `scripts/drift-detector.sh` → _pending_
- Manual smoke: open fresh terminal, confirm no warnings — _pending_

## Decisions made during implementation

_Populated during implementation. Likely topics:_

- Final variable name (`__src_f` per spec, but might rename if collision detected).
- Exact ordering of the brace-expanded list (matches current pre-refactor order; confirm).
- Whether IDEAS-002 functions.zsh slot was included in the brace expansion (depends on landing order).
- Profile-shell pre vs post numbers (record both sets).

## Promotion candidates

- [ ] Lesson for `10_projects/dotfiles/90-lessons.md`? **Maybe** — "shell rc parity testing pattern" (capture pre-state, refactor, assert post-state == pre-state) generalizes to other rc refactors.
- [ ] ADR-worthy decision for `10_projects/dotfiles/30-architecture/adr-XXX.md`? **No** — refactor, not architecture.
- [ ] New pattern candidate for `00_meta/patterns/`? **Maybe** — `pattern-rc-refactor-parity` if applied again (load-secrets refactor, etc.).

## Archive checklist

- [ ] `proposal.md` frontmatter set to `status: archived`
- [ ] Folder moved: `specs/IDEAS-003-sourcing-loop/` → `specs/archive/IDEAS-003-sourcing-loop/`
- [ ] Backlog entry in vault `11-tasks.md` ticked with PR link
- [ ] Promotions above executed (if any)
