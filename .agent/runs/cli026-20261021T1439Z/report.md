# Run cli026-20261021T1439Z

- Armed: 2026-10-21T14:39Z, expires 22:39Z. Objective: advance CLI-026 (#495) with the Go harness verbs the owner is not needed for.
- Status: in progress (draft written 15:15Z; updated at exit).

## Items (max_items 5, reached)

| # | Branch | PR | State |
|---|---|---|---|
| 1 | feat/cli026-copilot-catalog (slice 2) | #2864 | ready, awaiting review |
| 2 | feat/cli026-harness-agents (slice 4) | — | pushed `5dcadc32`, PR opens after #2864 lands |
| 3 | feat/cli026-harness-doctrine (slice 4) | — | pushed `fb7fcdb6`, stacked on agents |
| 4 | feat/cli026-harness-deploy (slice 5, Go half) | — | pushed `5105456c`, stacked on doctrine |
| 5 | feat/cli026-harness-check (slice 6, Go verb) | — | pushed `9cb7de03`, stacked on deploy |

The branches are stacked; each PR opens only after its predecessor lands, with origin/main merged in (advisor).

## Filed

- #2866: a cancelled duplicate `spec-gate` run blocks the merge state with every check green (owner: option A or B).

## Owner

- Release PR #2853 (0.77.0) is the owner's.
- After a release carries the verbs: slice 3 and the slice 5 rewire (setup scripts, `records-harness` converge), and the slice 6 wiring (`ci.yml`, `dotf doctor` drift check). Those paths are frozen for this run.
