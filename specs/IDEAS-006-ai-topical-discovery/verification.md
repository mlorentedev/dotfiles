---
tags: [spec, verification, ideas-006]
created: "2026-05-25"
---

# Verification - IDEAS-006-ai-topical-discovery

> Status: skeleton. Populated by the implementation PR on the feature branch — OR by the R5 validation finding if the spec is abandoned at gate.

## Evidence

| # | Criterion | Evidence |
|---|---|---|
| 1 | Discovery loop exists in setup-linux.sh | _pending_ |
| 2 | At least 2 agents migrated to install.sh | _pending_ |
| 3 | Bats: dummy installer discovered + invoked | _pending_ |
| 4 | Bats: failure isolation (agent error → log_warn, continue) | _pending_ |
| 5 | Un-migrated agents still work via hardcoded blocks | _pending_ |
| 6 | `ai/README.md` documents convention | _pending_ |
| 7 | CI grep-check: no double-install for any agent | _pending_ |
| 8 | Shellcheck clean on setup-linux.sh + new install.sh files | _pending_ |
| 9 | **R5 validation finding** (required regardless of outcome) | _pending_ |

## Test status

- Bats: `bats tests/ai-topical-discovery.bats` → _pending_
- Shellcheck: `shellcheck setup-linux.sh ai/*/install.sh` → _pending_
- Drift detector: `scripts/drift-detector.sh` → _pending_
- Manual smoke: clean-VM run of setup-linux.sh, both migrated and un-migrated agents install correctly → _pending_

## Decisions made during implementation

_Populated during implementation OR R5 abandonment._

**R5 validation finding (MUST be recorded — captures the cost-benefit of the spec itself):**

- Sample-migrated agent: ?
- LOC delta in setup-linux.sh: -? LOC (block removed) / +? LOC (loop added) / +? LOC (in new install.sh) = NET ?
- Mental clarity gain: subjective; record the user's read after sample migration
- Drift detector impact: ?
- Verdict: PROCEED / ABANDON

If verdict = ABANDON: archive with `--abandoned`, fill criteria with "N/A — spec abandoned at R5 validation gate".

_Other expected decisions during implementation:_

- Agent migration order (alphabetical vs by-complexity).
- Whether failure isolation logs to stderr or to the setup-linux.sh log file.
- Exact format of warning message (`log_warn "Agent <name> install failed (continuing)"`).
- Whether to require `install.sh` to be executable (`chmod +x`) or just present (`bash <file>` works either way).

## Promotion candidates

- [ ] Lesson for `10_projects/dotfiles/90-lessons.md`? **Yes regardless of outcome** — if proceeded: "Topical discovery scoped to one subtree (ai/) is the sane middle ground between hardcoded and full-holman"; if abandoned: "Validate cost-benefit BEFORE refactoring — sometimes the hardcoded version is fine. R5 validation gate pattern."
- [ ] ADR-worthy decision for `10_projects/dotfiles/30-architecture/adr-XXX.md`? **Maybe** — "Why ai/<agent>/ is the ONLY subtree we apply topical discovery to" if proceeded.
- [ ] New pattern candidate for `00_meta/patterns/`? **Maybe** — `pattern-scoped-discovery` or `pattern-validate-before-refactor` depending on outcome.

## Archive checklist

- [ ] `proposal.md` frontmatter set to `status: archived` OR `status: abandoned`
- [ ] Folder moved: `specs/IDEAS-006-ai-topical-discovery/` → `specs/archive/IDEAS-006-ai-topical-discovery/` OR `specs/archive/_abandoned/IDEAS-006-...`
- [ ] Backlog entry in vault `11-tasks.md` ticked (with PR link if archived, with `(abandoned)` marker if not)
- [ ] Promotions above executed (if any)
- [ ] R5 validation finding documented in lessons regardless of outcome
