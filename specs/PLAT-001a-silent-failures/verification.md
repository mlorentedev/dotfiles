---
tags: [spec, verification, templates]
created: "2026-10-05"
---

# Verification - PLAT-001a-silent-failures

## Evidence

### W1

- [x] AC1, AC2: `TestInstall_StagedBinaryMustExecuteAtThePin` (exec format error, no version, below the pin; Dest untouched; the probe ran the staged copy, not Dest).
- [x] AC3: `TestInstall_PackageManagerToolMustRunOnPathAfterward` (npm not on PATH, npm shadowed below the pin, uv-tool not on PATH).
- [x] AC4: the existing `TestInstall_*`, `TestInstallNpm_*`, `TestInstallUvTool*` and `TestRunToolsInstall_*` tests pass unchanged in their assertions. Only their seams now model a binary that executes.
- [x] End-to-end on darwin/arm64 (macOS 27.0.1), with real GitHub release assets and an isolated `HOME`:

  | Catalog `darwin` asset | dotf 0.64.0 (released) | this branch |
  |---|---|---|
  | `sops-v{version}.darwin.{goarch}` (correct) | installed, exit 0 | installed, exit 0, `sops 3.13.1` runs |
  | `sops-v{version}.linux.{goarch}` (the F-030 mistake) | **installed, exit 0**; places an ELF; `exec format error` on first use | `the downloaded sops-v3.13.1.linux.arm64 does not run on darwin/arm64`, exit 1, nothing placed |

## Test status

- `cd cli && go build ./... && go vet ./... && go test ./...`: all packages ok (darwin/arm64).
- `GOOS=windows go vet ./...` and `GOOS=linux go vet ./...`: ok.
- `golangci-lint run` (v2.12.2, the pin in `versions.conf`): 0 issues.
- No regressions in the existing Go suite: yes.
- Bats baseline on darwin, taken before this change: 1632/1776 pass. The 144 failures already exist on `main` and are recorded as F-054 in #2013. W1 changes no shell file.

## Decisions made during implementation

- The probe runs the **staged** copy (in the temp dir, under its command name) rather than the placed one. A failing binary therefore never shadows a working copy on PATH, even briefly.
- A version below the pin after install is an error on all three channels. For npm/uv it means another copy earlier on PATH answers, so the next command would not get the pin either.

## Promotion candidates

- [ ] Lesson for the repo's `docs/lessons/`? <decided at archive>
- [ ] ADR-worthy decision for the repo's `docs/adr/adr-XXX.md`? <decided at archive>
- [ ] New pattern candidate for `00_meta/patterns/`? <decided at archive>

## Archive checklist

- [ ] `proposal.md` frontmatter set to `status: archived`
- [ ] Folder moved: `specs/PLAT-001a-silent-failures/` -> `specs/archive/PLAT-001a-silent-failures/`
- [ ] Bitácora board ticket for this spec moved to Done / closed with PR link (ADR-018)
- [ ] Promotions above executed (if any)
