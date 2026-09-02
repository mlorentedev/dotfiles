---
tags: [spec, verification, templates]
created: "2026-09-01"
---

# Verification - OPS-040-dead-migration-purge

## Evidence

Every command below was executed on this branch; nothing here is asserted from reading.

- [x] AC1 — no deploy-time secret fetch in either script → `bats tests/setup-windows.bats -f 'neither setup resolves a deploy-time secret'` → `ok 1`
- [x] AC2 — blocks 2–7 absent → loop over the nine executable forms (`export ANTIGRAVITY_ENDPOINT=`, `SetEnvironmentVariable("CLOUDCODE_URL"`, `Removing orphan master from pre-SDD-007 path`, `Removed legacy GEMINI.md`, `rm -f "$HOME/.claude/init-project.sh"`, `Creating python symlink`, `-OutFile $bunInstaller`, …) → `OK`
- [x] AC3 — blocks 8–9 absent from `setup-linux.sh` → same shape → `OK`
- [x] AC4 — HIVE-118 and MEM-002 intact, rc exports intact → `bats tests/guard-no-claude-mem.bats` 3/3 `ok`, including *"the MEM-002 cleanup block IS present in both setup scripts (anti-regression)"*
- [x] AC5 — C15 skip → `ok 1 … # skip no 'dotf secrets show <id>' call sites in the tree — nothing to resolve (the shell/ps1 sweep found none)`
- [x] AC6 — `bash -n` clean; `shellcheck -S warning` clean; `bats tests/*.bats` → TAP plan `1..1535`, **ok=1528 (77 of them skips), not_ok=7**

> **Corrected after the independent review (finding F1).** This line first read `ok=1526 not_ok=7 skipped=76`, which sums to 1609 against a plan of 1535. The counting `awk` matched `/# skip/` on lines that also matched `/^ok /`, so every skip was counted twice and the pass count was correspondingly off. The material claim — seven failures, all pre-existing — was unaffected and the reviewer reproduced it independently at the true merge base `16d8f96`, which is a stricter baseline than the `3a11d97` used here.
>
> **On AC6's own wording**, also per the review (F4): "passes under both bash and zsh" is not a discriminating clause, because bats re-execs its own bash and the invoking shell does not change test semantics — one run is what exists and one run is what is recorded. And **PSScriptAnalyzer was never run locally** (no `pwsh` on this box; those cases skip). It is enforced by CI at `.github/workflows/ci.yml` `lint-powershell`, which went green on #1433. That is the evidence for the PowerShell half, and it is CI's, not this session's.
- [x] AC7 — `bats tests/guard-doctrine-target-not-deleted.bats` 2/2 `ok`
- [x] AC8 — lessons 256 and 257 written and indexed

### The seven bats failures are pre-existing

Baselined by running the same files on a clean `main` worktree at `3a11d97`:

| Test | On this branch | On clean `main` |
|---|---|---|
| 6 × `tests/install-dotf.bats` | red | **red** — reads the ambient PATH `dotf`, and `install_dotf` refuses to overwrite a `dev` source build |
| `BUG-771: copilot native skills are not deployed…` | red | **red** (`tests/skills-pipeline.bats`, 22 ok / 1 not ok) |

Both are fixture-isolation defects tracked at #1409, independently observed by the session working `feat/persona-skill-enforcement`. The suite went 14 red → 7 red on this branch; the seven that remain are the baseline, not a regression.

### Where the endpoint variables actually live (repo-wide sweep)

The first pass grepped only the two setup scripts and called block 2 inert. A repo-wide sweep corrected that:

```
.zshrc:42-43                          export ANTIGRAVITY_ENDPOINT / CLOUDCODE_URL   ← the live source, untouched
tests/antigravity.bats:84             pins that .zshrc export                        ← untouched, still green
cli/internal/doctor/checks_deploy.go:801
                                      sys.env("ANTIGRAVITY_ENDPOINT", "https://cloudcode-pa.googleapis.com")
.bashrc                               (no export — pre-existing parity gap, not in scope)
```

`dotf doctor` defaults to production when the variable is unset, inside a section that SKIPs when `agy` is absent, so the check reads identically before and after this change. `.zshrc` continues to export both, so a running shell is unaffected. **The deletion is a no-op outside the setup process**, which is the correct scope for it — but the reason is narrower than "these are inert", and the spec now says so.

### Fail-first proof for the new guard

Ran `tests/guard-doctrine-target-not-deleted.bats` against a scratch tree holding `main`'s setup scripts plus the shipped manifest:

```
not ok 1 guard: no setup script deletes a harness doctrine deploy target
# expected NOT to find /(rm[[:space:]]+-[rf]+|Remove-Item)[^|;]*GEMINI.md/ in .../setup-linux.sh, but it is there:
# 503:    rm -f "$GEMINI_HOME/GEMINI.md"
```

Green on this tree. The guard fails for the right reason, at the right line.

### Shellcheck

`shellcheck -f gcc setup-linux.sh` → **16 findings**, against **19** on `main`. All are `info` (SC1091 / SC2015 / SC2016) and all pre-existing; `-S warning` is silent. No finding is introduced by this change.

