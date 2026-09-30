---
spec: "AI-034-opencode-npm-channel"
verdict: "PASS"
reviewed_sha: "b776e9da2b18dce2587adce65ad5f18bf4218978"
reviewer: "nan/deepseek-v4-flash"
date: "2026-09-30"
---

## Adversarial review

**Scope**: AI-034-opencode-npm-channel — one install channel per tool class, opencode moved to `packages.json`, `dotf tools version` + the doctor shadowed-copy WARN.
**Sources**: `specs/AI-034-opencode-npm-channel/{proposal,tasks,verification}.md`, `features.json`, `git diff 8bab48aff71e259c6fe62116facc8c9d41f8965d...HEAD` (launcher-resolved base), the spec's own commit `016bf1a` (PR #1311), and the live tree at `b776e9d`.

> **Scope caveat, stated up front.** The launcher-resolved base predates the spec's commit: the range spans **262 commits / ~88.8k insertions across 1121 files**, most of them other specs (AI-038, OPS-042, CLI-0xx, SEC-0xx, …). AI-034's own commit is `016bf1a`, whose 23-file / 749-insertion diff matches `proposal.md` exactly. Every finding below is judged against the AI-034 surface as it stands at `b776e9d`; the large range is a launcher base-selection artifact, not scope creep in this change.

### Spec and task alignment

- All `tasks.md` implementation and closing boxes are `[x]`; no `[AGENT-DRAFT]` / `[AGENT-SUGGESTION]` tags remain in the spec folder.
- **AC1** — `packages.json` declares `opencode` as `npm`/`opencode-ai`/`1.16.2` (verified by reading the file) and `versions.conf` has no `OPENCODE_VERSION`. The spec's own commit removed **only** `OPENCODE_VERSION` from `versions.conf` (`git show 016bf1a -- versions.conf` → one line), so the other `versions.conf` churn in the range (OBSIDIAN/YARN/PI/GIT) is not scope creep from this spec. `features.json` f1 command: **exit 0** (re-run this session).
- **AC2** — no opencode install block, no `SST.opencode` winget entry, no winget version-pin machinery, and no `~/.opencode/bin` PATH export in `.zshrc`/`.bashrc`/`powershell/profile.ps1`. `grep -n "SST.opencode"` and `grep -n "opencode/bin"` over the setup scripts + profiles return only explanatory comments. The surviving `opencode` references in both setups are config deployment and a post-deploy *assertion* that names the retired `~/.opencode/bin` copy — the ADR-036 §5 migration, not a channel.
- **AC3** — `dotf tools version <name>` prints the first semver and exits 1 with empty stdout when there is none; both setups gate `hive service` on `dotf tools version hive` (setup-linux.sh:1065, setup-windows.ps1:746). Verified end-to-end with a binary built from this tree against a fake banner tool.
- **AC4** — `checkOpenCode` reads the pin via `catalogPin(..., "opencode")` → `matchPinFloorFrom(..., "packages.json")` (checks_deploy.go:339), and `checkShadowedCatalogTools` **is wired** at checks_deploy.go:395 (a defined-but-unregistered function would have failed AC4 in production while its unit test passed). `loadCatalog` reads the checkout before the mirror.
- **AC5** — ADR-036 exists with the channel table, the pin-SSOT decision, and the Linux migration consequence; amended in place for AI-038 and ADR-041. f5 command: **exit 0** (re-run this session).

### Findings

