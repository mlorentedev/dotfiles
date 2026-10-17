---
id: "DX-005-zsh-completions"
type: spec
status: implementing # draft | implementing | verifying | archived
created: "2026-10-16"
issue: "mlorentedev/dotfiles#2645"   # repo#NNN — GitHub issue / Project item that tracks this spec
tags: [spec, proposal]
template_version: "1.0"
wip_override: "owner exception 2026-10-16: two small shell-UX PRs asked for in one session (11 active, limit 10, 2026-10-16)"
---

# DX-005-zsh-completions

## Why

On a machine without oh-my-zsh (the Mac), `.zshrc` runs `compinit -C` unconditionally, so `~/.zcompdump` is never rebuilt and a completion added after it is invisible; `compinit` also runs before the Homebrew `shellenv` block, and nothing writes `_dotf` or `_herdr` into any `fpath` directory, so `dotf <TAB>` and `herdr <TAB>` complete nothing. Measured 2026-10-16: the dump is dated 2026-10-05, `ls /opt/homebrew/share/zsh/site-functions` holds neither file, and no installer or converge step generates them. Issue #2645.

## What

After this PR a new shell:

- builds the completion dump again when it is older than 24 hours or when a file in `~/.zsh/completions` is newer than it, and trusts it (`compinit -C`) otherwise, so startup stays as fast as today;
- sees Homebrew's site-functions and `~/.zsh/completions` on `fpath` before `compinit`, each once;
- offers a menu on repeated TAB (`menu select`);
- completes `dotf` and `herdr`, because `dotf converge` (new step `shell-completions`) writes `~/.zsh/completions/_dotf` and `_herdr` from each tool's own `completion zsh`, and a second run reports no change.

## Out of scope

- Marking the plugin's pin in `versions.conf` in the PR that adds the Go support: a released `dotf` would read the mark as a registry tool and fail every machine's sync (see Risks). The mark is its own one-line PR after a release.
- Linuxbrew (`/home/linuxbrew/.linuxbrew`): the brew block only knows the two macOS prefixes today and widening it changes Linux PATH order. Filed separately if wanted.
- bash completions and PowerShell: the step is zsh-only (linux, darwin).

## Risks / open questions

- The `compinit` ordering changes for every machine, so the sandboxed bats (`tests/zshrc-guards.bats`) must stay green and must prove the stale-dump rule on both sides.
- Moving the brew block earlier keeps it ahead of the later PATH prepends, so PATH order is unchanged.
- zsh-autosuggestions installs through mise's `http:` backend (no registry entry, no release asset): a table in `cli/internal/tools/mise_files.go` carries each version's archive URL and sha256, `ParseMisePins` rejects a version with no digest, and the sync checks the plugin file itself. Windows skips it. A pin named in `versions.conf` before a released `dotf` carries the table breaks that machine's sync, so `tests/versions-conf.bats` holds the order (floor 0.72.0).
- The step runs `dotf completion zsh` from PATH: it writes the completions of the `dotf` it finds, which is the pinned release.

## Acceptance criteria

- [x] AC1: a dump older than 24 h, or older than a file in `~/.zsh/completions`, is rebuilt; a fresh one is not (`-C`).
- [x] AC2: Homebrew's site-functions and `~/.zsh/completions` are on `fpath` before `compinit`, with no duplicates; the brew block no longer follows `compinit`.
- [x] AC3: `menu select` is set.
- [x] AC4: `dotf converge --only shell-completions` writes `_dotf` and `_herdr`, a second run reports `changed=0`, and a tool absent from PATH is skipped by name.
- [x] AC5: startup of `zsh -ic exit` does not regress by more than 10 ms against the pre-change median.
- [x] AC6: a pin whose name is in the file-tool table installs through mise's http backend at a pinned digest, is verified by its file, is repaired when the file is gone, is not pruned as an orphan, is skipped on Windows, and a second sync reports `changed=0`; `.zshrc` sources it after `compinit` only when the file is there.

## References

- Bitácora board: #2645
- Prior art: ctechols' compinit-once-a-day gist (the 24 h rule); caching generated completions as `_tool` files in an fpath directory instead of `source <(tool completion zsh)` at startup.
