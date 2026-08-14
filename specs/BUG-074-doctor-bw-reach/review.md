---
spec: "BUG-074-doctor-bw-reach"
verdict: "FAIL"
reviewed_sha: "a11a459dda9443620cec3abb95a242a3872e0667"
reviewer: "claude-opus-5"
date: "2026-08-13"
---

## Adversarial review

**Scope**: BUG-074-doctor-bw-reach / PR #950 (branch `fix/doctor-bw-reach`) — round 2, against
`a11a459` (`145516c`, `13b3d12`, two main merges, `12b8e77`, `acc495f`, `a11a459`).

**Sources**: `specs/BUG-074-doctor-bw-reach/{proposal,tasks,verification}.md` + `features.json` +
the round-1 `review.md`; `git diff origin/main...HEAD`; `cli/internal/doctor/checks_bw_reach.go`,
`checks_bw_reach_test.go`, `system.go`, `doctor.go`, `report.go`, `checks_secrets_tooling.go`,
`checks_pat.go`, `checks_contract.go`, `checks_test.go`, `testhelpers_test.go`;
`cli/internal/env/env.go`; `cli/internal/spec/review.go`; `cli/internal/tools/{catalog,install}.go`;
`packages.json`; `secrets/registry.yaml`; `setup-linux.sh`; `tests/Dockerfile.integration`;
`.zshrc` / `.bashrc`; `docs/lessons.md`. Live: `go build`/`go vet`/`go test ./...`,
`golangci-lint run` (pinned 2.12.2), `check-spec-gate.sh`, a 9-mutant battery, a throwaway
reproduction test (removed), the real `bw` binary, and a `dotf` built from this head.

### Spec and task alignment

- **All three round-1 Majors are verifiably closed.** I re-ran the reviewer's own mutant
  (`s.Backend == "bw"` → `"bitwarden"`): it is now caught by `TestBWBackedSecrets_CountsOnlyBWBackend`.
  The bounded-exec seam is real — a compiling deadline-ignored variant of the production closure
  (`_ = ctx` + `exec.Command`) makes `TestCommandOutputBounded_KillsAnOverrunningCommand` go red
  after 10s. And risk 3's rewritten chain checks out line by line: `tests/Dockerfile.integration:55`
  is `RUN … bash setup-linux.sh`, `setup-linux.sh:1505` is `dotf doctor || log_warning …`, and the
  image installs no `bw`.
- **Verification.md's round-2 mutation table is accurate.** I reproduced all eight rows
  independently: producer predicate, producer counts-everything, producer silent fallback
  (`RepoRegistryPath` → `ResolveRegistryPath`, caught by `TestBWBackedSecrets_MissingRegistryErrors`),
  bounded-exec deadline, negative-skew guard, staleness never fires, reach claimed without `bw sync`,
  severity consumer forced advisory — **8/8 detected**, tree reverted and clean after each.
- **The `a11a459` doc-drift fix is correct.** `bwBackedSecrets` really does call
  `env.RepoRegistryPath()` (`checks_bw_reach.go:205`), and the new wording in `system.go`,
  `proposal.md` and `verification.md` describes `RepoRegistryPath`'s actual contract (checkout-only,
  fails loud) rather than `ResolveRegistryPath`'s (prefers checkout, falls back). Verified against
  `env.go:176-212`. Not a residual finding.