### LOC

`setup-linux.sh` 1738 → 1686 (−52), `setup-windows.ps1` 2341 → **2305** (−36, numstat 17+/53−). **4079 → 3991, −88 net.**

> **Corrected after the independent review (finding F1).** This first read `2301` and `−92`. The Windows figure was measured before two later edits added explanatory comments to that file — the claude-mem reword and the CLI-020 note — and the stale number was then carried into the commit message, the PR body and the handoff without being re-measured. Re-measured here with `wc -l` and `git diff --numstat 16d8f96...HEAD`. The direction and the reason are unchanged: the gap between 88 net and 150 raw deletions is the comments left behind recording what was removed and on what evidence.

## Decisions made during implementation

- **The ticket's framing was wrong and is replaced, not executed.** "Delete ~200 lines of finished migrations" became a per-block classification by skip-cost. Of eleven blocks, one had never worked (#1431), one cannot be shown converged on Windows (HIVE-118), and one was destroying a live harness deploy target on every run (block 4).
- **Block 4's deletion is a bug fix and carries a guard.** `harness/manifest.json` declares `.gemini/GEMINI.md` as agy's doctrine target with an append-in-place contract; both setups `rm -f`'d it, and a bats test pinned that behaviour. Only ordering hid the loss.
- **MEM-002 excluded whole, not partially.** Its fix needs a `claude plugin marketplace remove` that may not exist, a state file the repo has never touched, and a Windows leg. Coupling unverified new behaviour to a deletion PR would make both harder to review.
- **The rc `BUN_INSTALL` exports stay**, against the ticket. `bun` is installed on msi at `~/.bun/bin/bun`; deleting the PATH export removes a working binary from the user's shell. Zero consumers *in the repo* is not zero consumers *on the machine*.
- **Retired tests are replaced by comments, never by absence-assertions.** Pinning a deletion is not an invariant.
- **No `# EXPIRES:` CI check.** The convention is written into lesson 257; enforcing it is construction, and adding an expiry-less migration inside this PR would refute its own lesson. Owner decision, raised in the PR body.

## Promotion candidates

- [x] Lesson for `docs/lessons/` — **two**: 256 (a cleanup block's description of what it deletes is not evidence) and 257 (a migration with no removal date is indistinguishable from live code).
- [ ] ADR-worthy? No. Nothing here changes a structural decision.
- [ ] New pattern for `00_meta/patterns/`? Candidate: "classify a cleanup by skip-cost, then probe its target" — hold until a second instance appears.

## Independent review outcome

**PASS** — `nan/glm5.3-flash`, drawn at random from `harness/reviewer-pool.json`, at `reviewed_sha` `d0c8b16`. Rubric: Scope A, Reliability A, Maintainability A, Correctness B, Verification B, Handoff-readiness B. No Blocker, no Major.

The reviewer verified by execution rather than reading: it re-ran the `features.json` commands, re-derived the seven-failure baseline at the true merge base, and reproduced the guard fail-first with **three** mutations — including `rm -f "$HOME/.codex/AGENTS.md"`, the *second* manifest target, which proves the target list is manifest-driven rather than a hardcoded `GEMINI.md`. That is a stronger proof than the one in this file.

Six findings, all Minor or Question, dispositioned:

| # | Finding | Disposition |
|---|---|---|
| F1 | Recorded LOC and suite tally do not reconcile | **Fixed above.** Both were mine and both were wrong; corrections carry the reason. |
| F2 | `rm -f "$GEMINI_HOME/config/.migrated"` (`setup-linux.sh:463`) left unclassified — one reference repo-wide, no producer, consumer or expiry | **Ticketed: #1438.** A ready-made instance of lesson 257, and pre-existing rather than introduced here. |
| F3 | Guard matches only two removal spellings; `rm --force`, `unlink`, pwsh aliases evade it | **Ticketed: #1439.** THEORETICAL — no such form exists in the tree; the manifest wiring is proven. |
| F4 | AC6's "both bash and zsh" cannot fail as written, and PSScriptAnalyzer is CI-only | **Fixed above**, in the AC6 note. |
| F5 | Blocks 3/5 labelled "dead" where the operative reason is "leftovers are inert" | **Accepted, not edited.** The label is imprecise and the outcome is right; `proposal.md` is staleness-watched, so editing it post-review would invalidate this review for a wording fix. Recorded here instead. |
| F6 | `tasks.md` "PR opened" unchecked while #1433 is merged | **Resolved by accepting #1433 as the referenced PR**, per the reviewer's own recommendation. Not ticked — that edit would post-date the review and re-trigger staleness. |

Mid-review, `origin/main` advanced to `06c3b7a` as #1433 squash-merged. The reviewer verified the merged state is content-identical (`git diff origin/main HEAD` empty, both setup-script blobs matching) and recorded that the review remains valid for it.

## Archive checklist

- [x] Independent adversarial review passed — reviewer `nan/glm5.3-flash`, not the implementer
- [ ] `proposal.md` frontmatter set to `status: archived`
- [ ] Folder moved to `specs/archive/OPS-040-dead-migration-purge/`
- [ ] #1333 closed by the archiving PR (ADR-018)
