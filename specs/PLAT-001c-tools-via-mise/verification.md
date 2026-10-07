---
tags: [spec, verification, templates]
created: "2026-10-06"
---

# Verification - PLAT-001c-tools-via-mise

## Evidence

Map every acceptance criterion from `proposal.md` to concrete proof (commit hash, test name, or observed behavior).

- [x] AC1 (goos/goarch key wins) -> T1a / tests `TestAssetName_PrefersGoosGoarchOverGoos`, `TestValidKey`, `TestLoad_RejectsAnUnknownAssetKey`
- [x] AC2 (`./name`, `*name` checksum lines) -> T1a / test `TestExpectedChecksum_AcceptsSha256sumNameForms`
- [x] AC3 (no asset is a skip, as the plan says) -> T1a / tests `TestInstall_NoAssetForThisPlatformIsSkippedNotFailed`, `TestPlanAndInstallAgreeOnAPlatformWithNoAsset`
- [ ] AC4 -> T1b (after the release carrying T1a is the `DOTF_VERSION` pin)
- [ ] AC5, AC6 -> T2

## Test status

- Test suite (T1a, macOS arm64): `go build ./... && go vet ./... && GOOS=windows go vet ./... && go test ./... -count=1` -> ok; `golangci-lint run` (2.12.2) -> 0 issues; features f1 to f3 -> PASS
- Manual smoke test (T1a): `go run ./cmd/dotf tools list` loads the repo catalog under the new key validation
- No regressions: yes. `TestInstall_UnsupportedOS` asserted the opposite of AC3 (an error for a missing asset) and is replaced by the AC3 test

## Decisions made during implementation

Brief log of non-obvious trade-offs or course corrections taken during the work. Routine choices belong in commit messages, not here.

- T1 split into T1a (reader) and T1b (mise entry). An installed `dotf` 0.64 reads a GOOS/GOARCH-keyed entry as "no asset", and before T1a it returned an error for that, so `dotf tools install` would exit 1 on every machine until the pin moves (#1814 class).
- `docs/runbooks/tool-installation.md` was rewritten. It listed manual install commands, the debt AUDIT-009 flagged, and now documents the channels and how to add a catalog entry.

## Promotion candidates

Answer each line `yes: <path>`, naming the file you promoted, or `no: <reason>`. `dotf spec archive` refuses a line left unanswered, a `no` without a reason, and a `yes` whose file does not exist; a `00_meta/` path is looked up in the vault.

- [ ] Lesson for the repo's `docs/lessons/`? <yes: path / no: reason>
- [ ] ADR-worthy decision for the repo's `docs/adr/adr-XXX.md`? <yes: path / no: reason>
- [ ] New pattern candidate for `00_meta/patterns/`? Only if this recurs in >1 project. <yes: path / no: reason>

## Archive checklist

- [ ] `proposal.md` frontmatter set to `status: archived`
- [ ] Folder moved: `specs/PLAT-001c-tools-via-mise/` -> `specs/archive/PLAT-001c-tools-via-mise/`
- [ ] Bitácora board ticket for this spec moved to Done / closed with PR link (ADR-018)
- [ ] Promotions above executed (if any)
