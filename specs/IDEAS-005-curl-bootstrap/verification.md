---
tags: [spec, verification, ideas-005]
created: "2026-05-25"
---

# Verification - IDEAS-005-curl-bootstrap

> Status: skeleton. Populated by the implementation PR on the feature branch.

## Evidence

| # | Criterion | Evidence |
|---|---|---|
| 1 | `install.sh` exists, executable | _pending_ |
| 2 | Idempotent: clone-or-pull semantics | _pending_ |
| 3 | `DOTFILES_DIR` env var honored | _pending_ |
| 4 | `DOTFILES_REPO` env var honored | _pending_ |
| 5 | Fails fast without `git` | _pending_ |
| 6 | Fails fast on non-git pre-existing dir | _pending_ |
| 7 | README has one-liner + verify caveat | _pending_ |
| 8 | Bats: clone-path test | _pending_ |
| 9 | Bats: pull-path test | _pending_ |
| 10 | Shellcheck clean | _pending_ |
| 11 | Smoke test in fresh container | _pending_ |

## Test status

- Bats: `bats tests/install-bootstrap.bats` → _pending_
- Shellcheck: `shellcheck install.sh` → _pending_
- Smoke test (Docker): `docker run --rm -it ubuntu:24.04 bash -c "apt-get update && apt-get install -y git curl && curl -fsSL https://raw.githubusercontent.com/<user>/dotfiles/main/install.sh | DOTFILES_SKIP_SETUP=1 bash"` → _pending_

## Decisions made during implementation

_Populated during implementation. Likely topics:_

- Final `DOTFILES_REPO` URL.
- Whether to add SSH-vs-HTTPS support (currently HTTPS-only for unauthenticated clone).
- Specific error messages for the 3 failure paths.
- Whether to log clone/pull progress or stay quiet.
- Whether `install.sh` itself should support `--update` / `--force` flags (currently relies on env vars).

## Promotion candidates

- [ ] Lesson for `10_projects/dotfiles/90-lessons.md`? **Likely yes** — "curl | bash safety: minimal install.sh + verify-before-pipe instruction in README" (codifies the trade-off).
- [ ] ADR-worthy decision for `10_projects/dotfiles/30-architecture/adr-XXX.md`? **Yes** — "Canonical entry point: install.sh (zero-state) → setup-linux.sh (delegator)" if the layering is important to lock down.
- [ ] New pattern candidate for `00_meta/patterns/`? **Maybe** — `pattern-curl-bootstrap-safe` if other personal repos adopt the same shape.

## Archive checklist

- [ ] `proposal.md` frontmatter set to `status: archived`
- [ ] Folder moved: `specs/IDEAS-005-curl-bootstrap/` → `specs/archive/IDEAS-005-curl-bootstrap/`
- [ ] Backlog entry in vault `11-tasks.md` ticked with PR link
- [ ] Promotions above executed (if any)
