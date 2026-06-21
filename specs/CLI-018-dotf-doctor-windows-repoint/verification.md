---
tags: [spec, verification]
created: "2026-06-21"
---

# Verification - CLI-018-dotf-doctor-windows-repoint

## Evidence (PR-A)

- [x] **Orca hook check behaves** → `checks_orca.go` `checkOrcaHook`; `TestCheckOrcaHook` table covers skip-when-absent, `timeoutSec` ≥30 PASS / <30 FAIL / any-too-low FAIL, `HttpWebRequest` PASS / `Invoke-WebRequest` FAIL, and both-files combined.
- [x] **Registered in the full sweep, not `--quick`** → `doctor.go` calls `checkOrcaHook` inside the `!opts.Quick` block after `checkAntigravity`.
- [x] **Cross-OS** → no `GOOS` gate; SKIPs when neither Orca file is present (the off-Windows / no-Orca state).

## Test status

- `go -C cli test ./internal/doctor/...` → **ok** (4.9s), incl. `TestCheckOrcaHook`.
- `go -C cli vet ./internal/doctor/...` → clean. `gofmt -l` clean for the 3 touched files.
- `go -C cli test ./...` → `internal/spec` + `internal/vault` `TestEmbeddedTemplatesMatchVault` **fail locally** — this is the **pre-existing #461 drift** (embedded templates vs vault SSOT), unrelated to this change (doctor-only) and a SKIP on CI (no vault). `internal/doctor` is green.

## Decisions made during implementation

- **Cross-OS, not `GOOS`-gated.** The Orca check skips on file-absence, which is equivalent to a Windows gate (the files only exist where Orca is installed) but simpler and faithful to the `.ps1` (which also skips when absent).
- **`timeoutSec` via regex, not full JSON parse** — mirrors the PowerShell check exactly (flag *any* hook timeout < 30), and avoids coupling to orca.json's nested schema.
- **§4 junction (BUG-012) not ported** — the `.ps1` itself marks it *secondary*, superseded by BUG-014 (already in `dotf doctor`'s `checkClaudeMem`); claude-mem-heal owns the repair. §4 deployed-file residual deferred to PR-B.

## Promotion candidates

- [ ] Lesson for `docs/lessons.md`? no — the cross-OS-parity lesson is already captured (CLI-020).
- [ ] ADR-worthy? no — executes ADR-020/021.
- [ ] New pattern? no.

## Archive checklist (after PR-B merges)

- [ ] `proposal.md` `status: archived`
- [ ] Folder → `specs/archive/CLI-018-dotf-doctor-windows-repoint/`
- [ ] Close #380 (PR-B link)
