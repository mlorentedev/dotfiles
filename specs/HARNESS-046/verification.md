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

## Review dispositions (agy/gemini-3.1-pro-high, FAIL)

Two earlier launches produced no verdict. `nan/deepseek-v4-flash` was refused with `403 This request is not permitted` (evidence on #1923), and `nan/glm5.3-flash` hit the 45-minute deadline.

- **Blocker, REAL: the guard ran nowhere automatically. Applied.** It moved to `scripts/check-roster-consistency.py`; leaving it in the spec folder would have broken f4 the moment the spec archived. It is wired through two suites. `tests/roster-consistency.bats` runs everywhere, CI included, against a throwaway vault and a stubbed `dotf`: a reordering passes, a missing skill fails naming the role, a duplicated skill fails, and fewer than three skills fails. `tests/roster-consistency-real.bats` is the real-dependency sibling that `stub-real-pairing.bats` requires. It drives the real `dotf harness resolve-skills` over both `skills:` forms, and it runs the guard against the live vault wherever one resolves. Each case skips by name where its dependency is absent. So CI enforces the comparison logic, and the live comparison runs wherever the vault exists. The pre-commit hook runs the legacy `scripts/test.sh` smoke suite, not bats, so this does not run on every commit.
- **Blocker, REAL: strict list equality failed on a reordering. Applied.** The guard now compares `sorted()` lists, a multiset: forced skills are consumed as a set, so order carries no meaning, and sorting still catches an id listed twice. It was red on HEAD for `curator` (same 8 ids, different order) and is green after. Mutation: restoring `!=` turns the reorder fixture and the live case red.
- **Major, REAL: the `AGENTS.md` payload exceeds agy's limit and is truncated. Deferred to #1241.** That is the doctrine-budget arc. HARNESS-084 shipped the `full-only` mechanism for it (#1964), and #1241 stays open for the policy. It is not a change this spec made.

All five `features.json` verifications exit 0 after the fixes (f4 now also runs the fixture suite).

## Review dispositions (round 2, agy/gemini-3.1-pro-high, PASS WITH GAPS)

- **Minor, THEORETICAL, `IndexError` on an `AGENT.md` without frontmatter: declined.** The guard exits non-zero with a traceback that names the line. That is a loud failure, which is what this guard owes; it never reports a clean pass. A friendlier message would change only the wording.
- **Minor, THEORETICAL, malformed roster rows are skipped silently: declined.** A role whose row fails to parse is still reported, from the other side: its invocable definition then has no row, and the guard says `invocable definition has no ROSTER.md row`. The drift is caught, just named from the definition.

## Since this landed

- `hermes-nan` was retired on 2026-09-30 (its next home is decided under kubelab#1933). The catalog entry now carries `status: retired` and still points at `80_agents/hermes-nan/` without duplicating its state, so AC5 holds for the retired record.

## Promotion candidates

- [x] Lesson for the repo's `docs/lessons/`? no: the guard's shape, *check the source of record, never the generated copy*, is recorded above and has one generator so far.
- [x] ADR-worthy decision for the repo's `docs/adr/adr-XXX.md`? no: role definitions inside the existing harness compile contract.
- [x] New pattern candidate for `00_meta/patterns/`? no: a candidate only once a second generator needs the same protection.
