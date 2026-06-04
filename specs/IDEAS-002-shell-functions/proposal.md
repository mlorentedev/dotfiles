---
id: "IDEAS-002-shell-functions"
type: spec
status: implementing
created: "2026-05-25"
tags: [spec, proposal, ideas-002, shell, dotfiles-survey, tier-1]
template_version: "1.0"
---

# IDEAS-002: Swiss-army shell functions

> **Naming**: file lives at `<repo>/specs/IDEAS-002-shell-functions/proposal.md`.
> Origin: dotfiles-survey research, Tier-1 idea (#2) from mathiasbynens (`.functions`).

## Why

<!-- from research/dotfiles-survey.md §"Top 6 ideas a aplicar" #2: Swiss-army funcs from mathiasbynens, Tier-1 ROI -->

Six small daily-use utilities show up over and over in shell sessions: making a dir and cd-ing into it (`mkd`), compressing with the best available compressor (`targz`), turning a file into a `data:` URL (`dataurl`), serving the CWD over HTTP (`server`), inspecting a TLS cert's CN+SANs (`getcertnames`), and seeing what a file gzips to (`gz`). Each one is 5-15 lines. mathiasbynens has been refining these in `.functions` for over a decade — they are battle-tested, autocontained, and have zero external deps beyond what's already on every dev box. Right now this repo has none of them. Every time the user needs `mkdir -p foo && cd foo` they retype it, every time they need a quick HTTP server they Google `python -m http.server`. Versioning these once removes a thousand tiny paper-cuts and codifies the user's daily toolkit.

## What

A new file `.zsh/functions.zsh` containing six functions. Each is bash+zsh compatible (per the project's Prohibited Patterns table), each has a one-line doc comment, each is autocontained.

| Function | Behavior | Notes |
|---|---|---|
| `mkd <dir>` | `mkdir -p` + `cd` in one command | One-liner |
| `targz <input>` | Create `input.tar.gz`, choosing zopfli → pigz → gzip by availability and input size | Falls back gracefully |
| `dataurl <file>` | Print `data:<mime>;base64,<...>` URL for the file | Uses `file --mime-type` for MIME |
| `server [port]` | `python3 -m http.server` in CWD, opens browser to `localhost:<port>` | Default port 8000 |
| `getcertnames <host>:<port>` | Extract CN + Subject Alternative Names from a TLS cert | Wraps `openssl s_client` + `openssl x509` |
| `gz <file>` | Show original vs gzipped size + ratio | Read-only, doesn't write the .gz |

The file is sourced from `.zshrc` and `.bashrc` (explicitly OR via the IDEAS-003 sourcing loop if that lands first — see Risks). Source order: after `aliases.zsh`, before any prompt setup.

## Out of scope

- **PowerShell port** — tracked as IDEAS-002b. Each function has a PowerShell-idiomatic equivalent (`New-Item -Force` + `Set-Location`, etc.) but is a separate effort.
- **Importing all 50+ functions from mathiasbynens** — only the curated 6. A larger sweep can come later, but each function added carries a maintenance + test cost; start small.
- **Replacing existing aliases with functions** — aliases stay as aliases. This spec only ADDS.
- **Adding completion definitions** for these functions — pure dispatch, no flags, completion is overkill.

## Risks / open questions

- **R1 (BLOCKER for `targz`)**: compression-tool detection. Logic must be: if `zopfli` available AND input < 50MB → zopfli; elif `pigz` available → pigz; else `gzip`. Bats test must cover the gzip-only fallback (CI doesn't have zopfli/pigz). Without graceful degradation, `targz` breaks in CI.
- **R2 (BLOCKER for `dataurl`)**: `file --mime-type` not on every distro (alpine minimal lacks it). Fallback: if `file` missing, default to `application/octet-stream`. Document the degraded behavior.
- **R3**: shellcheck. mathiasbynens's originals use a couple of bash-only patterns (`local` in subshells with assignment, `${var,,}` lowercasing). Each function gets shellcheck'd; rewrite anything that warns.
- **R4 (BLOCKER for portability)**: Prohibited Patterns. No `echo -e` (use `printf '%b'`), no `&>/dev/null` (use `>/dev/null 2>&1`), no `((counter++))` under `set -e`. Audit each function against the table in `.claude/CLAUDE.md`.
- **R5**: name collision. `server`, `mkd`, `gz` are short — risk of stomping a user binary on `$PATH`. Mitigation: document the names in README; users can `alias server=mathiasbynens_server` locally if conflict (IDEAS-001 makes this easy).

## Acceptance criteria

- [ ] `.zsh/functions.zsh` exists and contains 6 functions, each with a one-line doc comment.
- [ ] `shellcheck .zsh/functions.zsh` exits 0 with no warnings.
- [ ] `.zshrc` and `.bashrc` source the file (directly OR via IDEAS-003 loop).
- [ ] Bats test `mkd /tmp/test-ideas002-$$/nested` → directory exists AND `pwd` reports the new dir.
- [ ] Bats test `dataurl <tmp-file-with-known-content>` → output begins with `data:` and contains `;base64,` and decodes back to the original content.
- [ ] Bats test `gz <tmp-file>` → output contains both the original size and the gzipped size (numeric assertions).
- [ ] Bats test `targz <tmp-dir>` → produces `tmp-dir.tar.gz` decompressible with `gzip -d` (gzip-only path in CI).
- [ ] Bats test: bash and zsh both source `.zsh/functions.zsh` cleanly (existing matrix test extended).
- [ ] `server` and `getcertnames` covered by smoke-only tests (require network/python; mark as skippable in CI if needed).

## Completeness review

Standard items considered:

- **Rate limit / cost guard** — N/A.
- **Idempotency** — each function is read-only or creates new files. `mkd` is idempotent (`mkdir -p`).
- **Regression test** — covered per function.
- **Cert provisioning** — N/A (we *consume* certs in `getcertnames`, not provision).
- **Rollback** — single-commit revert removes the file and the source line.

Adding (not in template, load-bearing here):

- **Documentation**: README gets a "Shell helpers" section listing the 6 functions with one-line summaries. Without this, the functions are invisible to anyone who didn't read the spec.
- **Sourcing strategy**: if IDEAS-003 lands first, this spec consumes the loop; if not, explicit `source .zsh/functions.zsh` line. The spec is written to accommodate either order — pick the implementation path at branching time.

## References

- Research source: `research/dotfiles-survey.md` § "Top 6 ideas a aplicar" #2 (Tier 1).
- Upstream: mathiasbynens/dotfiles `.functions` (https://github.com/mathiasbynens/dotfiles/blob/main/.functions).
- Project rules: `.claude/CLAUDE.md` Prohibited Patterns table — MUST adhere.
- Related: IDEAS-003 (sourcing loop) — landing-order dependency, see Risks R-sourcing.
- Related: IDEAS-001 (local overrides) — provides escape hatch for name-collision conflicts.

## LOC estimate

~120 LOC functions + ~80 LOC bats + ~20 LOC README/doc = **~220 LOC total**. Above the 50-LOC threshold; full SDD discipline applies (no `skip-sdd` label).
