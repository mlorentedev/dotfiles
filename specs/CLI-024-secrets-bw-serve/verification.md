---
tags: [spec, verification, templates]
created: "2026-08-15"
---

# Verification - CLI-024-secrets-bw-serve

## Evidence

- [x] AC1 (unlock: hidden prompt, daemon start, POST /unlock, no leak) -> `TestSecretsUnlock_Succeeds_PasswordNeverInOutput`, `TestSecretsUnlock_WrongPassword_ErrorsWithoutLeakingIt` (`cli/internal/cmd/secrets_unlock_test.go`)
- [x] AC2 (serve-backed BWReader, selected automatically) -> `TestBWServeReader_Field_MatchesBWGetShape`, `TestBWFallbackReader_UsesServe_WhenUnlocked` (unit-verified; the <2s live wall-clock claim is the one open item below)
- [x] AC3 (no-daemon fallback, zero consumer change) -> `TestBWFallbackReader_FallsBackToShellout_WhenLocked`, `TestBWFallbackReader_FallsBackToShellout_WhenAbsent`, full existing suite green unmodified
- [x] AC4 (doctor reports daemon state distinctly) -> `TestCheckBWServeDaemon_Absent/_Locked/_Unlocked/_StatusUnreadable` (`cli/internal/doctor/checks_bw_serve_test.go`)
- [x] AC5 (localhost-only bind) -> `TestBWServeCommand_BindsLocalhostOnly`
- [x] AC6 (`secrets lock`, idempotent unlock) -> `TestSecretsLock`, `TestSecretsLock_NoDaemon`, `TestSecretsUnlock_Idempotent`

## Test status

- Test suite: `cd cli && go build ./... && go vet ./... && go test ./... -count=1` -> all 13 packages `ok`, 0 failures
- Lint: `golangci-lint run ./...` -> `0 issues`
- Manual smoke test: **not yet run.** The two remaining `tasks.md` items (live daemon benchmark against a real `dotf secrets verify`, live end-to-end `unlock`/`lock` with the operator's own master password) require the operator's own terminal — the agent implementing this spec does not type, source, or hold the Bitwarden master password, by the same ADR-028 boundary this spec exists to enforce. Run:
  ```
  dotf secrets unlock            # types the real master password, interactively, yourself
  time dotf secrets verify       # compare against the 14-50s CLI-shellout baseline (OPS-021 spike, #675/#585)
  dotf secrets lock
  dotf secrets unlock            # again, confirms idempotent (should say "already unlocked" only if run before Start()'s daemon exits — otherwise re-prompts, which is also correct)
  ```
  then paste the wall-clock and a confirmation the password never showed up in `ps`/history back into this file before archiving.
- No regressions in existing test suite: yes — full suite green before and after every commit in this branch

## Decisions made during implementation

- Narrowed scope mid-spec (before any code): `BWWriter`/`BWCreator`/`BWFolderResolver` (the `set`/`migrate`/`render` write path) stay out of this PR — read-path only. The measured pain and the AC7-blocking friction are 100% read-path; the write path is invoked far less often and stays on the proven CLI shellout. Recorded in `proposal.md`'s Out of scope as a deliberate fast-follow, not a cut corner.
- The fallback selection (AC2/AC3) landed as a new `BWFallbackReader` type wired at `cmd/secrets.go`'s `bwReader` package var, not inside `resolve.go`'s `resolvers()` map as tasks.md originally sketched. `resolvers()` already delegates to whatever `BWReader` `Loader.BW` holds, so the dispatch table needed no change — a smaller diff for the same acceptance criteria.
- The `/list/object/items` envelope shape (used by `BWServeReader.getItemJSON`) is **inferred**, not empirically verified — only `/status`, `/unlock`, `/lock` were probed live in the OPS-021 spike (no unlocked vault was available for a real item search). Flagged in a code comment at the call site; the live verification task above is what confirms or corrects it.
- Lock policy for the MVP: no automatic re-lock (explicit `dotf secrets lock` or reboot/logout only) — decided via AskUserQuestion during `/spec fill`, trading timeout-logic complexity for simplicity, matching the actual pain (staying unlocked across sessions).

## Promotion candidates

- [ ] Lesson for the repo's `docs/lessons.md`? No — the squash-rebuild / rate-limit-contention lessons from this session belong to the *other* spec (CLI-024-secrets-file-migrate) already archived; nothing new and non-obvious surfaced here yet (pending the live task above, which might change this).
- [ ] ADR-worthy decision for the repo's `docs/adr/adr-XXX.md`? No — implementation detail within ADR-028's existing scope; the OPS-021 spike decision itself was recorded on issue #585, not promoted to an ADR (a CLI backend choice, not an architectural one).
- [ ] New pattern candidate for `00_meta/patterns/`? No — single-repo, single-occurrence.

## Archive checklist

- [ ] `proposal.md` frontmatter set to `status: archived`
- [ ] Folder moved: `specs/CLI-024-secrets-bw-serve/` -> `specs/archive/CLI-024-secrets-bw-serve/`
- [ ] Bitácora board ticket for this spec moved to Done / closed with PR link (ADR-018)
- [ ] Promotions above executed (if any)
