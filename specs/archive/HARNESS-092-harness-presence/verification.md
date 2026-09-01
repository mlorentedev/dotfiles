---
tags: [spec, verification]
created: "2026-08-28"
---

# Verification - HARNESS-092-harness-presence

## Evidence

Run on the Windows work box, 2026-08-28, worktree `dotfiles-wt-presence`, branch
`feat/harness-presence`, `dotf` built from the branch.

- [x] **AC1** → `TestBuildPresence_MatchesTheShellRendering` (exact bytes of the shell's header + bullets,
  autonomous skipped, `none` for a skill-less persona, sha equal for LF and CRLF input),
  `TestBuildPresence_RespectsTargetsAndSkipsAutonomous` (incl. `targets: [pi]` not matching `copilot` —
  the shell's substring match did), `TestBuildPresence_EmptyWhenNoPersonaTargetsTheHarness`,
  `TestLoadPresence_ManifestWithoutPresenceIsNoTargets`. Mutations: autonomous listed → red; sha over
  raw bytes → red.
- [x] **AC2** → `TestDeployPresence_InjectsEveryTargetOnceAndKeepsTheRest` (four files, two runs, one
  region, user prose and GENERATED region intact, region appended after the content),
  `TestInjectPresence_ReplacesAStaleRegionInPlace`, `TestInjectPresence_HonoursCRLF`,
  `TestDeployPresence_AbsentTargetIsASkipAndEmptyRosterInjectsNothing`;
  `TestHarnessPresenceCmd_ReportsEachTarget` (stdout/stderr split the setups read),
  `TestHarnessPresenceCmd_FailsOnABrokenRecord`. Mutation: append instead of replace → red.
- [x] **AC3** → `tests/compile-harness.bats`: "--deploy delegates presence to dotf harness presence with the
  checkout as --repo-root", "a failing dotf harness presence fails --deploy", "a dotf that predates harness
  presence makes --deploy fail loudly"; `tests/setup-windows.bats`: "injects agent presence via dotf
  harness presence after Deploy-SkillRecord" (+ parity: the Linux engine delegates and carries no injector).
  `bash -n scripts/compile-harness.sh` ok; `setup-windows.ps1` parse 0 errors, non-ASCII 32 = origin/main,
  CRLF 2357 / bare LF 0.
- [x] **AC4** → `TestCheckAgentPresence_ByStatus` (all injected → PASS "presence current in 2"; no region →
  WARN naming the file and `dotf harness presence`; roster changed after injection → WARN stale; copilot's
  file not compared without the binary, compared with it; no repo → SKIP). Mutation: stale counted as
  current → red.
- [x] **AC5** → box, in order:
  1. `grep -c 'BEGIN HARNESS AGENT-PRESENCE'` on `~/.claude/CLAUDE.md`, `~/.pi/agent/AGENTS.md`,
     `~/.config/opencode/AGENTS.md`, `~/.copilot/copilot-instructions.md` → **0, 0, 0, 0** (the #1326 measurement).
  2. `dotf harness presence --render claude` → the seven-persona roster (architect, builder, ...).
  3. `dotf harness presence` → `[deploy] presence -> <file> (<agent>)` × 4.
  4. `dotf harness presence` → `[deploy] presence current: <file> (<agent>)` × 4.
  5. Regions: 1, 1, 1, 1; every begin marker carries `sha256:5e0b469a4de5feb6`; each file kept its single
     line-ending style (LF, 0 CRLF).
  6. `dotf doctor` → `[Agent presence (forced skills)] (1 checks, all ok)`.

## Test status

```text
go build ./... && go vet ./... && GOOS=linux go vet ./... && go test ./internal/harness/ ./internal/cmd/ ./internal/doctor/   -> ok
golangci-lint run ./...   -> 0 issues
bats tests/setup-windows.bats   -> 122/122
bats tests/compile-harness.bats -f 'agents:|doctrine|presence|HARNESS-092'   -> 31/32 on the box; the one
  failure ("an absent dotf warns ... does not fail the deploy") fails identically on origin/main here:
  the test reduces PATH to /usr/bin:/bin and this box's jq lives under WinGet. Linux CI runs the full
  suite with /usr/bin/jq. (The full suite runs at ~1 test/min on this box; the subset is what was run.)
```

- No regressions in the existing suite: yes. The four presence scenario tests left
  `compile-harness.bats` and live in Go with the implementation. `shellcheck scripts/compile-harness.sh`
  → 0 warnings/errors (the two orphaned `AGENT_*` marker variables it flagged after the injector
  left were removed; the markers are Go's now).

## Decisions made during implementation

- **Delegate, do not duplicate.** The shell keeps `build_agent_presence` only as a wrapper over
  `--render` because `deploy_doctrine` composes the compact agy/codex payload from it; the
  injector is deleted. No awk fallback: a roster rendered by a second parser is the drift this
  port removes (the AC7/#1319 lesson), and an absent dotf is a loud `--deploy` failure.
- **Sha over the LF form.** The Windows checkout copies instructions files CRLF; a sha over raw
  bytes would make every Windows region read "stale" to a Linux doctor and vice versa.
- **Idempotent means untouched.** The shell rewrote the file on every run; the port compares the
  region and leaves an in-sync file alone, which is what makes "did this change?" answerable.
- **Whole-name `targets:` match.** `Persona.AppliesTo` already matched whole names; the shell's
  substring test (`[pi]` matched `copilot`) is a latent defect, not behaviour to preserve.
- **Doctor WARN, not FAIL.** The remedy is one idempotent command setup already runs.
- **The AGENT-PRESENCE markers have one spelling.** `harness.PresenceBeginPrefix/EndMarker` is the
  SSOT; doctor's mirror constants derive from it and `TestHarnessMarkerConstants` now pins that
  plus the shell's deletion of its own `AGENT_*` copy (it used to pin the shell's copy — caught
  after the rebase onto #1365/#1366).

## Promotion candidates

- [ ] Lesson: no — the ticket and the AC7 lesson (#1319) already carry the shape.
- [ ] ADR-worthy decision: no — first writing slice of CLI-026 under ADR-020 C7.
- [ ] Pattern: no.

## Archive checklist

- [ ] `dotf spec review HARNESS-092-harness-presence` PASS
- [ ] `proposal.md` frontmatter set to `status: archived`
- [ ] Folder moved to `specs/archive/HARNESS-092-harness-presence/`
- [ ] Bitácora #1326 closed with the PR link
