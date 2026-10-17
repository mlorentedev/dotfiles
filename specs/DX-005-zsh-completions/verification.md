---
tags: [spec, verification, templates]
created: "2026-10-16"
---

# Verification - DX-005-zsh-completions

## Evidence

PR 1 (`.zshrc` + `shell-completions`). Autosuggestions is PR 2, not covered here.

- [x] AC1 (dump rule) -> `tests/zshrc-guards.bats`: "a fresh dump is trusted…", "a dump older than a day is rebuilt…", "a completion newer than a fresh dump rebuilds it…". Mutation check: forcing `compinit -C` fails the last two; starting `_zcfull` empty fails the aged-dump test.
- [x] AC2 (fpath) -> "~/.zsh/completions leads fpath and fpath holds no duplicate", "the Homebrew block runs before compinit…".
- [x] AC3 (menu) -> "completion menus are selectable (menu)".
- [x] AC4 (step) -> `go test ./internal/converge -run ShellCompletions` (write, drift, dry run, skipped tool, nothing-to-run, empty or failing completion); live: `dotf converge --only shell-completions` in a scratch HOME printed `1 changed … wrote _dotf, _herdr`, then `0 changed, 1 ok`.
- [x] AC5 (latency) -> medians of 20 runs of `zsh -ic exit` under a scratch ZDOTDIR, warm dump: main `.zshrc` 44.4 and 45.5 ms; this branch 44.2 and 44.7 ms; this branch with `~/.zsh/completions` populated 43.9 ms. The daily full `compinit` costs 51.5 ms once.

PATH order: `print -l $path` under a clean `PATH=/usr/bin:/bin:/usr/sbin:/sbin` and an empty `HOMEBREW_PREFIX`, main `.zshrc` versus this branch: identical (73 entries, `/opt/homebrew/bin` at 47 in both), so moving the brew block did not reorder PATH.

End to end: a shell with the written files reports `_comps[dotf]=1`, `_comps[herdr]=1`, and `~/.zsh/completions` first on `fpath`.

## Test status

- `bats tests/zshrc-guards.bats` 12/12; `bats tests/guard-zsh-*.bats` green.
- `cd cli && go build ./... && go vet ./... && go test ./internal/converge ./internal/cmd`; `GOOS=windows go vet` on both; `golangci-lint run` (pinned 2.12.2) 0 issues.
- Not measured: Linux (no Linux box in this session); the step and the rc use no macOS-only call, and the step is `linux, darwin`.

## Decisions made during implementation

- The step compares content, not mtime, and runs each tool's own `completion zsh`; a stdout-only runner (`ExecStdout`) keeps a tool's stderr warning out of the file.
- The rc, not the step, owns dump freshness: a completion newer than the dump forces a full `compinit`, so the step needs no knowledge of `~/.zcompdump` or `ZDOTDIR`.
- Linuxbrew prefix left out of the brew block (out of scope, changes Linux PATH order).

## Promotion candidates

- [x] Lesson for the repo's `docs/lessons/`? no: the 24 h rule is the documented upstream pattern; nothing here was surprising.
- [x] ADR-worthy decision for the repo's `docs/adr/adr-XXX.md`? no: no alternative with lasting cost was chosen.
- [x] New pattern candidate for `00_meta/patterns/`? no: single project.

## Archive checklist

- [ ] `proposal.md` frontmatter set to `status: archived`
- [ ] Folder moved: `specs/DX-005-zsh-completions/` -> `specs/archive/DX-005-zsh-completions/`
- [ ] Bitácora board ticket for this spec moved to Done / closed with PR link (ADR-018)
- [ ] Promotions above executed (if any)
