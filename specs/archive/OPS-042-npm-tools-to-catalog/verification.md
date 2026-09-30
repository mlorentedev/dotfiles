---
tags: [spec, verification, templates]
created: "2026-08-29"
---

# Verification - OPS-042-npm-tools-to-catalog

## Evidence

Map every acceptance criterion from `proposal.md` to concrete proof (commit hash, test name, or observed behavior).

- [x] AC1 (two catalog entries; `dotf tools list` shows both) -> commit `1478f24` (#1382; `a2c8d82` was the pre-squash branch commit, unreachable after the squash) / `tests/packages-json.bats` (shape), box transcript below
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

## Amendment, 2026-09-23 — the `obsidian` entry was the wrong package (#1615)

The npm package `obsidian-cli` 0.5.1 that AC1 declared is **not an Obsidian CLI**. It is a 2020 third-party tool that imports test results into Obsidianqa.com, and it registers a global `obsidian` binary. It got in through a 404 on `@vorillaz/obsidian-cli`, "fixed" in #122 by dropping the scope; lesson 065 recorded that as correct. The evidence above, `obsidian --version` → `obsidian-cli v0.5.1`, parsed as a semver and so looked like proof. It only proved that *a* package answered with a version, not that it was the right one.

The `obsidian` that `dotf vault health` and obs-cli drive is the CLI built into the Obsidian desktop app, which is not an npm tool. The entry is removed. AC1 and f1 now assert its absence, and `tests/setup-linux.bats` refutes any obsidian npm entry under either name.

## Review (PASS-WITH-GAPS, 2026-09-30) — dispositions

`review.md` (`nan/qwen3.8-flash`, reviewed `6d1393d`) passed with six Minor gaps. None is fixed in this archive. Editing the contract set or the tested surface after the verdict would leave the archive recording a review of something that changed. So each gap is dispositioned here:

- **Row 1, the yarn guard is quoting-sensitive (REAL): deferred to #1864 §1.** The guard should match the property, not the historical spelling.
- **Row 2, AC4 and f4 still name obsidian (REAL): declined for this spec.** AC1 and f1 were amended on 2026-09-23 to assert that obsidian is absent, and `tests/setup-linux.bats` refutes it under either name. Only AC4's wording and f4's alternation still carry the name. Editing them now would void this verdict. The absence is guarded by f1 and the bats test, so the stale wording can no longer let obsidian back in.
- **Row 3, `a2c8d82` does not resolve (REAL): applied.** The evidence line now cites `1478f24` (#1382).
- **Row 4, the yarn row is a floor, so yarn 4 passes (THEORETICAL): deferred to #1864 §2.**
- **Row 5, claude-code is still a raw `npm install -g` (REAL): deferred to #1864 §3.** OPS-042 moved the two tools it named. It does not claim ADR-036 is complete for npm tools, and #1864 records that it is not.
- **Row 6, f4 is box-only and its run predates the amendment (THEORETICAL): accepted as stated.** f4 is a MINGW box check by design. Its 2026-08-29 transcript still proves that yarn installs at its pin, and the amendment only removed obsidian, which f1 now covers on every OS.

## Promotion candidates

Before archiving, flag what (if anything) should be promoted to the vault. If all three are "no", archive in repo is the only persistence.

- [x] Lesson for the repo's `docs/lessons/`? no: the move is ADR-036 applied; the precedence inconsistency is a ticket, not a lesson yet
- [x] ADR-worthy decision for the repo's `docs/adr/adr-XXX.md`? no: ADR-036 already records the decision this spec applied
- [x] New pattern candidate for `00_meta/patterns/`? no: nothing here is cross-project

## Archive checklist

- [ ] `proposal.md` frontmatter set to `status: archived`
- [ ] Folder moved: `specs/OPS-042-npm-tools-to-catalog/` -> `specs/archive/OPS-042-npm-tools-to-catalog/`
- [ ] Bitácora board ticket for this spec moved to Done / closed with PR link (ADR-018)
- [ ] Promotions above executed (if any)
