---
tags: [spec, verification, templates]
created: "2026-08-25"
---

# Verification - HARNESS-046

## Evidence

Every criterion was exercised on `msi`, 2026-08-25. Machine-checkable form and per-criterion evidence live in `features.json`; this is the readable summary.

- [x] **AC1 — all seven render** → `compile-harness.sh --refresh` reported seven agent records (`architect`, `builder`, `curator`, `hermes-nan`, `planner`, `reviewer`, `shipper`)
- [x] **AC2 — the generator is idempotent** → second pass byte-identical, `changed=0`
- [x] **AC3 — doctor validates them** → `[ OK ] every declared agent tier resolves for its deploy targets (7 checked)`
- [x] **AC4 — roster and definitions agree, no single-skill wrapper** → `check-roster-consistency.py` exit 0, `6 invocable roles, all >= 3 skills`
- [x] **AC5 — hermes-nan points, never duplicates** → record declares `kind: autonomous` and references `80_agents/hermes-nan/`; none of that tree is copied in

## Test status

- `compile-harness.sh --refresh`, twice: seven records, second pass `changed=0`
- `dotf doctor --verbose` against this tree: agent-tier check `(7 checked)`, all resolve
- `dotf doctor` on main, full sweep: **152 passed, 0 failed, 4 warned, 4 skipped** — no regression
- `check-roster-consistency.py`: exit 0 clean; exit 1 with a planted divergence; exit 0 again after restore

**The AC3 contrast is the part that matters.** The same check reports `(1 checked)` on `main` and `(7 checked)` here. A criterion that passes identically before and after a change has not measured the change — that contrast is what makes this evidence rather than a green line.

## Decisions made during implementation

- **`reviewer` holds no `edit` capability.** A reviewer that fixes what it finds has stopped being independent of it, and independence is the entire value of the role. The asymmetry is deliberate and stated in its boundaries.
- **`reviewer` declares tier `mid`, not `top`.** `mid` is the honest declaration for a subagent deployed into a Claude harness, which cannot run the reviewer pool's models anyway. Model independence for adversarial review is enforced by `harness/reviewer-pool.json`, which excludes Anthropic models by standing rule — a separate mechanism from the tier chain. Nothing reconciles the two declarations; that is recorded as out of scope rather than silently resolved.
- **`read-all-adrs` was added to `architect` on merit, not to satisfy a counter.** It declares itself a mandatory pre-step before `architecture-session`, so the role was under-specified rather than merely short.
- **The consistency guard reads the vault, never the generated records.** Checking the rendered copy against the catalog would pass whenever the generator faithfully rendered a *wrong* definition — which is the failure mode, not the guard.

## Known gap, not a defect of this change

`dotf agent run --role X` does not read these definitions. Nothing under `cli/internal/agent` references `harness/agents/`; `role` is passed through as a string, which is why a dispatch with `--role reviewer` succeeded before any reviewer persona existed. These records deploy as harness subagents and give the doctor tier check something real to validate. **"The personas render and deploy" and "the executor consumes them" are different claims, and only the first is made here.**

## Promotion candidates

- The consistency guard's shape — *check the source of record, never the generated copy* — generalizes past this spec and is a candidate for the shared library if a second generator needs the same protection.
