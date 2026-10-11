---
tags: [spec, verification, templates]
created: "2026-08-15"
---

# Verification - CLI-024-secrets-bw-serve

## Evidence

- [x] AC1 (unlock: hidden prompt, daemon start, POST /unlock, no leak) -> `TestSecretsUnlock_Succeeds_PasswordNeverInOutput`, `TestSecretsUnlock_WrongPassword_ErrorsWithoutLeakingIt` (`cli/internal/cmd/secrets_unlock_test.go`)
- [x] AC2 (serve-backed BWReader, selected automatically, under 2 s) -> `TestBWServeReader_Field_MatchesBWGetShape`, `TestSelectBWBackend_ReadAndWriteAlwaysAgree` (the unlocked case selects the daemon), `TestBWServeReader_ReadCostIsBoundedPerSecret` (the cost gate) and `BenchmarkBWServeReader_Field`; the live wall-clock is recorded below
- [x] AC3 (no-daemon fallback, zero consumer change) -> `TestSelectBWBackend_ReadAndWriteAlwaysAgree` (the locked, unauthenticated and unreachable cases fall back to the shellout on every half), full existing suite green unmodified
- [x] AC4 (doctor reports daemon state distinctly) -> `TestCheckBWServeDaemon_Absent/_Locked/_Unlocked/_StatusUnreadable` (`cli/internal/doctor/checks_bw_serve_test.go`)
- [x] AC5 (localhost-only bind) -> `TestBWServeCommand_BindsLocalhostOnly`
- [x] AC6 (`secrets lock`, idempotent unlock) -> `TestSecretsLock`, `TestSecretsLock_NoDaemon`, `TestSecretsUnlock_Idempotent`

## Test status

