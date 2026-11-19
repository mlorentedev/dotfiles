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
