---
spec: "AI-028-hive-install-model-migration"
verdict: "PASS-WITH-GAPS"
reviewed_sha: "6b10f02f911a3cf786ccca72616274547067616b"
reviewer: "nan/mimo-v2.6-flash"
date: "2026-11-05"
---

## Adversarial review

**Scope**: AI-028-hive-install-model-migration — `git diff 17c7d4067aa2f9c1022748c82072096d52e6a5d3...HEAD` (the base the launcher resolved). HEAD is `docs/ai028-closing` = origin/main (`0ea7fa98`) + one branch commit (`6b10f02f`, a `tasks.md` closing-list edit); the range also carries three months of main history that arrived through the branch tracking main. Per the review brief, the spec's own work was judged first — the PRs `tasks.md` names: #796, #798, #1311, #1993/#1994, #2001, #2869, #2876, #2901, #2922, #2994, #3013, #3130, #3183 — and the surrounding main work only where it interacts with them (none found conflicting).

**Sources**: `specs/AI-028-hive-install-model-migration/{proposal,tasks,verification,features}.md|json`, `review-request.json`, plus the code and tests named below.

### Spec and task alignment

- **Contract integrity**: `dotf spec digest AI-028-hive-install-model-migration` reproduces all three digests in `review-request.json` exactly (features.json `74bdb3…`, proposal.md `a19924…`, tasks.md `409c63…`). The contract set has not moved since launch.
- **Draft tags**: no live `[AGENT-DRAFT]`/`[AGENT-SUGGESTION]` in any scanned spec file. The only textual occurrence (`tasks.md:17`) sits on a `- [x]` line inside a code span, which `ScanUnresolvedTags` deliberately skips; review machinery files are on the gate's own deny-list. Clean.
- **AC statuses vs. reality**: AC1 superseded-by-AC9 (script + suite deleted with the timer) — consistent with `windows/hive-upgrade.ps1` and `tests/hive-upgrade-timer.bats` being absent from the tree. AC2/AC4 met in tree with named tests plus guest/CI records. AC3/AC7 withdrawn with dated reasons. AC5/AC6 proof is the cited text — verified at `specs/archive/AI-022-hive-daemon-activation/tasks.md` (SUPERSEDED block), `specs/archive/AI-023-hive-auto-upgrade-timer/proposal.md` (forward pointer), proposal Linux resolution. AC8/AC9 met with named, re-run-green tests (below).
- **AC4 mechanism spot-check**: the only `uv tool list` left on either setup's Windows/Linux path is the explanatory comment (`setup-windows.ps1:411`, `setup-linux.sh:459`); `mcp-servers.json` carries no `prerequisite_command` anywhere; `packages.json` gives hive `self_install` (windows) with `args: ["self-upgrade","{version}"]`, `no_build: ["cryptography"]`, `launcher: %LOCALAPPDATA%\hive\bin\hive.cmd`, pin 4.4.2; the daemon gate in both setups resolves the version via `dotf tools version hive`.
- **Closing checklist claims re-run**: `dotf lint powershell` → 31 files, exit 0 (matches `tasks.md`). `go build ./... && go vet ./...` → exit 0. Contract digests match. Branch scope delta is one spec-docs commit — no scope creep.
- **Test-deletion bar** (PR5 deleted `tests/hive-upgrade-timer.bats`, 33 cases, and `windows/hive-upgrade.ps1`): all five fields verified in this session — (1) exact tests named in `verification.md` AC1 (loud-no-install, silent-already-current, PyPI-unreachable, no-guard-collapse); (2) stronger proof shown: `tests/hive-daemon-gate.bats` red/green runs recorded in `verification.md` (red on `d233110d^`), **and I reproduced red in this session** by re-appending `uv tool upgrade hive-vault` to `setup-linux.sh` (bats test 2 failed; reverted); (3) non-test callers: `Register-HiveScheduledTask`'s only caller was the task, and the deployed script copy is removed by `hive-upgrade-retire`; (4) history: `git log --follow windows/hive-upgrade.ps1` shows creation in `6c479896` (#796, AC1) and deletion in `ec5170ff` (#2876), with `tasks.md` PR1 recording the rationale; (5) focused command: `bats tests/hive-daemon-gate.bats && go test ./internal/converge/ -run TestHiveUpgradeRetire` — green.

