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
- [x] AC6 — `bash -n` clean; `shellcheck -S warning` clean; `bats tests/*.bats` → **ok=1526 not_ok=7 skipped=76**
- [x] AC7 — `bats tests/guard-doctrine-target-not-deleted.bats` 2/2 `ok`
- [x] AC8 — lessons 256 and 257 written and indexed

### The seven bats failures are pre-existing

Baselined by running the same files on a clean `main` worktree at `3a11d97`:

| Test | On this branch | On clean `main` |
|---|---|---|
| 6 × `tests/install-dotf.bats` | red | **red** — reads the ambient PATH `dotf`, and `install_dotf` refuses to overwrite a `dev` source build |
| `BUG-771: copilot native skills are not deployed…` | red | **red** (`tests/skills-pipeline.bats`, 22 ok / 1 not ok) |

Both are fixture-isolation defects tracked at #1409, independently observed by the session working `feat/persona-skill-enforcement`. The suite went 14 red → 7 red on this branch; the seven that remain are the baseline, not a regression.

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

`setup-linux.sh` 1738 → 1686, `setup-windows.ps1` 2341 → 2301. **4079 → 3987, −92 net**, against 150 raw deletions — the difference is the comments left behind saying what was removed and on what evidence.

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

## Archive checklist

- [ ] `proposal.md` frontmatter set to `status: archived`
- [ ] Folder moved to `specs/archive/OPS-040-dead-migration-purge/`
- [ ] Independent adversarial review passed (`dotf spec review OPS-040-dead-migration-purge`) — the reviewer must not be the implementer
- [ ] #1333 closed by the archiving PR (ADR-018)
