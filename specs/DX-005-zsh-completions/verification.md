---
tags: [spec, verification, templates]
created: "2026-10-16"
---

# Verification - DX-005-zsh-completions

## Evidence

PR 1 (`.zshrc` + `shell-completions`) is AC1-AC5. PR 2 (zsh-autosuggestions, #2654) is AC6, below.

- [x] AC1 (dump rule) -> `tests/zshrc-guards.bats`: "a fresh dump is trusted…", "a dump older than a day is rebuilt…", "a completion newer than a fresh dump rebuilds it…". Mutation check: forcing `compinit -C` fails the last two; starting `_zcfull` empty fails the aged-dump test.
- [x] AC2 (fpath) -> "~/.zsh/completions leads fpath and fpath holds no duplicate", "the Homebrew block runs before compinit…".
- [x] AC3 (menu) -> "completion menus are selectable (menu)".
- [x] AC4 (step) -> `go test ./internal/converge -run ShellCompletions` (write, drift, dry run, skipped tool, nothing-to-run, empty or failing completion); live: `dotf converge --only shell-completions` in a scratch HOME printed `1 changed … wrote _dotf, _herdr`, then `0 changed, 1 ok`.
- [x] AC5 (latency) -> medians of 20 runs of `zsh -ic exit` under a scratch ZDOTDIR, warm dump: main `.zshrc` 44.4 and 45.5 ms; this branch 44.2 and 44.7 ms; this branch with `~/.zsh/completions` populated 43.9 ms. The daily full `compinit` costs 51.5 ms once.

PATH order: `print -l $path` under a clean `PATH=/usr/bin:/bin:/usr/sbin:/sbin` and an empty `HOMEBREW_PREFIX`, main `.zshrc` versus this branch: identical (73 entries, `/opt/homebrew/bin` at 47 in both), so moving the brew block did not reorder PATH.

End to end: a shell with the written files reports `_comps[dotf]=1`, `_comps[herdr]=1`, and `~/.zsh/completions` first on `fpath`.

### PR 2 (AC6)

- [x] AC6 -> `go test ./internal/tools` (parse rejects a version with no digest and `latest`; render pins URL and sha256; install, second sync `changed=0`; a deleted file is repaired with `mise install --force`; an older version on disk is missing; the pinned plugin is not an orphan and an unpinned one is; `BinaryPins`); `bats tests/zshrc-guards.bats -f plugin` (4 tests); `bats tests/versions-conf.bats` (premature mark fails: mutation check).
- Measured with mise 2026.10.3 in a scratch `MISE_DATA_DIR`/`MISE_CONFIG_DIR`: `mise ls --json` lists the plugin as `http:zsh-autosuggestions` (the short name is not in the registry, so the orphan check maps it back); a wrong `checksum` is rejected by mise; `mise install` treats an install directory without the file as installed, so only `--force` repairs it; `mise where http:zsh-autosuggestions` answers the install dir.
- Live, built `dotf tools sync` against a `versions.conf` marking the plugin: first run `installed: zsh-autosuggestions`, second `nothing to do`; after deleting the file the next run repaired it and the one after reported nothing to do.
- Latency (medians of 20 `zsh -ic exit`, two rounds): main 44.6 / 44.5 ms; branch without the plugin 43.6 / 44.0 ms; branch with the plugin sourced 45.0 / 44.5 ms.
- The rc builds the plugin path with the precedence `MiseShimsDir` uses (`MISE_DATA_DIR`, then `XDG_DATA_HOME/mise`, then `~/.local/share/mise`); on this Mac `mise where age` is under `~/.local/share/mise/installs`, so the two agree. `tests/zshrc-guards.bats` 17/17 also in `ubuntu:24.04` as a non-root user.
- Live prune (mise 2026.10.3, scratch dirs): with the plugin no longer in the config, `mise ls --prunable --json` lists `http:zsh-autosuggestions` and `mise prune --yes --tools http:zsh-autosuggestions` removed `installs/http-zsh-autosuggestions` and its cache; this is the spec `orphanTools` builds. The sync verifies `<root>/latest/<file>` too, the path `.zshrc` sources: with `latest` deleted the next sync repaired it, the one after reported nothing to do.
- Mark, 2026-10-17 on the macmini with the released dotf 0.73.0 (the first release carrying `fileTools`): `dotf tools sync --dry-run --versions <branch>/versions.conf` listed `zsh-autosuggestions 0.7.1` and `to install: zsh-autosuggestions`; the real sync printed `installed: zsh-autosuggestions` and `39 tool(s) at their pin` (exit 0); `installs/http-zsh-autosuggestions/latest/zsh-autosuggestions.zsh` exists; `zsh -ic` reports `$+functions[_zsh_autosuggest_start]` = 1; a second sync printed `nothing to do`.
- Not measured: the prune through the converge itself (`dotf tools sync` does not set `Prune`), a Linux host (the table and the rc use no macOS-only call), Windows (compiled and vetted with `GOOS=windows`, the skip is by `runtime.GOOS`), the suggestions appearing in a live terminal.

## Test status

- `bats tests/zshrc-guards.bats` 13/13 on macOS and in `ubuntu:24.04` as a non-root user with a group-writable `/usr/local/share/zsh/site-functions` (the CI runner shape); `bats tests/guard-zsh-*.bats` green.
- `cd cli && go build ./... && go vet ./... && go test ./internal/converge ./internal/cmd`; `GOOS=windows go vet` on both; `golangci-lint run` (pinned 2.12.2) 0 issues.
- Not measured: a real Linux host (the container run above stands in); the step and the rc use no macOS-only call, and the step is `linux, darwin`.

## Decisions made during implementation

- The step compares content, not mtime, and runs each tool's own `completion zsh`; a stdout-only runner (`ExecStdout`) keeps a tool's stderr warning out of the file.
- The rc, not the step, owns dump freshness: a completion newer than the dump forces a full `compinit`, so the step needs no knowledge of `~/.zcompdump` or `ZDOTDIR`.
- The full `compinit` runs with `-u`: stock Debian/Ubuntu ship a group-writable site-functions, and a plain run aborts non-interactively ("initialization aborted"), which failed CI on the first push. `-C` already trusted every file, so `-u` keeps the previous trust level.
- Linuxbrew prefix left out of the brew block (out of scope, changes Linux PATH order).

## Promotion candidates

- [x] Lesson for the repo's `docs/lessons/`? no: the 24 h rule is the documented upstream pattern; nothing here was surprising.
- [x] ADR-worthy decision for the repo's `docs/adr/adr-XXX.md`? no: no alternative with lasting cost was chosen.
- [x] New pattern candidate for `00_meta/patterns/`? no: single project.

## Review dispositions

Review `review.md` (nan/glm5.3-flash, PASS-WITH-GAPS on `9b9e99f0`):

- Finding 1, `ExecStdout` stderr exclusion untested: ticketed as #2686, root cause and fix there. The claim under *Decisions* holds by code read only until that test lands.
- Finding 2, the dump rule governs only the non-oh-my-zsh branch: declined. No machine this repo provisions runs oh-my-zsh. On an omz machine its own policy applies, and completions are not lost, only deferred to omz's next full run.
- Finding 3, `compinit -u` on every full rebuild: declined, already recorded under *Decisions*. `-C` trusted every file before, so the trust level is unchanged. Revisit together with the completion-directory policy if it ever changes.
- Question, review range carries other specs' merged work: no action. The launcher resolves the base from the spec's first commit, and the reviewer confirmed it found no cross-feature interaction.

## Archive checklist

- [ ] `proposal.md` frontmatter set to `status: archived`
- [ ] Folder moved: `specs/DX-005-zsh-completions/` -> `specs/archive/DX-005-zsh-completions/`
- [ ] Bitácora board ticket for this spec moved to Done / closed with PR link (ADR-018)
- [ ] Promotions above executed (if any)