### Findings

| Severity | Reality | Area | Finding | Evidence | Test (named, or UNTESTED) | Fix location |
|----------|---------|------|---------|----------|---------------------------|--------------|
| Minor | REAL | tests / AC9 guard coverage | The guard's title claims "no setup script installs an unattended hive upgrade" but greps only `setup-linux.sh` and `setup-windows.ps1`. `ai/hermes/setup.sh:83` still runs `uv tool install --upgrade hive-vault` (PyPI latest, not the pin) on every provisioning run — the shape #1993's amendment declared "no longer the target". Not a literal AC9 violation (no timer/task/hook; proposal's 2026-08-07 Linux resolution names hermes unchanged), but the model drifted after the amendment. | repo-wide grep: the only remaining `uv tool install --upgrade hive-vault` outside tests/specs is `ai/hermes/setup.sh:83`; `tests/hive-daemon-gate.bats` greps 2 files; no cron/schedule in-repo runs hermes setup (`ai/hermes/AGENTS.md`: "curled once") | UNTESTED (guard's sensitivity to the twin scripts proven by my mutation; nothing reads hermes's line) | tests (extend the guard) or code (route hermes through the pinned catalog) — disposition in `verification.md` / follow-up ticket; contract set is closed under this verdict |
| Minor | REAL | spec-accuracy / features.json | f8's behavior clause "refuses when the registered PATH cannot be read" described PR6's installer-level refusal `TestInstallUvTool_SelfInstallRefusesAnUnreadableRegisteredPath`, which PR8 deleted with the registered-PATH probe (`tasks.md` PR8 says so). No `registered` token remains in `cli/internal/tools` non-test code, and f8's own verification command (tools/cmd/doctor) no longer runs any test covering that clause. The surviving analogue is converge's step-level `TestTools_AnUnreadableRegisteredPathFailsTheStep` — passing, but outside f8's command. | grep + code read of PR8's seam; `tasks.md` PR8: "The registered-PATH probe and its refusal are deleted" | `TestTools_AnUnreadableRegisteredPathFailsTheStep` (passes; not in f8's command) | spec artifacts (features.json — contract set: do not edit under this verdict; disposition via `verification.md` or a follow-up ticket) |
| Minor | THEORETICAL | verification reproducibility | AC2's two-converge guest proof and AC9's live msi/seeded-guest proofs are cited from artifacts not committed here (`.agent/runs/…/report.md` on a sibling branch; msi/guest run-record dirs are machine state; `verification.md` states "The full 1909-line log is not committed"). They cannot be re-verified inside this repo. The named suites carry each AC independently, so no AC rests on them alone. | `verification.md` own admission + absent paths in tree | f4/f8/f9 all re-run green in this session (mitigation) | none required; note — commit future guest logs if live proof should be repo-reproducible |

**Question/assumption**: whether hermes provisioning ever re-runs unattended (no scheduling evidence found in this repo). If it never re-runs, finding 1 is a model-consistency item only; if a remote bootstrap can re-run it, the `--upgrade` upgrades hive against hive ADR-020's notify-then-user-upgrades doctrine — owner to confirm hermes's run cadence.

**Mutations performed (both reverted; tree clean apart from review state)**: (a) `uv tool upgrade hive-vault` appended to `setup-linux.sh` → `tests/hive-daemon-gate.bats` test 2 red; (b) `{version}` guard in `validateSelfInstall` disabled → `TestLoad_SelfInstall/no_{version}` red. Both restored with `git checkout`, re-verified green; `git status` shows only `review.md` and `review-request.json` untracked.

**Evidence re-run this session** (all fresh, all green):
- f4: `bats tests/hive-daemon-gate.bats` → 2/2 ok.
- f8: the exact `features.json` verification command → exit 0; `-v` shows 34 named PASSes incl. `TestInstallUvTool_SelfInstall/{absent,at_the_pin,below_the_pin,above_the_pin_is_never_downgraded}`, `TestInstallUvTool_SelfInstallProbesOnlyTheLauncher/*`, `TestLoad_SelfInstall/*` (7 refusal subtests), `TestTheRepoCatalogDeclaresHiveSelfInstalledOnWindows`, `TestHiveDaemon_NotInstalledFixFollowsTheOS`.
- f9: `bats tests/hive-daemon-gate.bats` + `go test ./internal/converge/ -run TestHiveUpgradeRetire -v` → 7/7 ok, incl. `TestHiveUpgradeRetire_UnreadableSchedulerFails` (fail-closed) and `..._NothingInstalledIsConverged` (idempotence).
- `tests/packages-json.bats` → all ok (incl. "every packages.json self_install passes the pin and belongs to a uv tool", "hive's Windows self install never builds cryptography from source").
- `go build ./... && go vet ./...` → exit 0; `dotf lint powershell` → 31 files, exit 0.

### Evaluator rubric

| Dimension | Grade (A-D) | Rationale (one line) |
|-----------|-------------|----------------------|
| Correctness        | B | All live ACs met with named passing tests and negative paths (unreadable scheduler, unset launcher, no_build, platform gates, refusals); two doc-level drifts (findings 1–2) keep it from A |
| Verification       | B | Every `features.json` command re-run green here and two mutations reproduced red; the live guest/msi proofs are well-recorded but not reproducible in-repo (UNVERIFIED, finding 3) |
| Scope              | A | Branch delta is one spec-docs commit; each spec PR scoped to one task block; no conflicting interaction found in the surrounding main work |
| Reliability        | B | Retire step fails closed and is idempotent (unit + live records); install failures are loud; hermes provisioning still silently upgrades to latest (finding 1) |
| Maintainability    | B | Small focused functions (`hiveUpgradeRetire` methods well under 40 lines), dead code removed with PR8 (`legacyshim.go`), guards simple grep-based bats — adequate, slightly brittle to renames |
| Handoff-readiness  | A | Proposal/tasks/verification/features updated across all 8 PRs, lessons 159/528 captured, ADR-036 amended, `docs/architecture.md` "hive install model", promotion candidates answered |

Aggregation: no D, no C → rubric alone permits PASS; severity axis carries three Minors, all REAL/THEORETICAL, each dispositioned below.

### Verdict

**PASS WITH GAPS** — no Blockers, no Majors; three Minor findings (two REAL, one THEORETICAL), each with a disposition line. Per `severity × reality`, minors cannot block; per the test-traceability gate, finding 1 is UNTESTED and is tracked rather than waived. Rubric all B or above.

### Recommended next steps

Route these through `verification.md` disposition (applied / ticketed / declined with a reason) or a follow-up ticket — **not** as edits to `proposal.md` / `tasks.md` / `features.json`, which are the contract set this verdict closes:

1. **Finding 1 (tests or code):** either add `ai/hermes/setup.sh` to the guard's grep set in `tests/hive-daemon-gate.bats`, or replace `uv tool install --upgrade hive-vault` there with the pinned catalog install — after confirming hermes's provisioning cadence (Question above). Ticket is the clean route.
2. **Finding 2 (spec artifacts):** file a ticket to reconcile f8's "registered PATH" clause in `features.json` with PR8's launcher-only probe (or fold `TestTools_AnUnreadableRegisteredPathFailsTheStep` into f8's verification command).
3. **Finding 3 (note):** accept as UNVERIFIED-in-repo; future guest/MSI proofs should commit their logs if repo-reproducible evidence is wanted.
4. A second-opinion pass on this verdict was not run: harness subagent delegation was not operator-requested in this launch, and the pool draw already provides review independence from the implementer.

**Verdict: PASS WITH GAPS. `dotf spec archive AI-028-hive-install-model-migration` is advisable** in this state: digests match the review request, no live draft tags remain, promotions are answered, and the verdict is passing. The three findings are tracked gaps, not archive gates.
