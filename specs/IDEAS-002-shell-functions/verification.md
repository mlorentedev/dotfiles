---
tags: [spec, verification, ideas-002]
created: "2026-05-25"
---

# Verification - IDEAS-002-shell-functions

> Status: skeleton. Populated by the implementation PR on the feature branch.

## Evidence

| # | Criterion | Evidence |
|---|---|---|
| 1 | `.zsh/functions.zsh` exists with 6 documented functions | _pending_ |
| 2 | `shellcheck` exits 0 on the file | _pending_ |
| 3 | `.zshrc` and `.bashrc` source the file | _pending_ |
| 4 | Bats: `mkd` creates dir + changes to it | _pending_ |
| 5 | Bats: `dataurl` round-trips file → URL → file | _pending_ |
| 6 | Bats: `gz` reports both sizes | _pending_ |
| 7 | Bats: `targz` gzip-only path produces valid tarball | _pending_ |
| 8 | Cross-shell: bash + zsh both source cleanly | _pending_ |
| 9 | Smoke tests for `server` + `getcertnames` (skippable) | _pending_ |

## Test status

- Test suite: `bats tests/shell-functions.bats` → _pending_
- Shellcheck: `shellcheck .zsh/functions.zsh` → _pending_
- Manual smoke: `mkd /tmp/x/y/z && pwd` → expect `/tmp/x/y/z` — _pending_
- No regressions in cumulative bats: _pending_

## Decisions made during implementation

_Populated during implementation. Likely topics:_

- `dataurl` fallback MIME when `file` missing: chosen value (likely `application/octet-stream`).
- `targz` detection order: chosen sequence (zopfli → pigz → gzip).
- Whether `server` opens browser on Linux (xdg-open) vs Mac (open) vs neither (CI-safe default).
- Exact handling of `getcertnames` certificate parsing edge cases (wildcard SANs, IP-only certs).
- If IDEAS-003 landed first: how the sourcing loop picked up `.zsh/functions.zsh`.

## Promotion candidates

Before archiving, flag what (if anything) should be promoted to the vault. If all three are "no", archive in repo is the only persistence.

- [ ] Lesson for `10_projects/dotfiles/90-lessons.md`? **Maybe** — "graceful degradation pattern" if `targz` detection logic generalizes to other tools (zstd, brotli...).
- [ ] ADR-worthy decision for `10_projects/dotfiles/30-architecture/adr-XXX.md`? **No** — utility addition, not architectural.
- [ ] New pattern candidate for `00_meta/patterns/`? **No** — these are stdlib-equivalents, not a novel pattern.

## Archive checklist

- [ ] `proposal.md` frontmatter set to `status: archived`
- [ ] Folder moved: `specs/IDEAS-002-shell-functions/` → `specs/archive/IDEAS-002-shell-functions/`
- [ ] Backlog entry in vault `11-tasks.md` ticked with PR link
- [ ] Promotions above executed (if any)
