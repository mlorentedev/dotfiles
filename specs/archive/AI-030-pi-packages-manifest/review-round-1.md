---
spec: "AI-030-pi-packages-manifest"
verdict: "FAIL"
reviewed_sha: "354539fa6c4f68c915b4b63487d8796da5f0eaf7"
reviewer: "nan/qwen3.8-flash"
date: "2026-09-30"
---

## Adversarial review

**Scope**: AI-030-pi-packages-manifest — `git diff 180116b0a1dee71b57b8ec64ff40813e711e1fb1...HEAD` (launcher-resolved base; 320 commits, so the AI-030-relevant slice was judged against every file the spec's own criteria name, including the HARNESS-139 port that this spec's criteria now point at).

**Sources**: `specs/AI-030-pi-packages-manifest/{proposal,tasks,verification,features}.md`/`.json`; `ai/pi/packages.json`; `ai/pi/README.md`; `cli/internal/pi/{packages,apply}.go`; `cli/internal/cmd/{pi.go,pi_test.go}`; `cli/internal/doctor/checks_pi_extensions*.go`; `tests/pi-packages.bats`; `setup-linux.sh:712-856`; `setup-windows.ps1:1274-1287`; `.github/workflows/ci.yml:425-520`; commit `e4b39f55` (HARNESS-139) and `9c44d7a7` (AI-030).

### Spec and task alignment