| Severity | Reality | Area | Finding | Evidence | Test (named, or UNTESTED) | Fix location (code / tests / spec / vault) |
|----------|---------|------|---------|----------|---------------------------|---------------------------------------------|
| Major | THEORETICAL | maintainability / SSOT | ADR-036 §3 says "Version detection happens once, in Go" and `dotf tools version`'s help calls `ProbeVersion` "the one extraction every caller shares", but the installer keeps its **own** probes with a different exit-code policy: `Installer.pathVersion` and `Installer.installedVersion` return `""` on **any** non-zero exit, while `ProbeVersion` deliberately keeps the output (its doc: "several tools print the version and then complain about something unrelated"). For a tool that prints a version and exits non-zero, `dotf tools version`/`dotf doctor` see the version while `dotf tools install` sees absent and reinstalls on every run — an idempotence break (`pattern-setup-script-idempotence`) and an AC1-style "already installed; skipping" that never fires. Not real for today's catalog: `opencode`/`bw`/`copilot`/`yarn` all exit 0 here. | Reproduced with a scratch test (created and deleted, tree clean): a fake tool printing `faketool 9.9.9` then `exit 1` gives `pathVersion=""` vs `ProbeVersion="9.9.9"` — output logged, divergence reproduced. Current four npm catalog tools measured `exit=0` on this box. | UNTESTED — no test exercises `pathVersion`/`installedVersion` with a non-zero exit plus output; `version_test.go` covers `ProbeVersion` for it, `install_test.go` does not for the installer | code (route `pathVersion`/`installedVersion` through `tools.ProbeVersion`, or state the policy difference at the ADR/comment) + tests |
| Minor | THEORETICAL | doctor / path resolution | `dirsProviding` de-duplicates by `filepath.Clean` only, not by physical directory, so one install reachable through two PATH spellings of the same directory (a symlink alias such as `/bin` → `/usr/bin`, or scoop's `current`/versioned pair) is counted as two copies and emits a spurious shadowed-copy WARN. The warning says "remove the other channel's copy" for a copy that does not exist. | Read of `checks_catalog.go` `dirsProviding` + `fs.go` `isExecFile` (stat, no `EvalSymlinks`). Not reproduced on this box; `TestCheckShadowedCatalogTools_*` covers exact-path duplication, not alias duplication. | UNTESTED (no alias case in `TestCheckShadowedCatalogTools_NamesEveryDirectoryProvidingTheTool`) | code + tests (`filepath.EvalSymlinks` on the candidate dir, then de-dupe; add an alias case) |
| Minor | THEORETICAL | doctor / error handling | When neither `packages.json` copy is readable, `loadCatalog` returns an empty `Catalog{}` and both `catalogPin` and `checkShadowedCatalogTools` go silently quiet (pin check becomes SKIP, shadow check emits nothing) rather than surfacing that the catalog could not be read — the operator sees health while the pin/coverage is absent. | Read of `checks_catalog.go` `loadCatalog` (ignores `tools.Load` errors) and `matchPinFloorFrom` (`pin == "" → Skip`). Not reproduced. | UNTESTED (no test feeds an unreadable/malformed catalog to `checkShadowedCatalogTools` or `checkOpenCode`) | tests (+ code if a distinct "catalog unreadable" line is wanted) |
| Question | — | process | `review-request.json` carries `review_digest_before: ""`, so the round has no prior-contract digest to compare against; and the resolved base predates 261 unrelated commits. Neither is a defect in the change, but it means "the whole change" for this review had to be bounded to the spec's own commit to be meaningful. | `review-request.json`; `git rev-list --count 8bab48a..HEAD` = 262. | N/A | tooling (launcher) — no spec change |

**No Blocker; one Major, and it is THEORETICAL** (reproduced divergence, but no current catalog tool triggers it). Nothing in the security checklist (injection / secrets / auth / async / concurrency / memory) is touched by this diff: `ProbeVersion` runs `exec.Command` directly (no shell, so no command injection), no credentials are read or printed, and the new code is single-threaded path/file inspection.

### Mutation evidence (test-traceability)

Two mutations were applied and reverted; both were caught by a **named** test, then the tree was restored clean (`git diff` empty for both files):

- `checks_catalog.go`: `len(dirs) > 1` → `> 2` → `TestCheckShadowedCatalogTools_NamesEveryDirectoryProvidingTheTool/two_directories_->_WARN_naming_both` **FAIL**.
- `version.go`: `if err != nil && len(out) == 0` → `if err != nil` (discard output on any non-zero exit) → `TestProbeVersion/version_printed_before_an_unrelated_non-zero_exit` **FAIL**.

Live CLI behaviour, binary built from this tree (`go build ./...` exit 0):

```text
$ dotf tools version ocbanner      # prints "OpenCode locked." then 1.16.2
1.16.2                             # exit 0
$ dotf tools version nover         # prints only "usage: thing"
<empty stdout>                     # exit 1, error on stderr
$ dotf tools version definitely_not_a_tool_xyz
<empty stdout>                     # exit 1
```

Existing suites re-run this session: `go test ./internal/tools/ ./internal/cmd/ -run 'TestProbeVersion|TestToolsVersion'` → **ok**; `go test ./internal/doctor/ -run 'TestCatalogPin|TestCheckShadowedCatalogTools|TestCheckOpenCode'` → **ok**; `bats tests/opencode.bats tests/versions-conf.bats tests/hive-upgrade-timer.bats` → **0 failures**; `bats tests/setup-windows.bats` → **0 failures** (128 cases ok, including "setup-windows.ps1 provisions Node.js before dotf tools install, the npm channel's prerequisite (ADR-036)").

### Evaluator rubric

| Dimension | Grade (A-D) | Rationale (one line) |
|-----------|-------------|----------------------|
| Correctness        | A | Every AC verified against live artifacts; negative paths covered by named tests (banner token, non-zero exit, absent tool, one dir quiet, duplicated PATH entry). |
| Verification       | A | `features.json` f1/f5 re-run exit 0; named Go + bats tests green; CLI exit/empty-stdout reproduced; two mutations shown caught. |
| Scope              | A | `016bf1a` matches `proposal.md` file-for-file (23 files) and touched only `OPENCODE_VERSION` in `versions.conf`; the wide range is a launcher base artifact, disclosed above. |
| Reliability        | A | Error paths handled: empty/absent output → `""`/exit 1; non-zero-with-output preserved by design; duplicated PATH entry collapses; module builds on this tree. |
| Maintainability    | B | New functions are short (`checkShadowedCatalogTools` ~10 lines, `dirsProviding` ~20), single-purpose, with WHY comments citing AI-034; but the "one extraction" the ADR and CLI help claim is not the one the installer uses — `install.go` keeps two probes with a different exit-code policy (Major finding above). |
| Handoff-readiness  | A | ADR-036 + amendment, `verification.md` decisions, `features.json` non-vacuous commands; follow-ups (pi prefix flags, channel pruning) recorded in ADR/proposal. |

Rubric aggregation: no D, no C → **PASS** (one B, the rest A).

### Verdict
PASS

`severity × reality`: the open findings are one Major **THEORETICAL** (probe divergence, reproduced but not triggered by any current catalog tool) and two Minor **THEORETICAL**, all UNTESTED for their specific edge. A Major that is THEORETICAL is tracked, not blocking; no Blocker (any reality) and no REAL Major exists, so the verdict is PASS.

### Recommended next steps

The contract set (`proposal.md`, `tasks.md`, `features.json`) is now **closed** — do not edit it, or this verdict goes stale and `dotf spec archive` refuses it. Track the two minors by disposition in `verification.md` (applied, ticketed, or declined with a reason), or carry them into a follow-up ticket:

- **(code + tests)** Harden `dirsProviding` against symlink-alias duplication (`filepath.EvalSymlinks`, then de-dupe) and add the alias case to `TestCheckShadowedCatalogTools_NamesEveryDirectoryProvidingTheTool`. Low risk, low effort; a false WARN is cosmetic but trains operators to ignore the line.
- **(code + tests)** Route `Installer.pathVersion`/`installedVersion` through `tools.ProbeVersion` (or narrow ADR-036 §3 and the CLI help to "in the Go layer, once per caller") and add an installer test for a tool that prints a version then exits non-zero.
- **(tests)** Add an unreadable/malformed-catalog case so the doctor's silent SKIP is a deliberate, tested behaviour rather than an untested side effect; decide whether it should WARN.
- **(archive)** `dotf spec archive AI-034-opencode-npm-channel` is **advisable** in the current state: `status: verifying`, all ACs evidenced, this review passing and fresh against the contract digests in `review-request.json`. This verdict is not itself modified by those follow-ups.
