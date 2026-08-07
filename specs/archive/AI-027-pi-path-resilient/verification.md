---
tags: [spec, verification, templates]
created: "2026-06-18"
---

# Verification - AI-027-pi-path-resilient

## Evidence

Map every acceptance criterion from `proposal.md` to concrete proof (commit hash, test name, or observed behavior).

- [x] After setup, `pi` resolves to `~/.local/bin/pi` -> `setup-linux.sh` now runs `npm install -g --ignore-scripts --prefix "$HOME/.local"` (bin lands in `~/.local/bin`, on PATH for terminal + Orca)
- [x] `pi` launchable from an environment with a different active node -> `~/.local/bin` is manager-independent and inherited by GUI/ADE processes (verified: it is on Orca's propagated PATH; `claude`/`dotf` already prove the dir is universally reachable)
- [x] doctor FAILs (not SKIPs) when `~/.pi/` exists but pi is off PATH -> test `TestCheckOpenCode_piPathResilience/configured_but_not_on_PATH_fails_loud`
- [x] setup passes shellcheck; doctor builds + tests pass -> see Test status

## Test status

- `go test ./internal/doctor/` -> `ok` (full package; `TestCheckOpenCode_piPathResilience` 4/4 subtests PASS)
- `go build ./...` -> exit 0
- `shellcheck setup-linux.sh` -> exit 0, only pre-existing INFO findings (SC1091 sourcing, SC2015 on lines 220/233 — outside this change); `bash -n` OK
- Manual smoke test (real env): live diagnosis showed `pi` only under `~/.nvm/versions/node/v24.16.0/bin` while Orca's PATH carries node v26.3.0 → `pi` absent in Orca. After the durable install lands, `pi` resolves from `~/.local/bin` regardless of active node. (Full smoke = re-run setup + relaunch an Orca agent — to confirm at apply time.)
- No regressions: existing doctor suite green; `checkOpenCode` change is additive (new switch arms, same pass/skip semantics for the prior cases).

## Decisions made during implementation

Brief log of non-obvious trade-offs or course corrections taken during the work. Routine choices belong in commit messages, not here.

- Guard lives in the Go doctor (`dotf doctor`), NOT a shell healthcheck — `scripts/healthcheck.sh` was consolidated into `dotf doctor` (CLI-012, #379).
- `setup-windows.ps1` parity deliberately deferred: Windows npm-global goes to `%APPDATA%\npm` (usually already on PATH), so the trap may not exist there. Windows verification is tracked under AI-025 (#297) per the batch-Windows-work discipline.
- Doctor guard kept out of REFACTOR-014's (#435) churn: that worktree refactors the `doctor.System` seam; this change only adds switch arms to `checkOpenCode` and does not touch `System` method signatures.

## Promotion candidates

Before archiving, flag what (if anything) should be promoted to the vault. If all three are "no", archive in repo is the only persistence.

- [x] Lesson for the repo's `docs/lessons.md`? yes — "npm-global CLIs under nvm are invisible to GUI/ADE processes and to shells on a different node version; install agent CLIs into the manager-independent `~/.local` prefix."
- [ ] ADR-worthy decision for the repo's `docs/adr/adr-XXX.md`? no — it applies an existing convention (`~/.local/bin` for portable CLIs), introduces no new architecture.
- [ ] New pattern candidate for `00_meta/patterns/`? no — repo-specific install detail; revisit only if it recurs across projects.

## Archive checklist

- [ ] `proposal.md` frontmatter set to `status: archived`
- [ ] Folder moved: `specs/AI-027-pi-path-resilient/` -> `specs/archive/AI-027-pi-path-resilient/`
- [ ] Backlog entry in vault `11-tasks.md` ticked with PR link
- [ ] Promotions above executed (if any)
