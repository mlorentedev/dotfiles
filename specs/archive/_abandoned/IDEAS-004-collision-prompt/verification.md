---
tags: [spec, verification, ideas-004]
created: "2026-05-25"
---

# Verification - IDEAS-004-collision-prompt

> Status: skeleton. Populated by the implementation PR on the feature branch.

## Evidence

| # | Criterion | Evidence |
|---|---|---|
| 1 | `prompt_collision()` exists in `utils.sh` | _pending_ |
| 2 | `link_file()` integrates with it | _pending_ |
| 3 | `DOTFILES_SETUP_FORCE=1` env var honored | _pending_ |
| 4 | Bats: all 6 input paths covered | _pending_ |
| 5 | Bats: force-mode no-hang test | _pending_ |
| 6 | Bats: timestamp-collision protection | _pending_ |
| 7 | CI workflow exports the force env var | _pending_ |
| 8 | Cross-shell parity (bash + zsh) | _pending_ |
| 9 | No regressions in setup-linux integration suite | _pending_ |

## Test status

- Unit test: `bats tests/collision-prompt.bats` → _pending_
- Integration: sandbox `setup-linux.sh` re-run on pre-populated `~/` → _pending_
- Shellcheck: `shellcheck scripts/utils.sh` → _pending_
- CI workflow lint: `actionlint .github/workflows/*.yml` → _pending_

## Decisions made during implementation

_Populated during implementation. Likely topics:_

- Final default action under force mode (R5 — likely `backup`).
- Implementation of the `*-all` state global (`__DOTFILES_COLLISION_MODE` or alternative).
- Exact timestamp format for backup filenames.
- Whether the prompt was rendered with color (TTY check) or plain.
- Stdin-mocking strategy in bats (heredoc vs file vs pipe).

## Promotion candidates

- [ ] Lesson for `10_projects/dotfiles/90-lessons.md`? **Likely yes** — "Force-mode env var is mandatory for any interactive helper in setup scripts" (generalize from this incident).
- [ ] ADR-worthy decision for `10_projects/dotfiles/30-architecture/adr-XXX.md`? **Maybe** — "setup-linux.sh UX policy: interactive by default, non-interactive via env var" if more interactive prompts get added.
- [ ] New pattern candidate for `00_meta/patterns/`? **Maybe** — `pattern-non-destructive-bootstrap` capturing the backup-by-default principle.

## Archive checklist

- [ ] `proposal.md` frontmatter set to `status: archived`
- [ ] Folder moved: `specs/IDEAS-004-collision-prompt/` → `specs/archive/IDEAS-004-collision-prompt/`
- [ ] Backlog entry in vault `11-tasks.md` ticked with PR link
- [ ] Promotions above executed (if any)
