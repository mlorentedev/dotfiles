---
tags: [spec, verification, templates]
created: "2026-11-12"
id: "BUG-172-update-installer-converge-failure"
type: spec-verification
status: active
owner: manu
---

# Verification - BUG-172-update-installer-converge-failure

## Evidence

- [x] AC1: `TestRunPin_ConvergeFailsInsideInstaller` (`internal/update`), cases "installer's converge failed a step" and "two steps failed": status `converge-failed`, the message holds the running version, the pin, `installed` and `system-upgrade` (two steps joined as `system-upgrade, cache-gc`), and the error names the step. `TestFailedConvergeSince` (`internal/cmd`): a `last.json` with `result: failed` finished after the installer's start returns `[system-upgrade]`, as does one finished within the start second.
- [x] AC2: `TestRunPin_ConvergeFailsInsideInstaller` "no converge from the installer" keeps `install-failed` and "installing it failed"; the existing `TestRunPin` install-failure rows still pass with `InstallerConverge` unset. `TestFailedConvergeSince` returns nil for a report older than the installer, a passing report, no report and an unreadable one.
- [x] AC3: `TestFailedConvergeSince` "a failed run with no failed step names the run" returns `[the run (1 step failed)]`.

## Test status

- `go test ./internal/update/ ./internal/cmd/` ok. `go vet ./...` and `GOOS=windows go vet` clean. `golangci-lint run` 0 issues natively and with `GOOS=windows`.
- Mutation: comparing against the installer's start without truncating it to the second fails the same-second case.
- Command level: `TestUpdate_NamesTheStepWhenTheInstallersConvergeFails` runs `dotf update` against a fake installer that writes a failed converge report while it runs and exits 1; the output names the pin as installed and `system-upgrade`. Removing the `InstallerConverge: failedConvergeSince` binding fails it.
- Mutation, after the review: in `update.Run`, an unset start (`var start time.Time`) or one taken after `Install` returns fails `TestRunPin_ConvergeFailsInsideInstaller` (3 and 2 subtest failures, as the second review measured); removing the zero-time guard in `failedConvergeSince` fails "no installer start recorded proves nothing".

## Decisions made during implementation

- The proof that the pin installed is the converge report, not a re-read of the binary's version (option B in the issue): `install.sh` converges only after placing the binary, so a failed report newer than its start is its own, and it already names the steps. No installer contract change, and the failed step comes from a structured record instead of console text.
- The existing `converge-failed` tag, not a new one: it already means a deploy whose converge failed a step, and doctor reads it as such.

## Promotion candidates

- [x] Lesson for the repo's `docs/lessons/`? no: a misattributed error, recorded on #3395 and in this spec
- [x] ADR-worthy decision for the repo's `docs/adr/adr-XXX.md`? no: no new contract
- [x] New pattern candidate for `00_meta/patterns/`? no: "judge a report by when it finished relative to the run that should have written it" is specific to this seam

## Archive checklist

- [ ] `proposal.md` frontmatter set to `status: archived`
- [ ] Folder moved: `specs/BUG-172-update-installer-converge-failure/` -> `specs/archive/BUG-172-update-installer-converge-failure/`
- [ ] Bitácora board ticket for this spec moved to Done / closed with PR link (ADR-018)
- [x] Promotions above executed (if any)

## Review dispositions

The first review, by nan/qwen3.8-flash at `b911c61a`, gave FAIL.

1. Major, REAL (F1): the installer's start instant was taken in a closure in `cmd/update.go` that no test drove, and `failedConvergeSince` accepted any failed report against a zero instant. Applied, in a stronger form than suggested. `update.Run` now takes the instant itself, just before `Install`, and passes it to `InstallerConverge(since)`, so the order lives in the tested package. `TestRunPin_ConvergeFailsInsideInstaller` asserts the instant is non-zero and no later than the install, and `failedConvergeSince` returns nil for a zero instant (new row "no installer start recorded proves nothing"). The deleted-assignment and moved-assignment mutants both fail now.
2. Minor, THEORETICAL (F2): the literal `"failed"`. Applied: `converge.StatusFailed.String()`.
3. Minor, SPECULATIVE (F3): the same-second window. Declined, as the review itself concludes; the trade-off is in the proposal's Risks.
4. Minor, REAL (F4): `features.json` state and evidence. Left `pending` on purpose: only the harness may set `passing`, after running each `verification` command.
5. Minor, REAL (F5): no operator doc told the two installer outcomes apart. Applied in `docs/runbooks/guide-self-deploy-timer.md`, next to the installer branch.
6. Question (F6): the proposal's AC boxes stay unticked; `tasks.md` and this file are the record of acceptance, and the contract files are not edited after review.

The second review, by nan/mimo-v2.6-flash at `2c82e918`, gave PASS-WITH-GAPS.

1. Major, THEORETICAL: the command's binding of `failedConvergeSince` was untested, and deleting it left the suite green. Applied: `TestUpdate_NamesTheStepWhenTheInstallersConvergeFails`, with the fake installer writing its converge report as it runs. The unbound mutant now fails.
2. Minor, REAL: `update.Run` is 49 lines against the 40-line rule (43 before this change). Ticketed as #3427 rather than folded in here, so this fix stays the size the review read.
3. Minor, REAL: the mutant failure counts were wrong. Corrected above.
4. Questions (AC boxes, `features.json` state): answered in the first round's dispositions; nothing new.