- All six `features.json` verification commands re-run independently: **14 test functions (17 cases
  counting `DegradesOnUnusableStatus`'s subtests), all pass**.
  `go build ./...`, `go vet ./...` clean; `golangci-lint run` → `0 issues` on the pinned 2.12.2
  (matches `versions.conf`); `check-spec-gate.sh --base-ref origin/main --head-ref HEAD` → OK.
- No `[AGENT-DRAFT]` / `[AGENT-SUGGESTION]` tags remain in any spec file.
- **Tier 1 is not dead code** — a question round 1 left open by never observing it. Verified against
  the real binary with a redirected `BITWARDENCLI_APPDATA_DIR`: `bw status` on a logged-out profile
  exits 0 and prints `{"serverUrl":null,"lastSync":null,"status":"unauthenticated"}`. The
  `unauthenticated` branch is reachable in production. That probe is also what surfaced finding 1.
- **PR state (not a finding).** The PR head is `5168df6`, a merge of `main` carrying PR #949's test
  work; it touches no file of this change and no contract file, so this review is not stale against
  it. `spec-gate` is RED for the designed archive-on-merge ordering ("this PR closes an issue whose
  spec is still active"), confirmed from the job log; lint / test (ubuntu + windows) /
  lint-powershell / CodeRabbit / GitGuardian are green, `cli` and `CI` still in progress at review
  time.
- `tasks.md` and `verification.md` are honest about round 2, including recording the review request
  *before* the review ran to avoid staling its own verdict. `docs/lessons.md` carries the promised
  entry and its index was genuinely regenerated (195 entries; index diffs clean against the body).

### Findings

| Severity | Reality | Area | Finding | Evidence | Test (named, or UNTESTED) | Fix location |
|---|---|---|---|---|---|---|
| Major | REAL | output parsing / exec | `bw status` is read through `CommandOutputBounded` (`CombinedOutput`, i.e. stdout **merged with stderr**) and then `json.Unmarshal`ed whole (`checks_bw_reach.go:92-101`). `bw` writes its JSON to stdout and diagnostics to stderr, so **any** stderr line from `bw` makes the parse fail, and the check returns at line 100 — all three tiers skipped, AC1/AC2/AC3 all silently unevaluated. The state I reproduced is not exotic: it is `bw`'s first-ever invocation on a machine, which is exactly what `dotf doctor` triggers as the last step of `setup-linux.sh` on a freshly provisioned box. Post-migration the same chatter downgrades the AC1 FAIL to a WARN — it defeats the escalation the severity policy exists for. | **Reproduced twice, live, on the real binary.** Stream separation: `BITWARDENCLI_APPDATA_DIR=$TD bw status 2>&1 1>/dev/null` → `Could not find data file, "…/data.json"; creating it instead.`; `… 2>/dev/null` → the clean JSON. End-to-end with `dotf` built from this head: run 1 (fresh app-data dir) → `[WARN] `bw status` returned no parseable JSON — reach unverified`; run 2 (same dir, `data.json` now present) → `[WARN] Bitwarden session is gone (`bw status`: unauthenticated) …`. Same root cause makes `bwFailDetail`'s `firstLine` (`:234-239`) return chatter instead of `bw`'s real error. | **UNTESTED** — every fake returns pure JSON (`bwStatusJSON`, `testhelpers_test.go:24-36`). `TestBWReach_DegradesOnUnusableStatus/not json` proves the degradation is *safe*, not that the check survives `bw`'s own stderr. Needs e.g. `TestBWReach_ToleratesCLIChatterOnStderr` feeding `"Could not find data file…\n{json}"`. | code (capture stdout separately, or extract the JSON object from the blob) + tests |
| Major | REAL | severity policy / spec-vs-code | Tier 3's sync failure calls `rep.Fail` unconditionally (`checks_bw_reach.go:149-152`) — it never consults `live`. So on today's machine (34/34 registry entries `age`, exposure 0) an unlocked vault plus any sync failure — offline, captive portal, Bitwarden outage, or the new 45s deadline firing — exits `dotf doctor` 1 for a condition that breaks nothing. That contradicts `proposal.md` `## What` ("advisory while everything is still on age, FAIL from the first migrated secret") **and the check's own header comment** 80 lines above it (`:69-73`), and it is inconsistent with doctor's established handling of an unreachable remote: `checks_pat.go:86` warns when `api.github.com` cannot be reached. Round 2's bounded exec widened this: what used to hang now deterministically produces this FAIL. | **Reproduced** with a throwaway test (added, run, deleted): `live=0`, status `unlocked`, sync returns `errors.New("bw timed out after 45s")` → `Failures()==1`, report line `[FAIL] Bitwarden sync FAILED on an unlocked vault …`. Counter-reading stated honestly: AC1's wording scopes exposure-keying to the `unauthenticated` case only, so this may be intended — but then the general policy sentence and the code comment both overstate it. | **UNTESTED** — `TestBWReach_UnlockedButSyncFailsIsAFail` pins `live=2` and `TestBWReach_UnlockedVaultProvesReach` pins `live=5`; no test drives a sync failure at `live=0`. | code (key it to `live`, e.g. `Warn` at 0) **or** spec (`proposal.md`: declare the flat FAIL deliberate). Either closes it; silence does not |
| Major | REAL | coverage / registration | Nothing proves the check is wired into the product. Deleting the registration line `checkBitwardenReach(sys, rep)` (`doctor.go:89`) leaves the **entire `cli/` suite green**: 13 packages `ok`, exit 0. Every test calls `checkBitwardenReach` directly. A refactor of `Run()` or a bad merge resolution can drop the whole reach section and CI will not notice — the same class as round 1's surviving producer mutant, one level up, and the same class as #898 (a check never observed failing is not evidence) that this spec exists to fix. | **Mutation run**, tree reverted, `git status` clean. The assertion pattern already exists in the file the new check bypassed: `TestRun_QuickSkipsHeavySections` (`checks_test.go:391`) asserts full-mode output contains `"Core tools in PATH"`; nothing analogous names `"Bitwarden reach"`. Note the section header prints even with nothing on PATH (`Section()` precedes the `has("bw")` skip), so the assertion is one line. | **UNTESTED** | tests |
| Minor | REAL | reporting | `rep.Fix("export BW_SESSION=$(bw login --raw) && bw sync")` (`checks_bw_reach.go:111`) is emitted on a plain read-only `dotf doctor`, and `checkBitwardenReach` does not even receive `opts.Fix`. `report.go:113` then prints `Applied 1 fix action(s)` for a repair nothing applied. Every other `rep.Fix` caller emits it *after* performing a repair (`checks_automemory.go:52`, `checks_memshape.go:119`, `checks_guard.go:43`, `checks_vault_hooks.go:86`, `checks_deploy.go:427`) or gates it on `--fix` (`checks_contract.go:35-37`). | **Reproduced live**: `dotf doctor` (no `--fix`) against a temp registry with 2 `backend: bw` entries → `[FIX ] export BW_SESSION=$(bw login --raw) && bw sync` and the summary line `Applied 1 fix action(s)`. | **UNTESTED** — no test asserts the FIX line or the summary counter. | code (fold the hint into the FAIL message, or take `fix bool` and gate it) |
| Minor | THEORETICAL | spec accuracy (CI risk) | Risk 3 names two triggers that would turn a migrated registry into red CI ("making doctor fatal in setup, or adding `bw` to the image"). It misses a third and less obvious one: `setup-linux.sh:301` runs `dotf tools install`, and `packages.json:19-25` declares `bw` as an npm-sourced tool (`@bitwarden/cli`, profile `full`). `bw` stays absent from the integration container only because the image installs no node/npm — so adding node/npm for any unrelated reason installs `bw` and activates the reach check in CI. The stated conclusion still holds on safeguard (a) alone (`dotf doctor \|\| log_warning`), but a safeguard the spec calls load-bearing rests on an undocumented precondition. | `packages.json:19-25`; `setup-linux.sh:301` vs `:1505`; `tests/Dockerfile.integration` apt list (no node/npm/bw); `cli/internal/tools/install.go:170-198` (`installNpm`). | UNTESTED (nothing asserts node/npm or `bw` stay absent from the image) | spec |
| Minor | REAL | severity source | `bwBackedSecrets` resolves through `env.RepoDir()`, which falls back to a **cwd walk-up for `.git`** when `DOTFILES_REPO_DIR` is unset. Running `dotf doctor` from inside any other git repo therefore reads *that* repo's `secrets/registry.yaml`, fails, and degrades severity to advisory — converting a post-migration FAIL into a WARN. It says so loudly, which is the right direction, but the code comment (`:200-203`) documents only the "machine with no checkout at all" case, not this one. Mitigated in practice: `.zshrc:34` and `.bashrc:64` both export `DOTFILES_REPO_DIR`. | **Reproduced live**: `env -u DOTFILES_REPO_DIR dotf doctor` from a temp `git init` dir → `[WARN] registry unreadable (open …/otherrepo/secrets/registry.yaml: no such file or directory) — reach severity degraded to advisory`. | **UNTESTED** — `TestBWBackedSecrets_MissingRegistryErrors` sets `DOTFILES_REPO_DIR`, so it never exercises the walk-up branch. | code (or one line of spec/comment) |
| Question | — | threshold provenance | The 30d staleness threshold is justified solely as "30d < the observed 45d, so it lands while the token is still renewable" (`proposal.md:72-75`). The incident bounds token death at **≤45d**, not at **>30d**; no upstream Bitwarden refresh-token lifetime is cited anywhere in the spec or the code. Since round 2 reassigned the whole prevention claim to tier 2, that claim now rests on an inequality nothing establishes: if the real idle lifetime is ≤30d, tier 2 warns post-mortem. | `proposal.md:36-39`, `:72-75`; `checks_bw_reach.go:15-27`. No citation present in either. | n/a | spec (cite the upstream lifetime, or soften "still renewable" to "earlier than the only expiry we have observed") |

### Evaluator rubric

| Dimension | Grade (A-D) | Rationale (one line) |
|-----------|-------------|----------------------|
| Correctness        | C | Two REAL defects — a real `bw` stderr line skips all three tiers, and tier 3's FAIL ignores the exposure policy the spec and the code comment both state; not D because nothing ever claims false reach and every failure degrades to advisory. |
| Verification       | C | Genuinely stronger than round 1 (producer and production closure now covered; I re-ran all 8 claimed mutants and all die), but three new mutants/states survive untested: the registration line, the `live=0` sync failure, and `bw`'s real stderr. |
| Scope              | A | Diff is the check, its seam, its registration, its tests, one reworded PASS line, the spec folder and one `docs/lessons.md` entry — nothing unrelated. |
| Reliability        | C | The network is now genuinely bounded (a real round-2 fix), but the output-parsing path drops the entire signal on a reproducible real `bw` state and the failure severity contradicts the documented policy. |
| Maintainability    | B | Comments explain WHY at length and, after `a11a459`, accurately; functions are short; `firstLine` reused rather than redeclared. Smell: the check emits a `[FIX ]` action it cannot apply and cannot see `opts.Fix`. |
| Handoff-readiness  | A | Spec artifacts, the round-2 remediation log, the `docs/lessons.md` promotion (index regenerated, not appended) and the archive checklist are all present, and the review request was recorded before the review ran rather than after. |

### Verdict

**FAIL**

Three Majors, all **REAL** (each reproduced, not argued) and all **UNTESTED**, which the skill says
cannot be closed by the implementer's assurance alone. The rubric path agrees independently (three
Cs, no D → PASS-WITH-GAPS floor); the severity path is more severe and governs.

The load-bearing one is the first, and it is worth stating plainly: on a freshly provisioned
machine — the moment `setup-linux.sh` ends and doctor runs for the first time — `bw`'s own stderr
line makes this check print `reach unverified` and skip every tier it was built for. The change
correctly stopped believing `PATH` presence; it now believes `CombinedOutput` is JSON. Both are
assumptions about representation rather than behaviour, which is the #852 class the spec cites.

None of the three round-1 Majors regressed, and the fixes for them are real (verified by mutation,
not by reading). Two of the new findings exist *because* of round 2 — the bounded exec turned a hang
into a deterministic FAIL, and the producer's move to `RepoRegistryPath` made the cwd walk-up the
new severity-loss path. That is normal for a second pass, not a criticism of it.

### Recommended next steps (before archive)

`dotf spec archive BUG-074-doctor-bw-reach` is **not advisable** and will refuse
(`checkReviewGate` blocks on `verdict: FAIL`). Minimum set to flip to PASS:

1. **Parse `bw status` from stdout, not from stdout+stderr.** Either add a stdout-only bounded exec
   seam or extract the JSON object from the blob before unmarshalling; keep the combined output for
   the error-detail path. Add a named test feeding `"Could not find data file…\n{json}"` and confirm
   it goes red against today's code before landing it. Re-run the live check the same way I did
   (`BITWARDENCLI_APPDATA_DIR=$(mktemp -d) dotf doctor`) — that is the one-command reproduction.
2. **Decide tier 3's severity and make the artifacts agree.** Either key the sync-failure branch to
   `live` (with `TestBWReach_UnlockedSyncFailureIsAdvisoryWhenUnexposed` or similar), or amend
   `proposal.md` to say the flat FAIL is deliberate and why doctor's own PAT precedent does not
   apply. Either satisfies the gate.
3. **Prove the check is registered.** One assertion that `Run(...)` in full mode emits
   `Bitwarden reach (live secrets SSOT)`, verified red with `doctor.go:89` removed.
4. Non-gating: stop emitting `rep.Fix` on a read-only run (it inflates `Applied N fix action(s)`);
   add the node/npm trigger to risk 3; note the cwd-walk-up severity loss; cite or soften the 30d
   provenance.
5. Then the archive checklist's remaining items — `status: archived`, move the folder, close #944
   with the PR link. The `docs/lessons.md` promotion is already done this round.

**Re-review is required, not optional.** Steps 2-4 touch `proposal.md` and/or `tasks.md`, both
contract files (`cli/internal/spec/review.go:23`), so this verdict goes stale by construction the
moment they change. The path to green is fix → one fresh `/adversarial-review` against the new head,
in a session that did not write the fix.
