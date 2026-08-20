---
tags: [spec, tasks, ideas-006]
created: "2026-05-25"
---

# Tasks - IDEAS-006-ai-topical-discovery

> TDD order. One task = one focused commit. Tick as you go.
> **CRITICAL**: the first task is a validation gate (per R5). Do NOT proceed past Setup until R5 is resolved.

## Setup

- [ ] Branch created from main: `refactor/IDEAS-006-ai-topical-discovery`
- [ ] `proposal.md` is complete and acceptance criteria are testable
- [ ] **R5 validation gate**: sample-implement migration of ONE agent (recommend `claude` or `opencode`). Measure: LOC delta, mental clarity gain, drift detector impact. Record findings in this tasks.md AND in verification.md "Decisions". If win is marginal → archive spec as `abandoned`. If win is meaningful → continue.

## Implementation

> Only enter this phase if R5 validation gate passed.

- [ ] Write failing bats `tests/ai-topical-discovery.bats` #1: dummy `ai/_dummy/install.sh` touches a sentinel file. Run setup-linux.sh in sandbox → sentinel exists.
- [ ] Implement the `for installer in ai/*/install.sh` loop in `setup-linux.sh`. #1 passes.
- [ ] Test #2 (failure isolation): dummy `ai/_failer/install.sh` exits 1. Run setup-linux.sh → completes, log shows warning, OTHER agents still install.
- [ ] Implement `|| log_warn` failure-isolation wrapper. #2 passes.
- [ ] Migrate Agent A (e.g., `claude`): create `ai/claude/install.sh` with the logic currently in setup-linux.sh's `# Setup claude` block. Delete the hardcoded block in the SAME commit.
- [ ] Migrate Agent B (e.g., `opencode`): same pattern.
- [ ] CI guard test #3: for each `ai/*/install.sh`, no matching hardcoded block exists in setup-linux.sh (grep-based double-install guard).
- [ ] Update `ai/README.md` with the convention: idempotency, failure isolation, when to add install.sh.
- [ ] Drift detector run post-deploy: exit 0 expected.
- [ ] Shellcheck clean on setup-linux.sh + new install.sh files.

## Closing

- [ ] Every acceptance criterion from `proposal.md` is covered by at least one test
- [ ] features.json contains a row per criterion
- [ ] Lint clean
- [ ] No unrelated changes in the diff
- [ ] `verification.md` filled in (including R5 validation finding)
- [ ] PR opened referencing this spec folder
- [ ] If abandoned at R5: archive with `--abandoned` flag, fill verification.md with the cost-benefit finding, do NOT open implementation PR

## Machine-readable features

Drop the following into `<repo>/specs/IDEAS-006-ai-topical-discovery/features.json`:

```json
[
  {
    "id": "IDEAS-006-ai-topical-discovery-f1",
    "behavior": "setup-linux.sh discovers and invokes ai/*/install.sh",
    "verification": "bats tests/ai-topical-discovery.bats --filter 'discovery'",
    "state": "pending",
    "evidence": ""
  },
  {
    "id": "IDEAS-006-ai-topical-discovery-f2",
    "behavior": "Failure isolation: agent install error doesn't abort setup",
    "verification": "bats tests/ai-topical-discovery.bats --filter 'failure isolation'",
    "state": "pending",
    "evidence": ""
  },
  {
    "id": "IDEAS-006-ai-topical-discovery-f3",
    "behavior": "At least 2 existing agents migrated to install.sh pattern",
    "verification": "test $(ls ai/*/install.sh 2>/dev/null | wc -l) -ge 2",
    "state": "pending",
    "evidence": ""
  },
  {
    "id": "IDEAS-006-ai-topical-discovery-f4",
    "behavior": "No double-install: no agent has both install.sh AND a hardcoded block",
    "verification": "scripts/check-no-double-install.sh",
    "state": "pending",
    "evidence": ""
  },
  {
    "id": "IDEAS-006-ai-topical-discovery-f5",
    "behavior": "ai/README.md documents the convention",
    "verification": "grep -q 'install.sh.*idempotent' ai/README.md",
    "state": "pending",
    "evidence": ""
  },
  {
    "id": "IDEAS-006-ai-topical-discovery-f6",
    "behavior": "Drift detector clean after migration",
    "verification": "scripts/drift-detector.sh",
    "state": "pending",
    "evidence": ""
  }
]
```