- Test suite: `cd cli && go build ./... && go vet ./... && go test ./... -count=1` -> all 13 packages `ok`, 0 failures
- Re-run 2026-10-10 for the archive: build, vet (also `GOOS=windows`) and `go test ./... -count=1` -> 36 packages `ok`, 0 failures; `golangci-lint run` after `golangci-lint cache clean` -> 0 issues.
- Lint: `golangci-lint run ./...` -> `0 issues`
- Manual smoke test: **run 2026-10-10, see "Live evidence" above.** The original instructions follow. The remaining `tasks.md` items (live daemon benchmark, live end-to-end `unlock`/`lock`, and now an unattended-path check — see below) require the operator's own terminal — the agent implementing this spec does not type, source, or hold the Bitwarden master password, by the same ADR-028 boundary this spec exists to enforce. Run:
  ```
  dotf secrets unlock            # types the real master password, interactively, yourself
  time dotf secrets verify       # compare against the 14-50s CLI-shellout baseline (OPS-021 spike, #675/#585)
  dotf secrets lock
  dotf secrets unlock            # again, confirms idempotent (should say "already unlocked" only if run before Start()'s daemon exits — otherwise re-prompts, which is also correct)
  ```
  then, **with the daemon left unlocked**, the scenario this PR actually exists to fix (#976: `pi`/`opencode`/`pollex` broken since #961 flipped `NAN_API_KEY` age→bw — every unattended `dotf secrets run` now dies on `Vault is locked`, and `pi` IS the adversarial-review primary reviewer, so a locked vault takes the review mechanism down too):
  ```
  tmux new -d -s bw-serve-smoke 'dotf secrets run -- pi --version > /tmp/bw-serve-smoke.log 2>&1; touch /tmp/bw-serve-smoke.done'
  # wait for the .done marker, then:
  cat /tmp/bw-serve-smoke.log     # must succeed with NO password prompt (no TTY in this session at all)
  ```
  A detached tmux pane has no TTY — exactly the shape that broke, and exactly where `bw unlock --raw`'s documented non-interactive stdin bug would have lived had we routed through the CLI shellout instead of the daemon. This is the acceptance evidence that actually matters for #976, not just the wall-clock number.
  Paste the wall-clock, the unattended-launch confirmation, and a confirmation the password never showed up in `ps`/history back into this file before archiving.
- No regressions in existing test suite: yes — full suite green before and after every commit in this branch

## Live evidence (2026-10-10, macmini, operator's unlocked session)

- **AC2 (wall-clock):** the operator ran `time dotf secrets verify` → `41 ok, 0 missing, 0 failed`, `1.361 total`, with 39 entries served by bw. The OPS-021 baseline was 14-50 s; the target was under 2 s.
- **AC2/AC3 (unattended, the #976 failure mode):** in a detached tmux session with no TTY, `dotf secrets run --only NAN_API_KEY -- pi --version` exited with `rc=0` and printed `1.1.0`, with no prompt.
- **AC5 (live bind):** the process listing of the running daemon is `bw serve --hostname 127.0.0.1 --port 8087`. It carries only the localhost bind, with no password and no session argument.
- **AC4 (live):** `dotf doctor` → `[bw serve daemon (optional local unlock cache)] (2 checks, all ok)`.
- **AC1/AC6 (live):** the operator ran `dotf secrets lock && dotf secrets unlock && dotf secrets unlock`, which printed:
  - `locked (pid 2313 …)`;
  - one hidden prompt, then `unlocked, vault cache synced at 2026-10-10T23:42:05Z`;
  - `already unlocked, vault cache synced at …23:42:10Z`, on the same pid with no second prompt.

  No leak was found:
  - The shell history holds `dotf secrets unlock` only, with no argument.
  - The daemon's process arguments carry only the localhost bind.
  - `~/.dotfiles/state/bw-serve.log` is mode 0600, 3 lines, and contains 0 matches for `password|session|BW_`. This was counted, not printed.
- The `/list/object/items` envelope, inferred at implementation time, is confirmed by the 41 live resolutions above.

## Decisions made during implementation

- Narrowed scope mid-spec (before any code): `BWWriter`/`BWCreator`/`BWFolderResolver` (the `set`/`migrate`/`render` write path) stay out of this PR — read-path only. The measured pain and the AC7-blocking friction are 100% read-path; the write path is invoked far less often and stays on the proven CLI shellout. Recorded in `proposal.md`'s Out of scope as a deliberate fast-follow, not a cut corner.
- The fallback selection (AC2/AC3) landed as a new `BWFallbackReader` type wired at `cmd/secrets.go`'s `bwReader` package var, not inside `resolve.go`'s `resolvers()` map as tasks.md originally sketched. `resolvers()` already delegates to whatever `BWReader` `Loader.BW` holds, so the dispatch table needed no change — a smaller diff for the same acceptance criteria.
- The `/list/object/items` envelope shape (used by `BWServeReader.getItemJSON`) is **inferred**, not empirically verified — only `/status`, `/unlock`, `/lock` were probed live in the OPS-021 spike (no unlocked vault was available for a real item search). Flagged in a code comment at the call site; the live verification task above is what confirms or corrects it.
- Lock policy for the MVP: no automatic re-lock (explicit `dotf secrets lock` or reboot/logout only) — decided via AskUserQuestion during `/spec fill`, trading timeout-logic complexity for simplicity, matching the actual pain (staying unlocked across sessions).

## Promotion candidates

- [ ] Lesson for the repo's `docs/lessons.md`? no: the squash-rebuild / rate-limit-contention lessons from this session belong to the *other* spec (CLI-024-secrets-file-migrate) already archived; nothing new and non-obvious surfaced here yet (pending the live task above, which might change this).
- [ ] ADR-worthy decision for the repo's `docs/adr/adr-XXX.md`? no: implementation detail within ADR-028's existing scope; the OPS-021 spike decision itself was recorded on issue #585, not promoted to an ADR (a CLI backend choice, not an architectural one).
- [ ] New pattern candidate for `00_meta/patterns/`? no: single-repo, single-occurrence.

## Archive checklist

- [x] `proposal.md` frontmatter set to `status: archived`
- [x] Folder moved: `specs/CLI-024-secrets-bw-serve/` -> `specs/archive/CLI-024-secrets-bw-serve/`
- [ ] Bitácora board ticket for this spec moved to Done / closed with PR link (ADR-018)
- [x] Promotions above executed (if any)

## Archive review round 1 dispositions (2026-10-10)

Round 1 (`nan/deepseek-v4-flash`, FAIL, reviewed `09518785`) is committed unchanged in its own commit. Dispositions:

| Finding | Disposition |
|---------|-------------|
| Major: AC2 asks for a benchmark test and none exists; f2 cannot fail on the speed claim | Applied. `TestBWServeReader_ReadCostIsBoundedPerSecret` gates what keeps the read path under 2 s (2 requests per read, no forced sync, no backoff over 41 entries); a sync per read turns it red (123 requests, want 82). `BenchmarkBWServeReader_Field` measures the per-read cost. f2 now runs the gate. A wall-clock assertion against a local fake was rejected: it measures none of the baseline's cost, which is a `bw` process per read. |
| Major: f2 `passing` while its evidence says the claim is unverified | Applied. f2's evidence now carries the gate, the benchmark and the live timing. |
| Major: `tasks.md` and this file cite the removed `BWFallbackReader` and three deleted tests | Applied. Both name `SelectBWBackend` (#1611, BUG-084) and the tests that cover AC2/AC3 today. |
| Minor: f4 evidence names three test functions that do not exist | Applied. It names `TestCheckBWServeDaemon_States` and `_StatusUnreadable`. |
| Minor (speculative): the unlocked daemon's API is unauthenticated on localhost | No change. Pre-existing and stated in the proposal; the localhost bind is the bound, and its test is mutation-proven. |
| Not verified: `golangci-lint` clean | Applied. Re-run after a cache clean: 0 issues. |

## Archive review round 2 dispositions (2026-10-10)

Round 2 (`agy/gemini-3.1-pro-high`, PASS, reviewed `eb7b8644`) is committed unchanged with this PR. Its three Minor findings:

| Finding | Disposition |
|---------|-------------|
| The unlocked daemon's API is unauthenticated on localhost (speculative, pre-existing) | No change, as in round 1: stated in the proposal and bounded by the mutation-proven localhost bind. |
| `proposal.md` keeps its AC boxes unticked | No change. The proposal is contract set and the reviewer advises against editing it; `tasks.md` and this file record each AC as met. |
| Live evidence cannot be reproduced without the master password | No change. That is ADR-028's operator boundary; the live run is recorded above, and the unit gates cover what an agent can run. |

