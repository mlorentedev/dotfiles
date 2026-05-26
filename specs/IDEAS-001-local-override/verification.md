---
tags: [spec, verification, ideas-001]
created: "2026-05-25"
---

# Verification - IDEAS-001-local-override

> Status: skeleton. Populated by the implementation PR on the feature branch (not this research worktree).

## Evidence

| # | Criterion | Evidence |
|---|---|---|
| 1 | `.zshrc` ends with conditional source of `$HOME/.zshrc.local` | _pending_ |
| 2 | `.bashrc` ends with conditional source of `$HOME/.bashrc.local` | _pending_ |
| 3 | `.gitignore` includes `.zshrc.local` and `.bashrc.local` | _pending_ |
| 4 | `.zshrc.local.example` and `.bashrc.local.example` exist with ≥2 use-cases each | _pending_ |
| 5 | Bats: zsh sources local file when present | _pending_ |
| 6 | Bats: zsh no-ops cleanly when local file absent | _pending_ |
| 7 | Bats: same coverage for bash | _pending_ |
| 8 | Drift detector exit 0 post-deploy | _pending_ |

## Test status

- Test suite: `bats tests/local-override.bats` → _pending_
- Manual smoke test: `echo 'export TEST=1' > ~/.zshrc.local && zsh -ic 'echo $TEST'` should print `1` — _pending_
- No regressions in existing test suite (`bats tests/*.bats`): _pending_

## Decisions made during implementation

_Populated during implementation. Likely topics:_

- Should `.local` files load before or after the `load-secrets.sh` invocation? Recommendation: AFTER, so `.local` can override secret-derived env vars if needed.
- Exact insertion location vs end-of-file when there's a trailing newline / heredoc / etc.
- Whether to add `.shellrc.local` (R3 open question — currently deferred).

## Promotion candidates

Before archiving, flag what (if anything) should be promoted to the vault. If all three are "no", archive in repo is the only persistence.

- [ ] Lesson for `10_projects/dotfiles/90-lessons.md`? Likely **yes** — meta-lesson on "non-sensitive machine-local vs sensitive: don't conflate". Candidate body: "The `.local` override pattern complements (not replaces) the age secrets system. age = secrets, .local = non-sensitive machine-local config. Keeping the two systems orthogonal prevents users from drifting toward putting API keys in `.local`."
- [ ] ADR-worthy decision for `10_projects/dotfiles/30-architecture/adr-XXX.md`? **Probably no** — small mechanism, not architectural.
- [ ] New pattern candidate for `00_meta/patterns/`? **Maybe** — `pattern-rc-local-override` if other projects adopt; defer until recurrence detected.

## Archive checklist

- [ ] `proposal.md` frontmatter set to `status: archived`
- [ ] Folder moved: `specs/IDEAS-001-local-override/` → `specs/archive/IDEAS-001-local-override/`
- [ ] Backlog entry in vault `11-tasks.md` ticked with PR link
- [ ] Promotions above executed (if any)
