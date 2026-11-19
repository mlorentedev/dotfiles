---
tags: [spec, verification, templates]
created: "2026-11-19"
id: "HARNESS-201-force-with-lease-own-branch"
type: spec-verification
status: active
owner: manu
---

# Verification - HARNESS-201-force-with-lease-own-branch

## Evidence

- [x] AC1 (verdict table) -> commit b8473917 / `TestGuardCommand` (about 150 rows, red first: 104 failures against a stub), `TestGuardCommand_Heredocs`
- [x] AC2 (exit 0 or 2, never 1) -> commit 325354ba / `TestPushGuard_NeverExitsOne`, `TestPushGuard_AgainstARealRepository`
- [x] AC3 (real branch resolution, Windows paths) -> commit 325354ba / `TestPushGuard_DefaultBranchResolution`, `TestNativeDir`
- [x] AC4 (bound beside the gate) -> commit 325354ba / `TestPushGuardAndGateCoexistOnPreToolUse`, `TestBoundHarnessHooksResolveToRealSubcommands`
- [x] AC5 (deny and allow lists pinned) -> commit d7d0357c / `TestClaudeTemplate_*` in cli/internal/claude, bats test 32
- [x] AC6 (retire through the record) -> commit d7d0357c / `TestClaudeTemplate_*` in cli/internal/deploy
- [x] AC7 (ADR) -> commit cf10a10b / docs/adr/adr-065-agents-may-force-with-lease-their-own-branch.md

## Test status

- Mutation checks on the guard, all killed: drop main/master backstop, ignore abbreviations, ignore unknown dir, ignore checked-out branch, ignore +refspec, allow unknown lease options, accept short sha.
- `GOOS=windows go vet ./...`: exit 0. `bats tests/claude-settings-template.bats`: 33 ok.
- Known unrelated failures on this Windows box: symlink-creation tests (no privilege) in doctor and evals; gofmt reports on files this change does not touch.
- First review (agy, FAIL at 9b3504d8): one Blocker (refusal reason never reached stderr in the compiled binary) and four Majors (heredoc forms, wrapper options, subshell inside a quoted substitution, attached -C and -c). Each was reproduced as a red test first (`TestGuardCommand_ReviewBypasses`, and the cmd helper now reads stderr only), then fixed in 952d443c. The /c/ spelling now also runs against a real repository (`TestPushGuard_MsysDriveDirectoryAgainstARealRepository`).
- Second review (agy, FAIL at 71fa28e3): three Majors (heredoc spellings, wrapper by path and repository moves, eval/-Command/ANSI-C quotes), each red first in `TestGuardCommand_SecondRoundBypasses` and `TestGuardCommand_CdInsideASubshellDoesNotCarryOut`, then fixed. The ADR now says outright that this is a parser, not a sandbox.
- Review rounds 3, 4 and 5 (agy, FAIL at f7b05ee4, 4499a0d1 and 50a4ade6) each found three to six further Majors in the same families (child-shell cd, redirect and wrapper spellings, heredoc feeders, line continuation, PowerShell options). Each realistic one was red first and fixed (TestGuardCommand_ThirdRoundBypasses, FourthRoundBypasses, FifthRoundBypasses and the cd and option tests). The rounds do not converge: three heredoc-feeder spellings (feeder on a later line, a split shell name) stay open and are ticketed as #3611 with a proposed fail-closed text backstop. Closed by the #3611 backstop (`heredocFeedsAShell`, `TestGuardCommand_HeredocFeederBackstop`: 7 refused rows red first, 9 plain-push or no-force rows stay allowed); round 6 reviews it.
- Sixth review (agy, FAIL at 053b9f77, the first round after the #3611 backstop): five Majors. Two were the backstop's own (a force on a continuation line of the push; git's `--mir` prefix), three were new families: `timeout` and `nice` as wrappers, `env -S "git push" --force ...` (the words after the script are its arguments), a `cd` on the right of a pipe (a subshell), PowerShell backtick-newline, `Push-Location` and `iex`. Each was red first (`TestGuardCommand_SixthRoundBypasses`: 16 refused rows red, 4 plain rows stay allowed) and fixed in d77b27f7. The rounds still find new spellings each time; the ADR keeps the "parser, not a sandbox" limit.
- Final run at d77b27f7, go test -count=1 of internal/cmd, harness, deploy and claude: all ok (cmd 1073 s, harness 14 s, deploy, claude); `GOOS=windows go vet ./internal/harness` ok; `bats tests/claude-settings-template.bats` 33 ok (run at 053b9f77, no non-Go file changed since).
- Complexity: every new function is at 15 or below (was 17); the rest are above the ideal 10 and left as they are because splitting further hides the single switch each one is.
- Declined (theoretical): `git * push` glob variants for --mirror and main/master leases, and default-branch lookup for a URL remote. The hook is the control; the globs are the fallback while it is down, and main/master stay refused by name for a URL remote.
- TDD caveat: the cmd-layer file harness_pushguard.go was written before its tests; the guard table was red first.

## Decisions made during implementation

- Asymmetric parser: plain pushes pass; force-like pushes pass only in one literal shape; unparseable force-like input is refused.
- The hook only refuses; no allow rule for git push exists, so ordinary prompting still applies.
- Spec created with --over-wip-limit (11 active specs); the reason is in the proposal.

## Promotion candidates

- [x] Lesson for the repo's `docs/lessons/`? yes: docs/lessons/lesson-614-narrowing-a-deny-rule-in-a-merged-list-retires-nothing-without-a-record.md
- [x] ADR-worthy decision for the repo's `docs/adr/adr-XXX.md`? yes: docs/adr/adr-065-agents-may-force-with-lease-their-own-branch.md
- [x] New pattern candidate for `00_meta/patterns/`? no: one project so far, not recurring

## Archive checklist

- [ ] `proposal.md` frontmatter set to `status: archived`
- [ ] Folder moved: `specs/HARNESS-201-force-with-lease-own-branch/` -> `specs/archive/HARNESS-201-force-with-lease-own-branch/`
- [ ] Bitácora board ticket for this spec moved to Done / closed with PR link (ADR-018)
- [ ] Promotions above executed (if any)
