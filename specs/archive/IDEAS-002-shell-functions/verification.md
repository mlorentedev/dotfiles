---
tags: [spec, verification, ideas-002]
created: "2026-05-25"
---

# Verification - IDEAS-002-shell-functions

> Implemented on branch `feat/IDEAS-002-shell-functions`.

## Evidence

| # | Criterion | Evidence |
|---|---|---|
| 1 | file exists with 6 documented functions | `.zsh/functions.sh` — mkd, gz, dataurl, targz, server, getcertnames, each with a doc comment (bats: "defines all six functions") |
| 2 | `shellcheck` exits 0 (no warnings) | `shellcheck .zsh/functions.sh` → exit 0 (file carries `# shellcheck shell=bash`) |
| 3 | `.zshrc` and `.bashrc` source the file | bats "`.zshrc` sources .zsh/functions.sh", "`.bashrc` sources .zsh/functions.sh (bash parity)" |
| 4 | `mkd` creates dir + changes to it | bats "mkd creates a nested path and cd's into it" |
| 5 | `dataurl` round-trips file → URL → file | bats "dataurl emits a data: URL that round-trips to the original" |
| 6 | `gz` reports both sizes | bats "gz reports both original and gzipped sizes" |
| 7 | `targz` gzip-compatible path produces valid tarball | bats "targz produces a gzip-decompressible tarball" |
| 8 | Cross-shell: bash + zsh both source cleanly | bats "functions.sh sources cleanly under bash and zsh"; `bash -n` + `zsh -n` both pass |
| 9 | Smoke tests for `server` + `getcertnames` (skippable) | bats "server reports missing python3 instead of hanging", "getcertnames prints usage with no host" |

## Test status

- `bats tests/shell-functions.bats` → **18/18 pass**.
- `shellcheck .zsh/functions.sh` → exit 0, no warnings.
- `bash -n .zsh/functions.sh` / `zsh -n .zsh/functions.sh` → both clean.
- Full suite `bats tests/*.bats` → only 3 pre-existing failures in `shell-profile.bats`
  (#4/#5/#6, environment-dependent timing), identical on `main` before this change —
  no regression introduced by IDEAS-002.
- Manual smoke: `mkd /tmp/x/y/z && pwd` → `/tmp/x/y/z`; `dataurl` round-trips;
  `targz` output decompresses with `gzip -dc | tar -t`.

## Decisions made during implementation

- **File placement deviates from the proposal's literal `.zsh/functions.zsh`.**
  `.zsh/functions.zsh` already exists and holds a zsh-only helper (`colormap`,
  which uses zsh prompt/param expansion `${(l:...:)}` / `${${...}}`). ShellCheck
  reports those as hard errors (SC2296/SC2298) that no directive can suppress, so
  a file containing `colormap` can never satisfy AC2 (shellcheck-clean), and it
  cannot be sourced from `.bashrc` as a portable file without future zsh-isms
  breaking bash. The 6 portable functions therefore live in a new sibling file
  `.zsh/functions.sh` (`.sh` = POSIX/portable, `.zsh` = zsh-only by convention),
  sourced by BOTH rc files. This satisfies AC1's intent (6 documented functions
  in a sourced shell file) and keeps the invariants clean. `colormap` is left
  untouched.
- `dataurl` fallback MIME when `file` is missing: `application/octet-stream`
  (R2). `text/*` types get `;charset=utf-8` appended.
- `targz` detection order: zopfli (input < 50MB) → pigz → gzip (R1). zopfli has
  no stdin filter mode, so that branch compresses a temp tar in place; pigz/gzip
  run as stream filters. The test asserts the gzip-compatible property (true for
  all three) rather than a specific compressor.
- `base64` (coreutils) is used instead of mathiasbynens's `openssl base64` to
  drop the openssl dependency from `dataurl` (openssl is still needed by
  `getcertnames`, which is network/smoke-only).
- `server` opens the browser via `xdg-open` (Linux) or `open` (macOS) if present;
  best-effort and non-fatal, so it is CI-safe.

## Promotion candidates

- [ ] Lesson for repo `docs/lessons.md`? **Maybe** — "`.sh` vs `.zsh` split for
  cross-shell portability + the ShellCheck-can't-parse-zsh constraint" is a
  reusable dotfiles insight.
- [ ] ADR-worthy? **No** — utility addition, not architectural.
- [ ] New vault pattern? **No** — stdlib-equivalents, not a novel pattern.

## Archive checklist (post-merge)

- [ ] `proposal.md` frontmatter set to `status: archived`
- [ ] Folder moved: `specs/IDEAS-002-shell-functions/` → `specs/archive/IDEAS-002-shell-functions/`
- [ ] Backlog entry in vault `11-tasks.md` ticked with PR link
