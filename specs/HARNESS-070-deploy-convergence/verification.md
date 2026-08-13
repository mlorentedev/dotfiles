---
tags: [spec, verification, templates]
created: "2026-08-12"
---

# Verification - HARNESS-070-deploy-convergence

## Evidence

- [x] AC1 -> `cli/internal/doctor/checks_harness_mirror_test.go`, test `TestCheckHarnessMirrorOrphans` (5 subtests, all pass)
- [x] AC2 -> `cli/internal/doctor/checks_test.go`, test `TestCheckOptionalTools_DotfDrift` (updated to assert FAIL)
- [x] AC3 -> manual: `HOME=/tmp/fake-home bash scripts/compile-harness.sh --deploy` updated all 4 instruction files; a second run was a byte-for-byte no-op (`diff -rq` clean); `--check` stayed green throughout
- [x] AC4 -> `cli/internal/doctor/checks_instruction_drift_test.go`, tests `TestStripHarnessRegions`, `TestCheckInstructionDrift`, `TestHarnessMarkerConstants` (all pass); confirmed live on this machine — correctly FAILs on the 4 files that are genuinely stale here right now
- [x] AC5 -> `cli/internal/doctor/checks_symlinks_test.go`, test `TestCheckDeployedSkillSymlinks` (5 subtests); confirmed live — the 4 previously-flagged Orca-managed symlinks (`computer-use`, `find-skills`, `orca-cli`, `orchestration`) no longer fail `dotf doctor`

## Test status

- `cd cli && go build ./... && go vet ./... && go test ./...` -> all packages pass (13 packages, 0 failures)
- `golangci-lint run ./...` (from `cli/`, pinned v2.12.2 matching `versions.conf`) -> 0 issues
- `shellcheck scripts/compile-harness.sh` -> clean; `bash -n` / `zsh -n` -> clean
- `bats tests/compile-harness.bats` -> 44/44 pass (unmodified — read-only verification, `tests/*.bats` out of scope this session)
- `bats tests/setup-linux.bats` -> 64/64 pass
- Manual smoke test: sandboxed `--deploy` run (`HOME=/tmp/fake-home`) exercised the new `deploy_instructions` step end-to-end, including the copilot `requires_command` gate and idempotency; real-machine `dotf doctor` run confirmed both the AC4 and AC5 checks fire correctly against actual current drift/no-drift state
- No regressions in existing test suite: confirmed (both bats files above, full Go suite)

## Decisions made during implementation

- Followed advisor guidance to gate the shell-side instruction-file copy at the TOP of `do_deploy` (before `deploy_skills`/`deploy_agents`), since those inject marked regions into the same 4 files and a copy after either would clobber them.
- Extended `agents.presence[]` with `source`/`requires_command` fields rather than adding a new manifest key — the 4 presence entries already named exactly the 4 target files.
- While sandbox-testing AC3, found and fixed a pre-existing bug in `deploy_agent_presence`: it always printed a `[deploy] presence -> ...` success line even when `inject_agent_presence` skipped because the target file was absent, producing a directly contradicting pair of log lines. Fixed by having `inject_agent_presence` return non-zero on skip and gating the success print on that.
- The "4 symlinked skills" evidence handed off as BUG-100-flavored turned out, on investigation, to be a false positive in `checkDeployedSkillSymlinks` (Orca's own `~/.agents/skills` mechanism, unrelated to this repo's `harness/skills/` records) rather than a regression of the historical issue #100. Filed as #943 (BUG-074) and fixed in the same change, per the documented precedent in `specs/archive/AI-022-pi-harness-slot/`.
- `checkHarnessMirrorOrphans` and `checkInstructionDrift` are gated to SKIP silently (not FAIL) when the repo checkout is unresolvable, matching `checkDeployDrift`'s existing convention — a doctor run on a machine with only the deploy mirror present (no checkout) is a legitimate, common case.
- Test isolation lesson: `resolveRepoDir`'s `os.Getwd()`/git-root fallback resolves to the REAL checkout during `go test` execution (this repo has `.git` above `cli/internal/doctor`), so a naive "unresolvable repo" test case is nondeterministic across environments — worked around by either avoiding the scenario (mirror-orphans test) or by skipping the test when the fallback actually resolves (instruction-drift test).
- A pre-PR advisor pass caught two real blockers before push: (1) `checkInstructionDrift` compared the copilot file unconditionally, which would have made it a permanent FAIL on any machine with a leftover file and no `copilot` binary — the exact "FAIL no remedy clears" shape #843 is about; fixed by gating on `requires_command`, same as the deploy side. (2) `deployedInstructionTargets`'s doc comment promised a manifest-sync test that didn't exist yet; written as `TestCheckInstructionDrift_MatchesManifest`. Also caught: `features.json` had self-set `"state": "passing"`, which the spec template reserves for the harness only — reset to `"pending"`.

## Promotion candidates

- [x] Lesson for the repo's `docs/lessons.md`? Yes — the `resolveRepoDir` test-isolation trap above; and the "verify an existing guard actually reproduces before trusting reported evidence" pattern (the symlink evidence was mischaracterized in the handoff).
- [ ] ADR-worthy decision for the repo's `docs/adr/adr-XXX.md`? No — this is additive convergence to the existing ADR-012/ADR-013 model, not a new architectural decision.
- [ ] New pattern candidate for `00_meta/patterns/`? No — repo-specific.

## Archive checklist

- [ ] `proposal.md` frontmatter set to `status: archived`
- [ ] Folder moved: `specs/HARNESS-070-deploy-convergence/` -> `specs/archive/HARNESS-070-deploy-convergence/`
- [ ] Bitácora board ticket for this spec moved to Done / closed with PR link (ADR-018)
- [ ] Promotions above executed (if any)

> Archive intentionally deferred to a follow-up session per this repo's `adversarial-review` gate (independent review before archive, ideally a different session/agent than the implementer).
