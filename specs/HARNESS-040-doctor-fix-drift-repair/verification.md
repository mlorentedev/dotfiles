---
tags: [spec, verification, templates]
created: "2026-06-25"
---

# Verification - HARNESS-040-doctor-fix-drift-repair

## Evidence

Branch `feat/doctor-fix-drift-repair`. Tests run on a Windows host (the production target for the broken-junction case), so the Windows path branches are exercised natively, not just via the `GOOS` seam.

- [x] **Junction missing + source present → FAIL; `--fix` recreates → PASS; idempotent** → `TestCheckAutoMemoryLink/source_exists,_link_missing...` (FAIL) + `.../fix_links_it_then_a_re-run_is_idempotent` (FIX then PASS on re-run). Live: `dotf doctor` from the repo root now reports the section.
- [x] **No vault source → SKIP (both modes)** → `TestCheckAutoMemoryLink/no_vault_source_→_SKIP`. Live: running from `cli/` (a dir with no vault project) prints `[SKIP] … nothing to link`.
- [x] **Real non-empty dir → WARN, never destroyed by `--fix`** → `TestCheckAutoMemoryLink/real_non-empty_dir…` asserts `pathExists(ownFile)` after `--fix`. Live: `dotf doctor` from the repo root WARNs on this machine's real (diverged) auto-memory dir without touching its 4 files — the knowledge#120 case.
- [x] **OS-aware contract checks: `GOOS=windows`→windows dialect, `GOOS=linux`→unchanged** → `TestContractOS`, `TestCheckContractPath_Dialects` (linux + windows subtests), `TestCheckContractEnvVars_WindowsDialect`; existing `TestCheckContractEnvVars` (GOOS="") guards the linux regression. Live: PATH-entries section on this Windows machine went from 2 false `[WARN] …/.dotfiles/scripts not in PATH` to `(2 checks, all ok)` — the false session-start drift banner is gone.
- [x] **`cli go test ./...` passes for the touched packages; no unrelated changes** → doctor/mem/memlink all green; the 3 `TestEmbeddedTemplatesMatchVault` failures (initrepo/spec/vault) are pre-existing (reproduced with this branch stashed) and tracked by #461.

## Test status

- `go test ./internal/doctor/ ./internal/mem/ ./internal/memlink/` → **ok** (all three packages).
- `go build ./... && go vet ./...` → clean.
- Manual smoke: built `./cmd/dotf` and ran `dotf doctor` from both the repo root (WARN on the real diverged dir) and `cli/` (SKIP), plus confirmed the PATH-entries section is now all-ok on Windows.
- No regressions: the only failing tests in the module are the pre-existing template-drift checks (#461), confirmed by re-running them with this branch stashed.

## Decisions made during implementation

- **`memlink.Status` (read-only) added beside `Ensure`** so doctor can classify without the side effect of creating a link, mirroring `Ensure`'s exact decision order so the two never disagree.
- **OS encoding fixed at the root, in `memlink`**: `ClaudeProjectKey` now maps `/`, `\` AND `:` to `-` (Claude Code's real scheme), not just `/`. The retired shell-only `/` mapping was the latent cause of the wrong junction target on Windows. Consolidated as the single encoding shared by the session-start adapter and doctor (deleted the local `mem.encodeProjectPath`).
- **`isLink` now accepts `ModeIrregular`**: empirically, a `mklink /J` junction surfaces via `os.Lstat` as `ModeIrregular` (not `ModeSymlink`) on Go 1.26 — the existing `Ensure` no-op only worked by falling through to the `dirNotEmpty` branch.
- **`StateRealDir` is a WARN, never a destructive `--fix`**: when the auto-memory dir holds real (diverged) data, repair would mean data loss, which `memlink` refuses by contract. Honors the explicit "do NOT force-overwrite" guidance; the reconcile is manual.
- **Deferred, ticketed (per the fix-or-ticket rule):** #574 (Hive venv repair — cross-language) and #575 (`createLink` robustness for path components with bare cmd delimiters — surfaced by a comma in a `t.TempDir()` path; needs build-tagged Windows cmd-quoting).

## Promotion candidates

- [x] Lesson for the repo's `docs/lessons.md`? **yes** — "Windows junctions are `ModeIrregular`, not `ModeSymlink`, under Go 1.26; and Claude's project-key encoding maps `:`/`\` too, not just `/`." Both are non-obvious cross-OS gotchas likely to recur.
- [ ] ADR-worthy decision? no — consistent with ADR-021/ADR-025, no new architectural choice.
- [ ] New pattern for `00_meta/patterns/`? no — repo-specific, not cross-project.

## Archive checklist

- [ ] `proposal.md` frontmatter set to `status: archived`
- [ ] Folder moved: `specs/HARNESS-040-doctor-fix-drift-repair/` -> `specs/archive/HARNESS-040-doctor-fix-drift-repair/`
- [ ] Backlog entry ticked with PR link
- [ ] Promotions above executed (the `docs/lessons.md` entry)
