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

## Decisions made during implementation

- The proof that the pin installed is the converge report, not a re-read of the binary's version (option B in the issue): `install.sh` converges only after placing the binary, so a failed report newer than its start is its own, and it already names the steps. No installer contract change, and the failed step comes from a structured record instead of console text.
- The existing `converge-failed` tag, not a new one: it already means a deploy whose converge failed a step, and doctor reads it as such.

## Promotion candidates

- [x] Lesson for the repo's `docs/lessons/`? no: a misattributed error, recorded on #3395 and in this spec
- [x] ADR-worthy decision for the repo's `docs/adr/adr-XXX.md`? no: no new contract
- [x] New pattern candidate for `00_meta/patterns/`? no

## Archive checklist

- [ ] `proposal.md` frontmatter set to `status: archived`
- [ ] Folder moved: `specs/BUG-172-update-installer-converge-failure/` -> `specs/archive/BUG-172-update-installer-converge-failure/`
- [ ] Bitácora board ticket for this spec moved to Done / closed with PR link (ADR-018)
- [x] Promotions above executed (if any)

## Review dispositions