- All 11 `features.json` verification commands were run fresh at `354539fa`: **all exit 0** (f1..f11).
- Every `-run '^(...)$'` filter was re-run with `-v` and the named tests were confirmed to execute: `TestNewPlan`, `TestApplyConvergesAndASecondRunCallsNothing`, `TestPiPackagesApplyRemovesThenInstalls`, `TestLiveSourcesReadsBothEntryForms`, `TestIdentityIgnoresTheVersion`, `TestLoadManifestRefusesWhatItCannotRead`, `TestPiPackagesCheckRefusesAnUnreadableManifest`, `TestPiPackagesApplyWithoutPiWarnsAndExitsZero`, `TestPiPackagesApplyWithoutNpmWarnsAndExitsZero`, `TestPiPackagesApplySkipIsFirstAndLoud`, 6× `TestPiExtensions*`. No command passes vacuously on the filter-matches-nothing failure mode (lesson 309).
- `go build ./...` → OK; `go test ./... -count=1` (all packages) → 0 failures, so the "no regressions" claim holds for the Go suite.
- Non-vacuity proven by mutation, each reverted (`git status` clean apart from the launcher's `review-request.json`):
  - **M1** unpinned `npm:pi-effort` in the manifest → f2 exit 1 **and** `@test "pi packages: every declared source is pinned to a version"` red, while `"the pin guard rejects an unpinned source"` stayed green. AC2's two halves both have teeth.
  - **M2** drop `--pi "$PI_BIN"` from `setup-linux.sh` → f9 exit 1 (AC9 guard is real).
  - **M3** drop `--repo $DotfilesDir` from the Windows twin → f10 exit 1.
  - **M4** make `NewPlan` additive-only → `TestNewPlan`, `TestApplyConvergesAndASecondRunCallsNothing`, `TestPiPackagesApplyRemovesThenInstalls` all red.
  - **M5** make `LiveSources` string-only → `TestLiveSourcesReadsBothEntryForms` red (AC6 real).
  - **M7** stop `validate()` refusing an empty package list → `TestLoadManifestRefusesWhatItCannotRead/no_packages` and `TestPiPackagesCheckRefusesAnUnreadableManifest` red (AC8 real, and it is the guard that stops the destructive read-everything-removed case).
- No `[AGENT-DRAFT]` / `[AGENT-SUGGESTION]` tags in `proposal.md`, `tasks.md`, `verification.md` or `features.json` (grep count 0 each; matches inside `review-transcript.jsonl` are this run's own log, not spec artifacts).
- **Where the contract and the code disagree**: see F-01. The spec folder is not internally consistent with HEAD, and `ai/pi/README.md` — the operator-facing doc — agrees with the *code*, not with the spec.

### Findings

| Severity | Reality | Area | Finding | Evidence | Test (named, or UNTESTED) | Fix location (code / tests / spec / vault) |
|---|---|---|---|---|---|---|
| **Major** (borders on Blocker) | **REAL** | spec↔code: destructive behaviour declared out of scope | `proposal.md` "Out of scope — Removing packages": *"The reconcile is additive: it installs what is declared and missing. It never uninstalls what is present and undeclared, because a package installed deliberately outside this manifest is not evidence of drift and `pi remove` on a human's own extension is not setup's decision to make."* The code at HEAD does exactly that: it removes undeclared live entries, including hand-installed ones. Archiving this spec freezes a false non-goal as the manifest's provenance. | Reproduced in this review: `dotf pi packages apply --repo . --dry-run --agent-dir <synthetic>` against `{"packages":["npm:pi-effort@0.0.8","npm:some-human-favorite@1.2.3",{"source":"npm:@scope/x@2.0.0"}]}` printed `remove npm:some-human-favorite@1.2.3 (not declared)`, `remove npm:@scope/x@2.0.0 (not declared)`, and `pi packages: 2 to remove, 9 to install, 0 to retire`. Mutation **M4** shows the removal path is the tested, load-bearing behaviour, not an accident. HARNESS-139 `proposal.md` records the supersession ("AI-030 is superseded in part… dispositioned by the zombie-spec sweep (W1.4, #1626), **not in this change**"), so no spec owns the edit to AI-030's text. | `TestPiPackagesApplyRemovesThenInstalls` (proves the *code*, i.e. the spec text is the stale side); no test can cover a prose non-goal → the gap is the artifact | **spec** (`proposal.md`: Out-of-scope paragraph, "nine packages"→ten, AC4-AC10 wording) → **fix, then re-review** |
| **Major** | **REAL** (past incident of the identical mechanism, cited) | platform parity / silent non-execution | `setup-windows.ps1:1280` gates the reconcile on `Get-Command dotf` alone. `scripts/install-dotf.ps1` installs to `$env:USERPROFILE\.local\bin` and never touches `$env:PATH` (grep for `env:PATH` in that file: no matches). On a machine where that directory is not already on the *session* PATH, the Windows twin prints `dotf not found - pi packages not reconciled` and setup exits 0 — AC4 and AC10 silently do not happen. The Linux twin has the fallback (`command -v dotf` → `elif [ -x "$HOME/.local/bin/dotf" ]`) precisely because this failed before; the Windows script itself already uses the fallback at lines 1792 and 2199 for other blocks. | `setup-linux.sh:556-564` comment: *"the integration container installs dotf and then cannot see it in the same run (#1202 was the identical trap with jq)"*; named guard `@test "setup-linux.sh resolves dotf by path for the harness mirror, not only by name (#1202 class)"` (also cites #1305: "the integration container installed dotf and then skipped the mirror in the same run, and verify-setup.bats caught the gap"). Windows was made safe by an out-of-band CI workaround, not by the call site: `ci.yml:439-454` "Build dotf from this PR and put ~/.local/bin on PATH (TEST-003)" … "Install-Dotf then could not see the [newly built binary]". CI therefore *masks* the fresh-machine case rather than disproving it. | **UNTESTED** on Windows — the Linux guard is a whole-file grep with no per-call-site scope; no `tests/setup-windows.bats` case pins dotf resolution for the pi block | **code** (`setup-windows.ps1`: resolve `~\.local\bin\dotf.exe` as a fallback, and pass `--pi` parity) + **tests** (extend the #1202-class guard to both twins' pi blocks) |
| **Minor** | REAL | verification accuracy | `verification.md` "Test status" claims `bats tests/pi-packages.bats` → `1..16   all ok`. Actual at HEAD: `1..14`, all ok. It also still prints a full output block for `specs/AI-030-pi-packages-manifest/verify-reconcile.sh`, a script that is no longer in that folder (retired by HARNESS-139), so the quoted evidence is unreproducible — the "Re-pointed" prose below says so, but the stale block remains presented as a current run. | `bats tests/pi-packages.bats` → `1..14` (run here); `ls specs/AI-030-pi-packages-manifest/` shows no `verify-reconcile.sh` | n/a (evidence artifact) | **spec artifacts — but `verification.md`, which is outside the staleness set** |
| **Minor** | REAL | contract duplication / SSOT | f2 re-implements the pin regex inline (`jq … \| grep -vcE '^npm:@?[a-z0-9._/-]+@[0-9]+\.[0-9]+\.[0-9]+'`) instead of invoking the named bats test that holds AC2's first half. The same regex now exists in two commands plus the bats test: break the bats guard's regex and f2 stays green, so the contract silently loses a check. M1 shows they agree *today*. | `features.json` f2 vs `tests/pi-packages.bats` "every declared source is pinned to a version" | the bats test named above (f2 does not run it) | **spec** (`features.json`) — deferred to the next contract round with F-01 |
| **Minor** | THEORETICAL | input handling | `LiveSources` drops a live object-form entry that has no `source` key — no error, invisible to both directions. Harmless under the old additive rule; under removal semantics it means an entry pi wrote in a shape the loader does not know is never reported at all (`check` prints a clean plan, `apply` reports `changed=0`) while the package sits installed. | `cli/internal/pi/packages.go`: `if json.Unmarshal(entry, &obj) == nil && obj.Source != ""` — the else path appends nothing and records nothing | **UNTESTED** (no `packages_test.go` case for a source-less object entry) | **code + tests** |
| **Minor** | SPECULATIVE | identity parsing | `Identity()` strips everything after the *last* `@` in the remainder, so it treats a dist-tag (`npm:foo@latest`) and a local-path source containing `@` the same way it treats a semver version. Today's manifest is all pinned `npm:` sources, and the pin guard in CI keeps floating tags out, so this cannot fire in the shipped configuration — it is a parser robustness note, not a live defect. | `cli/internal/pi/packages.go` `Identity`; `TestIdentityIgnoresTheVersion` covers the scoped/semver cases only | `TestIdentityIgnoresTheVersion` (covers the pinned cases; not the tag/path cases) | — (surface only; do not gate) |
| **Minor** | REAL | doc drift outside the spec | `ai/pi/README.md:13` still says setup "installs the difference against the live `settings.json` on every run, through `pi install`" — install-only. The manifest's own `$comment` and the Go package doc say both directions. A builder reads the README first, so the wrong sentence is in the highest-traffic place. | grep for `additive\|pi remove\|BOTH direction` in `ai/pi/README.md` → no match for the removal semantics; line 13 text quoted | n/a (doc) | **code/docs** (`ai/pi/README.md`) |
| Question | — | archive mechanics | Every `features.json` entry is `"state": "pending"`; only the harness may write `passing`. Was the harness run against these commands after the HARNESS-139 re-point? The commands do exit 0 when run by hand (verified above), so this is a sequencing question, not a doubt about the tests. | `features.json` (all `pending`); the 11 commands run here all exit 0 | n/a | author confirmation |

### Evaluator rubric

| Dimension | Grade (A-D) | Rationale (one line) |
|---|---|---|
| Correctness | **C** | AC1-AC9, AC11 verified with named tests and six mutations; AC10's "same semantics" fails on the fresh-Windows path (F-02) and the shipped behaviour contradicts the spec's own non-goal (F-01). |
| Verification | **C** | Evidence is reproducible and honest about the un-run smoke test, but the file asserts `1..16` where the suite is `1..14` and reprints output from a deleted verifier. |
| Scope | **B** | The AI-030 files are on-contract; the port to Go is a documented, separately-specced dependency whose criteria this folder correctly re-points. Stated base carries 320 unrelated commits, which is the launcher's scope, not creep. |
| Reliability | **B** | Fails safe where it matters: empty/unreadable manifest refused (M7), unreadable live settings refused, missing pi/npm degrades to one warning, retire never deletes and never clobbers. One silent-skip path (F-02) and one silently-invisible entry (F-05). |
| Maintainability | **B** | ~316 Go lines, short functions, low complexity, seams make tests hermetic (`HOME` stubbed, so `resolvePi` is deterministic); regex duplication across three places is the blemish. |
| Handoff-readiness | **C** | `docs/lessons/lesson-231-…` captured, supersession acknowledged — but the spec that is being archived was never reconciled with the change that replaced its mechanism, and nobody owns that edit. |

Aggregation: no **D**, several **C** → PASS WITH GAPS on the rubric alone. Escalated by the severity path: **F-01 is a REAL Major whose fix lands in the contract set**, and the skill's rule is that a contract-set fix means *fix, then re-review*, not a passing verdict that asks for contract edits. FAIL is therefore the verdict, and it is a one-paragraph fix, not a rewrite.

### Verdict

**FAIL**

### Recommended next steps

Contract set (these are the point of a FAIL; a re-review follows them):

1. **F-01** — rewrite `proposal.md` "Out of scope → Removing packages" into what is now true: the reconcile is bidirectional, an undeclared live package **is** removed by `dotf pi packages apply`, and the decision was made in HARNESS-139 (#1628, decision D-2 on #1625) — with the removal-totality risk stated in **Risks**, not denied in **Out of scope**. In the same edit: ten packages, not nine; re-point AC4-AC10's wording at the Go mechanism so the criteria describe what runs. Then `dotf spec review AI-030-pi-packages-manifest` for round 2.
2. **F-04** (optional, same window) — make f2 call the named bats test instead of duplicating the regex.

Outside the contract set — apply any time, they will not invalidate the next review; disposition each in `verification.md` (applied / ticketed / declined with a reason):

3. **F-02** — give `setup-windows.ps1:1280` the `~\.local\bin\dotf.exe` fallback its twin has, and extend the #1202-class guard so it fails on *either* twin's pi block, not on a whole-file grep.
4. **F-03** — correct `1..16` → `1..14` and mark the retired `verify-reconcile.sh` block as historical rather than present evidence.
5. **F-06** — one line in `ai/pi/README.md` for the removal semantics.
6. **F-05** — `LiveSources` should report (or refuse) an object entry with no `source`, plus a table case.

### What is solid (mitigations, not praise)

- The removal semantics that F-01 exposes are **guarded where removal is dangerous**: `validate()` refuses a manifest with zero packages precisely because an empty want-list would plan the removal of everything live (`TestLoadManifestRefusesWhatItCannotRead/no_packages`, red under M7), and `loadPiTarget` errors rather than guessing when the live settings file exists but does not parse.
- Both twins pass `--repo` explicitly, which closes the worktree case where the wrong checkout's manifest would drive a destructive apply — and that is pinned by `@test "pi packages: both twins reconcile through dotf pi packages apply, naming their own checkout"` (red under M3).
- AC11's premise is verified, not asserted: the doctor check keys its FAIL/WARN on what is **on disk** under `~/.pi/agent/npm/node_modules`, and the quarantine target is outside `~/.pi/agent/extensions/` — the reason `.disabled/` next door would have been a no-op is documented against pi's own discovery globs, with `TestPiExtensionsFixQuarantinesOutsideTheScannedTree` and `TestPiExtensionsFixNeverClobbersAnEarlierQuarantine` holding it.
- `verification.md` states plainly that the real `pi` smoke test was **not** run and why. That is the correct disclosure, and it is the reason F-02 stayed invisible to the suite: the tests prove the plan and the call, never that the call site is reached.

**`dotf spec archive` is NOT advisable in this state.** Minimum set to flip to PASS: item 1 (contract edit) + a fresh round-2 review; the rubric C grades on Verification and Handoff-readiness clear with items 3-4-5, which need no re-review.
