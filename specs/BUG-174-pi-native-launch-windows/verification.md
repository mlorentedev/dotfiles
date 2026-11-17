---
tags: [spec, verification, templates]
created: "2026-11-16"
id: "BUG-174-pi-native-launch-windows"
type: spec-verification
status: active
owner: manu
---

# Verification - BUG-174-pi-native-launch-windows

## Evidence

- [x] AC1 -> `TestNativeTarget_StartsTheExecutableBehindABatchShim`; on Windows `TestExec_StartsTheNativeExecutableBehindABatchShim` (the real `Exec`, a `.cmd` first on PATH, the prompt `a & b %PATH% ^ | < > "q"` and a second line arriving intact, the shim's marker file absent)
- [x] AC2 -> `TestNativeTarget_RefusesWithoutANativeExecutableBehindTheShim` (eight answers)
- [x] AC3 -> `TestNativeTarget_LeavesANativeResolutionAlone`
- [x] AC4 -> `TestNativeTarget_NeverSubstitutesANamedFile`
- [x] AC5 -> on Windows `TestRunChild_StartsTheNativeExecutableBehindABatchShim` (the real `runChild`; reverting its launch line to `argv[0]` fails it, checked 2026-11-16); `TestSecretsRun_StartsTheNativeExecutableBehindABatchShim`, with the existing `TestSecretsRun_RefusesABatchShimChildWhereWindowsRunsItThroughCmdExe` unchanged

## Test status

- Test suite: on Windows 11 x64, `go test ./internal/lane/ -count=1 -run 'NativeTarget|TestExec_|RefuseBatch'` and `go test ./internal/cmd/ -count=1 -run 'TestRunChild_StartsTheNative|Secrets'` pass. The whole of both packages is CI's evidence (`test (ubuntu-latest)`, `test (macos-latest)`, `test (windows-latest)` on the PR head), not this machine's: `go test ./internal/lane/ -count=1` ran past the default 10 minute limit here, as `./internal/cmd/` does (#3566); `GOOS=linux go vet` and `GOOS=darwin go vet` of both packages exit 0.
- Manual smoke test, this machine, 2026-11-16, a throwaway test calling `NativeTarget(name, exec.LookPath, MiseNative)`:
  - `pi`: PATH gives `~\.pi\agent\bin\pi.cmd`, the target is `%LOCALAPPDATA%\mise\installs\pi\1.5.2\pi.exe`.
  - `agy`, `claude`, `copilot`: native on PATH, unchanged.
  - `opencode`, `bats`: mise answers with an extensionless npm stub, so they stay refused.
- Not exercised: `dotf secrets run -- pi` end to end. It needs the unlocked secrets store, and `dotf spec review` starts the released `dotf` from PATH, which does not carry this change yet.
- No regressions in existing test suite: yes on CI's three test legs; not established locally (see above).
- Tests never ask this machine's mise: both packages default the lookup to "nothing", and a test that wants a native target sets its own (lesson 610).

## Decisions made during implementation

- Ask mise instead of changing what is installed. Native mise shims (`mise-shim.exe`) and a native pi ahead on PATH were the first plan; neither changes what `exec.LookPath("pi")` finds while `~/.pi/agent/bin` is ahead of mise's shims, and both belong to PLAT-001c.
- Only an `.exe` is accepted from mise. Checking the answer with `RefuseBatchShim` was not enough: an extensionless path has a `.cmd` beside it, and Windows appends PATHEXT.
- `secrets run` asks the tool manager twice per start, not once: the command checks the child before the store is read (`assertSafeChildCommand`), and `runChild` resolves again after, through `safeChildTarget`, which is both its check and its launch target. Inside `runChild` there is one resolution (`TestRunChild_StartsTheNativeExecutableBehindABatchShim` asserts it). The target is not passed from the early check to the launch through the callers.
- `dotf doctor` does not report a batch-only launcher: #3572.

## Promotion candidates

Answer each line `yes: <path>`, naming the file you promoted, or `no: <reason>`. `dotf spec archive` refuses a line left unanswered, a `no` without a reason, and a `yes` whose file does not exist; a `00_meta/` path is looked up in the vault.

- [x] Lesson for the repo's `docs/lessons/`? yes: docs/lessons/lesson-611-a-tool-managers-path-for-a-command-is-not-a-file-windows-will-start.md
- [x] ADR-worthy decision for the repo's `docs/adr/adr-XXX.md`? no: it narrows an existing refusal inside its own rule (start a native executable), and changes no contract
- [x] New pattern candidate for `00_meta/patterns/`? Only if this recurs in >1 project. no: specific to how dotf starts agents on Windows

## Archive checklist

- [ ] `proposal.md` frontmatter set to `status: archived`
- [ ] Folder moved: `specs/BUG-174-pi-native-launch-windows/` -> `specs/archive/BUG-174-pi-native-launch-windows/`
- [ ] Bitácora board ticket for this spec moved to Done / closed with PR link (ADR-018)
- [ ] Promotions above executed (if any)
