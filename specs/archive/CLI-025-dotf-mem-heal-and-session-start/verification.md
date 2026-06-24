---
tags: [spec, verification]
created: "2026-06-22"
---

# Verification - CLI-025-dotf-mem-heal-and-session-start

> Strangler-fig spec, now COMPLETE. PR1 (`session-end`) + the session-start chain
> (PR2a core → memlink → PR2b-2a injectors → PR2b-2b adapter → PR3 cutover) all
> landed. The session hook cluster is one Go noun; every shell twin is deleted.

## Evidence (PR1 — session-end)

- [x] `dotf mem session-end` implemented in `cli/internal/mem`, cross-OS, table-tested
  -> `cli/internal/mem/session_end.go` + `session_end_test.go`
     (`TestSessionEnd_NoOps` 6 cases, `_HappyPath`, `_DefaultsSessionID`, `_EmptyVaultIsNoOp`).
     `cd cli && go test ./internal/mem/...` green.
- [x] The SessionEnd hook invokes `dotf mem session-end` directly (no shim script)
  -> `setup-linux.sh`: `EXPECTED_SESSION_END_COMMAND="$HOME/.local/bin/dotf mem session-end"`;
     `setup-windows.ps1`: `$expectedSessionEndCommand = "\"$dotfBin\" mem session-end"`.
     Stdin wiring + always-exit-0 covered by `cli/internal/cmd/mem_test.go`.
- [x] `session-handoff.{sh,ps1}` deleted + guard-grep pins no production caller
  -> `git rm scripts/session-handoff.{sh,ps1}` + the orphaned `tests/session-handoff.bats`;
     `tests/guard-no-session-handoff.bats` (3/3) pins absence + correct hook wiring.
- [x] No collision with `tests/claude-settings-template.bats`
  -> that suite tests the placeholder/merge *mechanism*, not the command value; 24/24 still pass.
- [x] Cross-spec hygiene: `specs/MEMORY-001-cross-agent-session-bridge/features.json`
  -> repointed its two dangling verifications (deleted bats + shellcheck) at the Go port;
     `behavior` corrected to `10_projects/<project>/sessions/` (the #542 location).
- [x] (PR2a→PR3) `session-start` port + byte-equivalent `additionalContext` — see below.

## Evidence (session-start — PR2a → PR3)

- [x] **PR2a (#554)** agnostic core `dotf mem session-start --format=stdout|markdown`
  -> `cli/internal/mem/session_start.go` (the `sb_*` emitters + render); byte-equivalence
     harness vs `session-brief.sh` green on Linux CI.
- [x] **memlink (#557)** OS-agnostic vault→memory link primitive (symlink POSIX / junction
  Windows) -> `cli/internal/memlink`; closes the MEMORY-002 R4 Windows gap; unblocks #551.
- [x] **PR2b-2a (#566)** Claude injectors (config reader, claude.json-size, knowledge-health,
  memory-temperature, doctor-drift, hive/work-SDK, auto-memory) -> `cli/internal/mem`, table-tested.
- [x] **PR2b-2b (#569)** adapter assembly + `additionalContext` envelope (jq-equivalent: ordered
  keys, no HTML escaping); **golden gate** vs the live `claude-session-start.sh` green on Linux CI.
- [x] **PR3 (this)** cutover: SessionStart hook repointed to `dotf mem session-start` in
  `setup-{linux.sh,windows.ps1}`; `git rm` of `claude-session-start.{sh,ps1}` + `session-brief.sh`
  + `ensure-memory-symlink.sh` (1156 LOC of shell) + 4 obsolete bats + the 2 migration gates
  (byte-equivalence tests retire with the shell they compared to); guard test pins no
  deploy/registration file invokes them; `tests/*.bats` 174/174 local, 0 fail.

## Test status

- Go: `cd cli && go build ./...` OK; `go vet ./...` OK; `go test ./internal/mem/ ./internal/cmd/` OK.
  Full `go test ./...` has 3 pre-existing `TestEmbeddedTemplatesMatchVault` FAILs
  (initrepo/spec/vault) = vault-template drift #461, unrelated to this change.
- Bats: `tests/guard-no-session-handoff.bats` 3/3; `tests/claude-settings-template.bats` 24/24.
- Manual smoke: deferred (interactive); the cmd test exercises the full stdin->resolve->write path.

## Decisions made during implementation

- **Direct invocation, not a shim (B over A).** The proposal originally said "thin shim
  that exec dotf mem". Reframed to direct hook invocation: a shim still ships a per-OS
  `.sh`/`.ps1` pair, miniaturizing the twin-drift the CLI convergence exists to kill.
  Hook `command` is now the absolute binary path + `mem session-end`; the only residual
  OS-variance (dotf on PATH) is owned by the env-contract/`dotf doctor` (ADR-025).
- **Absolute binary path, not bare `dotf`.** Robust when `~/.local/bin` is off the profile
  PATH (the live #531 drift); also faithful to the prior pattern (absolute script path).
- **Converged the twin drift.** The `.sh` used an em-dash heading, the `.ps1` a hyphen;
  the Go port emits one form (em-dash).
- **Vault resolver, not the hardcoded literal.** `vault.ResolveVault()` (ADR-025 cascade)
  replaces the `$HOME/Projects/knowledge` literal the shells hardcoded — retires it for
  this caller (#463).

## Promotion candidates

- [ ] Lesson for `docs/lessons.md`? yes — "a thin per-OS shim is still a twin; converge to
  direct CLI invocation and let the env-contract own the PATH invariant" (capture at spec archive).
- [ ] ADR-worthy? no — applies ADR-020/021/025, introduces nothing new.
- [ ] New pattern? no — instance of the existing CLI-convergence pattern.

## Archive checklist

> Do NOT archive yet — PR2/PR3 (session-start) are still open under this spec.

- [ ] `proposal.md` frontmatter set to `status: archived`
- [ ] Folder moved to `specs/archive/`
- [ ] Promotions above executed (if any)
