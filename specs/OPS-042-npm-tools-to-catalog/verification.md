---
tags: [spec, verification, templates]
created: "2026-08-29"
---

# Verification - OPS-042-npm-tools-to-catalog

## Evidence

Map every acceptance criterion from `proposal.md` to concrete proof (commit hash, test name, or observed behavior).

- [x] AC1 (two catalog entries; `dotf tools list` shows both) -> commit `a2c8d82` / `tests/packages-json.bats` (shape), box transcript below
- [x] AC2 (no npm blocks, no PS parser; scripts parse) -> `tests/setup-linux.bats` "parity: obsidian and yarn are catalog tools…", `tests/setup-windows.bats` "…through dotf tools install, not npm blocks"; `bash -n`, PSScriptAnalyzer 0 findings
- [x] AC3 (pins gone from `versions.conf`; doctor's yarn row reads `packages.json`) -> `TestCheckVersionMatch` (catalog fixture), box transcript below
- [x] AC4 (box) -> transcript below, Windows work box, 2026-08-29

## Test status

- Test suite: `cd cli && go test ./... -count=1` -> every package `ok`, `FAIL_COUNT=0`; `go vet` clean under `GOOS=windows` and `GOOS=linux`; `golangci-lint run` (pinned 2.12.2) `0 issues`
- `bats tests/setup-linux.bats tests/setup-windows.bats tests/packages-json.bats -f 'OPS-042|packages.json'` -> 9/9; `bash -n setup-linux.sh` clean; `Invoke-ScriptAnalyzer setup-windows.ps1` -> 0 findings
- Manual smoke test (AC4), binary built from this branch, `DOTFILES_DIR` and `DOTFILES_REPO_DIR` at the worktree so the catalog is this branch's:

  ```text
  --- dotf tools list ---
  copilot   1.0.81    full     npm:@github/copilot
  obsidian  0.5.1     full     npm:obsidian-cli
  yarn      1.22.22   full     npm:yarn
  --- dotf tools install ---
  obsidian 0.5.1 already installed; skipping
  yarn 1.22.22 already installed; skipping
  --- dotf doctor --verbose ---
  [ OK ] yarn version matches packages.json (1.22.22)
  ```

  The box's deploy mirror (`~/.dotfiles/packages.json`) predates #1359 and lists neither
  copilot nor these two, which is how the first AC4 attempt printed nothing: `dotf tools`
  reads the mirror only, while doctor reads the checkout first (see decisions).
- No regressions in existing test suite: yes

## Decisions made during implementation

- **`obsidian`, not `obsidian-cli`, is the catalog name.** The installer probes `<name> --version` on PATH, and the binary the package installs is `obsidian` (`obsidian --version` → `obsidian-cli v0.5.1`, which the semver probe parses). The package field carries `obsidian-cli`.
- **`OBSIDIAN_VERSION` was a pin that pinned nothing.** `1.12.4` is the Obsidian app's version, the npm package is at 0.5.1, and `git grep` found no consumer. Removed rather than corrected.
- **The doctor's yarn row joins copilot's and opencode's** — `matchPinFrom` against `catalogPin` — instead of keeping a `versions.conf` key the shell layer no longer reads (`checks_catalog.go`'s own rule).
- **`Test-VersionAtLeast` stays**: pi's block (#1294 follow-up) still calls it.
- **Found, not fixed here:** `dotf tools list|install` resolve `packages.json` from `DOTFILES_DIR` only (`cmd/tools.go:51,126`), while `doctor` resolves checkout-first per ADR-030 (`checks_catalog.go`). A box whose mirror lags the checkout gets two answers to "what is in the catalog". Ticketed as CLI-067 (#1381).

## Promotion candidates

Before archiving, flag what (if anything) should be promoted to the vault. If all three are "no", archive in repo is the only persistence.

- [ ] Lesson for the repo's `docs/lessons/`? no — the move is ADR-036 applied; the precedence inconsistency is a ticket, not a lesson yet
- [ ] ADR-worthy decision for the repo's `docs/adr/adr-XXX.md`? no
- [ ] New pattern candidate for `00_meta/patterns/`? no

## Archive checklist

- [ ] `proposal.md` frontmatter set to `status: archived`
- [ ] Folder moved: `specs/OPS-042-npm-tools-to-catalog/` -> `specs/archive/OPS-042-npm-tools-to-catalog/`
- [ ] Bitácora board ticket for this spec moved to Done / closed with PR link (ADR-018)
- [ ] Promotions above executed (if any)
