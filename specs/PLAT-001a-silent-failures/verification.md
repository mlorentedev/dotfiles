---
tags: [spec, verification, templates]
created: "2026-10-05"
---

# Verification - PLAT-001a-silent-failures

## Evidence

### W1

- [x] AC1, AC2: `TestInstall_StagedBinaryMustExecuteAtThePin` (exec format error, no version, below the pin; Dest untouched; the probe ran the staged copy, not Dest).
- [x] AC3: `TestInstall_PackageManagerToolMustRunOnPathAfterward` (npm not on PATH, npm shadowed below the pin, uv-tool not on PATH).
- [x] AC4: the existing `TestInstall_*`, `TestInstallNpm_*`, `TestInstallUvTool*` and `TestRunToolsInstall_*` tests pass unchanged in their assertions. Only their seams now model a binary that executes.
- [x] End-to-end on darwin/arm64 (macOS 27.0.1), with real GitHub release assets and an isolated `HOME`:

  | Catalog `darwin` asset | dotf 0.64.0 (released) | this branch |
  |---|---|---|
  | `sops-v{version}.darwin.{goarch}` (correct) | installed, exit 0 | installed, exit 0, `sops 3.13.1` runs |
  | `sops-v{version}.linux.{goarch}` (the F-030 mistake) | **installed, exit 0**; places an ELF; `exec format error` on first use | `the downloaded sops-v3.13.1.linux.arm64 does not run on darwin/arm64`, exit 1, nothing placed |

### W3

- [x] AC5: `tests/bash32-portable.bats`. It is red on `main` (6 sites across 4 scripts) and green on the branch. Its self-test checks that the pattern catches each construct and spares 3.2 code.
- [x] AC5: `/bin/bash scripts/compile-harness.sh --check` on macOS bash 3.2.57. `main` dies with `line 1439: mapfile: command not found`; the branch prints `[check] OK: no harness drift`.
- [x] AC6: the 17 affected bats files (the 15 that exercise compile-harness, plus the two ported eol and stub-pairing suites) pass 489/493 on darwin. The 4 left also fail on `main` and are outside W3. Two read this machine's real `~/.gemini` (F-008); two are the over-cap char/byte warning on darwin (F-054 triage).
- [x] Review round (pr-agent on #2015):
  - The guard now also scans `tests/*.bats` and `tests/*.bash`, since bats sources them with the bash on PATH, which is 3.2 on this Mac (`bash --version` measured). That found `local -A` in `gitattributes-eol.bats` and `stub-real-pairing.bats`, which failed on darwin with `local: -A: invalid option` (part of F-054). Both are ported and pass on 3.2. A mutation that adds an extensionless file with no eol rule is still caught.
  - The pattern also covers `declare -g`, `[[ -v ]]` and `${var@op}` (bash 4.2/4.4). The `;&` fallthrough only matches at the end of a case arm, so the bracket expression `[^|;&]` in `ci-age-pin.bats` is no longer a false positive. The guard skips its own file.
  - `nan-quality-bench.sh` no longer uses `${!arr[@]}`, which is a `bad substitution` in zsh. The names are now the positional parameters and the texts come from a `case`. bash 3.2 and zsh produce byte-identical output, and the evaluated texts equal `main`'s.
- [x] Ported scripts keep their behaviour:
  - `nan-quality-bench.sh`: the 4 prompts are identical to `main` and come out in the same order, under bash 3.2 and zsh.
  - `pin-actions.sh`: with a stub `gh` under bash 3.2, a repeated key makes no second API call and an unresolvable ref stays empty.

### Interim: setup's hard-coded linux-amd64 downloads (ahead of W2)

The first `install.sh` run on the Mac (2026-10-07, `dotf` 0.65.0) reproduced the first bullet of the proposal's problem list on a real machine. Setup logged `age installed (v1.3.1)` and `eza installed`, and left four ELF x86-64 files in `~/.local/bin`: `age`, `age-keygen`, `eza` and `shellcheck`. mise's age and shellcheck come earlier on an interactive PATH, so those two still worked. eza has no mise entry, and the unconditional `alias ls="eza ..."` made `ls` exit 126 in every new shell. W2 deletes the blocks once the tools come from mise on every OS, and a second setup run would have placed them again. So this interim fix lands first:

- [x] `setup-linux.sh` runs the age/eza/jq/gh and shellcheck blocks only when `host_is_linux_amd64` (`scripts/utils.sh`). Elsewhere it warns once and calls `remove_unrunnable_tool`, which deletes a `~/.local/bin` file only when running it exits 126 ("found but cannot execute"). Symlinks, tools that run and tools that fail for other reasons stay.
- [x] `age installed` is logged only after the placed `age --version` runs.
- [x] `.zsh/aliases.zsh` aliases `ls`/`ll`/`lla` to eza only when eza is on PATH. Otherwise `ll` and `lla` fall back to `ls -l` and `ls -la`.
- [x] `tests/setup-linux-only-downloads.bats`: 6 ok on the branch, 6 not ok on `main`. Every bats file referencing `setup-linux`, `utils.sh` or `aliases.zsh`: 950/950 on darwin. While writing it, a mid-test `[[ ]]` passed against the unfixed file under bash 3.2, which is #2164 (54 such assertions in the `os-sensitive` tier).
- W2 still owns the real fix. When it deletes the blocks, it deletes `host_is_linux_amd64`, the gate and this guard test with them.

## Test status

- `cd cli && go build ./... && go vet ./... && go test ./...`: all packages ok (darwin/arm64).
- `GOOS=windows go vet ./...` and `GOOS=linux go vet ./...`: ok.
- `golangci-lint run` (v2.12.2, the pin in `versions.conf`): 0 issues.
- No regressions in the existing Go suite: yes.
- Bats baseline on darwin, taken before this change: 1632/1776 pass. The 144 failures already exist on `main` and are recorded as F-054 in #2013. W1 changes no shell file.

## Decisions made during implementation

- The probe runs the **staged** copy (in the temp dir, under its command name) rather than the placed one. A failing binary therefore never shadows a working copy on PATH, even briefly.
- The staged copy lives in a hidden dir **inside Dest** (`.dotf-stage-*`), not the system temp dir. pr-agent review on #2014 pointed out that a `noexec` `/tmp` would refuse a binary that runs fine from Dest and blame the OS/arch. Staging on Dest's own mount fails exactly when the placed binary would, and the dir is removed on both success and refusal; the tests assert an empty or single-entry Dest.
- A version below the pin after install is an error on all three channels. For npm/uv it means another copy earlier on PATH answers, so the next command would not get the pin either.

## Promotion candidates

- [ ] Lesson for the repo's `docs/lessons/`? <decided at archive>
- [ ] ADR-worthy decision for the repo's `docs/adr/adr-XXX.md`? <decided at archive>
- [ ] New pattern candidate for `00_meta/patterns/`? <decided at archive>

## Archive checklist

- [ ] `proposal.md` frontmatter set to `status: archived`
- [ ] Folder moved: `specs/PLAT-001a-silent-failures/` -> `specs/archive/PLAT-001a-silent-failures/`
- [ ] Bitácora board ticket for this spec moved to Done / closed with PR link (ADR-018)
- [ ] Promotions above executed (if any)
