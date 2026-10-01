---
spec: "OPS-042-npm-tools-to-catalog"
verdict: "PASS-WITH-GAPS"
reviewed_sha: "6d1393d7ec4fc3a2c0b8d2c7454354f63fc0f3d2"
reviewer: "nan/qwen3.8-flash"
date: "2026-09-30"
---

## Adversarial review

**Scope**: OPS-042-npm-tools-to-catalog — `git diff 4718e4685d3c4aab4dcbff6a400581f5c36b961f...HEAD` (base as resolved by the launcher). That range is 224 commits / 945 files because the worktree branch carries everything merged since the base; the OPS-042 change inside it is two commits: `1478f24` (the move, PR #1382) and `52e99a2` (the amendment dropping `obsidian`, PR #1616, which amends AC1/f1). Those two, the state at HEAD, and the guards they left are what this review judges; the rest of the range is other specs' work and was not reviewed.

**Sources**: `specs/OPS-042-npm-tools-to-catalog/{proposal,tasks,verification,features}.md/json`; diff `4718e46...HEAD`; `packages.json`, `versions.conf`, `setup-linux.sh`, `setup-windows.ps1`, `cli/internal/doctor/{checks_tools,checks_catalog,checks_deploy}.go`, `cli/internal/tools/{install,catalog}.go`, `cli/internal/cmd/tools.go`, `tests/{setup-linux,setup-windows,packages-json}.bats`.

### Spec and task alignment

- All four ACs are ticked `[x]` in `proposal.md` and `tasks.md`; every implementation task carries diff evidence (files actually changed in `1478f24`), so no `[x]` is bare.
- AC1 (amended): verified at HEAD. `jq` shows exactly one relevant entry — `yarn 1.22.22 npm:yarn` — and no tool named `obsidian` or packaged `obsidian-cli`. `dotf tools list` built from HEAD and run with `DOTFILES_DIR` at this worktree printed `yarn 1.22.22 full npm:yarn`, so the second half of AC1 ("`dotf tools list` shows it") reproduces on Linux, not only in the August Windows transcript.
- AC2: verified. `setup-linux.sh` and `setup-windows.ps1` contain no obsidian/yarn npm block; `bash -n setup-linux.sh` clean. The deletion is real: `1478f24` removes 50 shell lines and 73 PowerShell lines, including the `versions.conf` parser.
- AC3: verified, and **verified causally** — dropping the `yarn` entry from `packages.json` (mutation, reverted) makes `TestCheckVersionMatch` fail, so the doctor row genuinely reads the pin from the catalog rather than from `versions.conf`. On this Linux box `dotf doctor --verbose` at HEAD printed `[ OK ] yarn version matches packages.json (1.22.22)`.
- AC4: partially reproducible from here. The box transcript is Windows-only and dated 2026-08-29, i.e. **before** the amendment removed `obsidian`; its `yarn` half was re-demonstrated on Linux at HEAD (doctor row above), the `dotf tools install` skip path is covered by named Go tests (`TestInstallNpm_SkipWhenAtOrAbovePin`, `_UpgradeWhenBelowPin`, `_Fresh`, `_RunFailure`, `_MissingPackage`).
- Ordering claim in "Why" (`dotf tools install` already runs before the deleted blocks) holds on both twins: `setup-linux.sh:293` vs. deleted blocks at 658/752; `setup-windows.ps1:711`, and `tests/setup-windows.bats` asserts Node.js's winget line precedes the `dotf tools install` line — the guard that a fresh Windows box would silently lose yarn if the order ever flipped.
- No `[AGENT-DRAFT]` / `[AGENT-SUGGESTION]` tags remain in any spec file.
- Profile field: `yarn` is declared `"profile": "full"`, but nothing in `cli/internal/tools` or `cli/internal/cmd` filters on `Profile` (grep: the field is only printed by `tools list`), so the move does **not** gate yarn behind a profile the old shell block ignored. Checked because it was the most likely silent regression.

### Findings

| Severity | Reality | Area | Finding | Evidence | Test (named, or UNTESTED) | Fix location (code / tests / spec / vault) |
|---|---|---|---|---|---|---|
| Minor | REAL | tests / AC2 guard | The yarn half of AC2's refute is quoting-sensitive: it matches the literal `npm install -g "yarn` (linux) and `yarnPkg` (windows). Reintroducing the block in any other form is invisible to the suite. Both *documented historical* forms are still caught, which is why this is Minor and not Major. | Mutation (reverted): inserted `npm install -g yarn@1.22.22` into `setup-linux.sh` → `bats … -f 'OPS-042'` 2/2 green, `bash -n` clean, and the f2 command from `features.json` as written still exited 0. | `tests/setup-linux.bats` "parity: yarn is a catalog tool and obsidian is not an npm tool (OPS-042, #1615)" — covers the form, not the property | tests (anchor on `npm install -g` + `yarn` in either order/quoting, or grep `install -g` and exclude the catalog-owned names) |
| Minor | REAL | spec artifacts (contract set) | AC4 and `features.json` f4 were never amended after `52e99a2` dropped `obsidian` from the catalog. They still assert "`dotf tools install` reports **obsidian and yarn** at pin". f4's command uses `grep -qE '^(obsidian\|yarn) .* already installed'`, an alternation that passes on a box where obsidian is deliberately absent — the AC is half-dead and its own command cannot notice. | `proposal.md` AC4 and `features.json` f4 vs. amended AC1 ("declares NO obsidian npm entry"); `jq` at HEAD shows no obsidian entry | UNTESTED (no test asserts f4's wording matches the amended catalog) | spec — **not actionable under this verdict** (contract edits invalidate the review); disposition in `verification.md` / follow-up ticket |
| Minor | REAL | verification evidence | `verification.md` cites commit `a2c8d82` for AC1. It does not resolve in this repo (`git log -1 a2c8d82` → "unknown revision") — it is the pre-squash branch commit, unreachable after squash-merge. The claim survives only through PR #1382. | `git log -1 --format=%H a2c8d82` failed at HEAD | n/a (documentation) | verification.md (outside the contract set, fixable now) |
| Minor | THEORETICAL | doctor semantics / spec drift | `tasks.md` AC3 says the yarn row uses `matchPinFrom` "like copilot and opencode"; the row is now `matchPinFloorFrom` (changed later, `7a81ffe` #1441). Consequence: a box carrying yarn 4.x (corepack/berry) satisfies the 1.22.22 **floor** and doctor PASSES "meets the packages.json pin", although the pin plainly means classic 1.22.x and the two are not interchangeable. Not a regression — the deleted shell block used the same `version_gte` floor — but the catalog schema cannot express a major-version ceiling, so nothing states the intent. | `cli/internal/doctor/checks_tools.go:153`; `matchPinFloorFrom` at `checks_deploy.go:956-966`; `atLeast()` branch | `TestCheckVersionMatch` proves only the below-pin WARN (`yarn --version` = `1.0.0`); the above-pin PASS branch for yarn is UNTESTED | tests + spec (a Go case pinning yarn's floor behaviour, or a line naming the floor in AC3) |
| Minor | REAL | scope honesty | "Two tools never moved" is a census, and censuses rot: `setup-linux.sh:637-639` still installs `@anthropic-ai/claude-code` with a raw `npm install -g` (as an `elif` fallback after the native installer), and pi's block is declared out of scope. Neither appears in the proposal's framing, so a reader finishing OPS-042 would believe ADR-036 is now complete for npm tools. | grep `npm install -g` in `setup-linux.sh`; claude is absent from `packages.json` | `tests/setup-linux.bats` has no ADR-036 completeness guard | spec / follow-up ticket (the correct exit is a ticket, not this diff) |
| Minor | THEORETICAL | verification reproducibility | f4's command is gated on `uname -s` starting `MINGW`, so on Linux/CI it fails at step 1 and verifies nothing while the entry reads `state: "verified"`. A box-only AC is legitimate; a box-only AC whose recorded run predates the amendment is weaker than the `state` field claims. | `features.json` f4; transcript dated 2026-08-29 vs. amendment 2026-09-23 | n/a (manual box evidence) | verification.md (note the re-run that supersedes it, or the Linux repro above) |

Cross-boundary risks I argued against and found clear: fresh-box PATH ordering on both twins (guarded by a named test); profile gating (no filter exists); `installAll` error policy (per-tool best effort, logs and continues, aggregates exit 1 — a 404'd tool cannot block yarn); duplicate-name catalog (`tests/packages-json.bats` "packages.json tool names are unique", `cli/internal/tools/catalog_test.go` rejects a dup); shadow-install (`checkShadowedCatalogTools` WARNs for every npm tool including yarn); the deleted `Test-VersionAtLeast` helper stays for pi; the weakened TEST-003 `Sync-SessionPath` census → floor is justified in-test and the real invariant (`GetEnvironmentVariable("PATH","Machine")` count = 0) is asserted in the same test body, so nothing went quiet by construction.

Mutation summary (all reverted; `git status` clean apart from this file and the launcher's own artifacts): reinstating the obsidian catalog entry turns `tests/setup-linux.bats` OPS-042 red and f1's `jq` fails; deleting the yarn entry turns `TestCheckVersionMatch` red; only the novel-quoting npm block (row 1) slipped through.

### Evaluator rubric

| Dimension | Grade (A-D) | Rationale (one line) |
|---|---|---|
| Correctness | B | All four ACs hold at HEAD and each was reproduced here (catalog shape, `tools list`, doctor row, script refs); gaps are in negative-path *strength* (AC2 quoting) and floor-vs-exact semantics, not behavior. |
| Verification | B | Reproducible commands plus a live doctor run, but AC4's recorded evidence predates the amendment, cites an unresolvable commit, and f4 is unfalsifiable off-Windows. |
| Scope | B | The two commits match the proposal; the accompanying TEST-003 relaxation was forced by the change and is documented in-test; the "two tools never moved" framing overstates ADR-036 completion. |
| Reliability | B | Per-tool best-effort install with aggregated exit, SKIP rows when the binary or catalog is absent, non-fatal in setup; npm-missing loses the old explicit "install Node.js then re-run" hint to a generic warning. |
| Maintainability | A | 123 lines of duplicated per-OS install logic deleted, one installer and one pin SSOT, doctor's yarn row aligned to `catalogPin`, guards assert invariants not populations. |
| Handoff-readiness | B | Amendment, root cause and lesson 285 recorded, CLI-067 ticketed (#1381); two spec artifacts (AC4/f4, the `a2c8d82` ref) left stale. |

Aggregation: no D, no C → all B/A → rubric permits PASS. No Blocker and no REAL Major; every finding is Minor, so the severity path also permits proceeding. Minors are open and tracked → **PASS-WITH-GAPS**.

### Verdict

**PASS WITH GAPS.** `dotf spec archive OPS-042-npm-tools-to-catalog` is **advisable** in the current state: `cli/internal/spec/review.go:39` treats `PASS-WITH-GAPS` as a passing verdict, the frontmatter `spec:` matches the folder name, `reviewed_sha` is the sha I examined, and `reviewer:` is spelled as `harness/reviewer-pool.json` spells it. Do not edit `proposal.md`, `tasks.md` or `features.json` before archiving — that would mark this review stale — so rows 2, 4 and 5 are for disposition, not for a pre-archive contract edit.

### Recommended next steps

Outside the contract set, safe now (disposition each in `verification.md` — applied, ticketed, or declined with a reason):

- **Tests**: strengthen AC2's yarn refute from a literal string to a property — flag a `npm install -g` (or `npm i -g`) naming yarn in either setup twin regardless of quoting or argument order. Name the test that proves it by mutating your own new guard once.
- **verification.md**: replace the unresolvable `a2c8d82` reference with `1478f24` (#1382) and the amendment `52e99a2` (#1616); append the Linux reproduction of the doctor row (`[ OK ] yarn version matches packages.json (1.22.22)`, this worktree, 2026-09-30) next to the August Windows transcript so AC4 does not rest solely on a pre-amendment run.
- **Ticket, don't fix here**: yarn 4.x satisfying a 1.22.22 classic floor — either a Go test naming the intended floor semantics for yarn, or an ADR-036 follow-up giving the catalog schema a major/ceiling field. File it; the current behaviour is inherited, not introduced.
- **Follow-up ticket for the ADR-036 census**: claude-code (`setup-linux.sh:637-639`) and pi are still hand-written npm blocks outside `packages.json`. OPS-042's "Why" reads as if the migration is otherwise complete; the ticket is the honest exit.
- **Carry into the next spec**: f4-style verification commands that begin with a platform `test` are unverifiable where the gate runs. Either mark such a feature `state: manual` or give it a Linux-runnable proxy, so `verified` never means "trusted from a transcript".
