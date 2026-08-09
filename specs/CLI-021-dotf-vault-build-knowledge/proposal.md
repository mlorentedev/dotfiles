---
id: "CLI-021-dotf-vault-build-knowledge"
type: spec
status: draft # draft | implementing | verifying | archived
created: "2026-08-08"
issue: "mlorentedev/dotfiles#490"   # repo#NNN — GitHub issue / Project item that tracks this spec
tags: [spec, proposal]
template_version: "1.0"
---

# CLI-021-dotf-vault-build-knowledge

## Why

`dotf vault` today means **one** thing (scaffold a vault entry: `vault project`, `vault work`)
while the *knowledge* half of the same noun — crystallize, maintain, health — lives entirely in
shell twins. Two disjoint meanings under one word is the noun collision this ticket exists to
resolve, and it is why nobody can guess where knowledge maintenance lives.

The cost is no longer theoretical. **BUG-060 (#850)** was a data-corruption defect in
`knowledge-crystallize.{sh,ps1}` that had to be fixed twice, once per twin, because the pair had
already drifted (the `.ps1` broke HARNESS-029 on Windows while Linux was fine). ADR-020 §5 says a
twin gets ported on contact precisely to stop that; the port was deferred to here because #490
sequences it as AUDIT-007 **PR5**.

## What

`dotf vault crystallize`, `dotf vault maintain` and `dotf vault health` exist and are behaviourally
equivalent to the shell scripts they mirror, **built beside them**. Nothing is deleted, nothing is
repointed, no caller changes. After this PR both paths work and produce byte-identical output on
the same input.

Per the parent ADR (`docs/adr/audit-007-cli-convergence-state.md`, row 5): *"Build+test only, no
deletes."* The cutover — repointing callers and deleting the cluster — is **PR7 / CLI-023**.

### Increments

Declared explicitly rather than discovered, in the WEB-019 style, because the three subcommands are
independently shippable and the atomic-PR cap will not hold all of them at once.

1. **`vault crystallize`** — the `MEMORY.md` maintainer. Largest and highest-risk (it writes user
   memory), and the one with a live defect history. Carries the golden-fixture harness the other
   two reuse.
2. **`vault health`** — read-only checks. Smallest, no writes, safest.
3. **`vault maintain`** — the weekly orchestrator (`vault-maintenance-weekly.{sh,ps1}`), which
   composes the other two. Last, because it depends on both.

## Out of scope

- **Deleting or repointing anything.** No twin removed, no caller edited, no doc updated to say
  `dotf vault crystallize` is canonical. That is CLI-023 (PR7). The full inventory of what must
  move then is captured in `tasks.md` §Flip checklist so it is not rediscovered later.
- **The `dotf vault {project,work}` scaffolder.** Untouched; this PR only adds siblings under the
  same noun. Template-surface work is #400 / #461, a different concern under the same word.
- **Changing behaviour.** Any bug found in the shell during porting is reproduced faithfully and
  ticketed separately — a port that "improves while translating" cannot be characterization-tested.

## Risks / open questions

- **The oracle now refuses the wrapped shape.** Since #862 (`9caedc1`) both twins exit 1 on a
  block-scalar `MEMORY.md`. That is the behaviour to port faithfully; the golden corpus therefore
  covers plain-markdown shapes only, with the refusal as its own case.
- **Characterization fidelity is the whole risk.** #672 (CLI-031) requires golden characterization
  tests for every twin port. The two behavioural BATS cases added in BUG-060 are the seed corpus;
  they must be extended to fixtures the *shell* generates, byte-compared against the Go output, not
  merely re-expressed as Go table tests. A port validated only against its own assumptions is how
  silent divergence enters.
- **The `.ps1` is not a faithful twin of the `.sh` today.** BUG-060 found them already divergent.
  So "port the twin" is ambiguous where they disagree — the Linux behaviour is the reference
  (ADR-020 precedent: CLI-024 reconstructs the Linux SUPERSET, not the `.ps1` subset), and any
  `.ps1`-only behaviour must be enumerated before it is dropped.
- **`decode_path` does a filesystem scan under `$HOME` to depth 4-5.** Faithful porting means
  carrying that cost into Go; worth measuring, not silently "optimising" during translation.
- **Windows path encoding is a known trap** — see the #689 regression (drive colon stripped, wrong
  single-dash key). The `.ps1` already defers to `dotf mem project-key` for this; the Go port must
  reuse that same code path, not reimplement it.
- ~~**Open question:** does `vault health` mean the shell's local checks, or the Hive
  `vault_health` MCP tool?~~ **Resolved 2026-08-09: the shell's local checks.** This is a twin
  port, so the oracle has to be the script — #672's golden characterization tests cannot run
  against an MCP surface. Aligning the two notions of "health" is separate work, if wanted at all.

## Decisions taken before implementation (2026-08-09)

- **The flip stays in #492.** Confirmed against this spec's own acceptance and the parent ADR
  row 5, and *not* merged into this ticket. Correcting an earlier claim: this port alone closes
  neither #857 nor #858 — #858 can close here (the fixture-shape inventory is its direction 2),
  #857 closes via #864, and the cutover stays #492's.
- **The wrapped-`MEMORY.md` shape leaves this ticket entirely.** It was measured before any code
  was written (`evidence-yaml-roundtrip.md`): the shape is invalid state the vault's own template
  forbids, produced once by accident across 17 files in vault commit `1c216229` (2026-05-26) and
  never since, and no `yaml.v3` roundtrip can edit it without destroying the markdown hard breaks
  the handoff convention depends on. It is handled by **#864**'s migration plus **#862**'s
  permanent refusal guard — not by CLI capability. Building support would have made this CLI a
  permanent consumer of a shape the vault declares invalid.

## Acceptance criteria

- [ ] `dotf vault {crystallize,maintain,health}` exist with `--help`, and `crystallize` supports
      `--all` and a positional project dir, matching the shell's interface.
- [ ] Golden characterization tests: for a fixture corpus, Go output is byte-identical to the
      shell's on the same input (per #672 / CLI-031).
- [ ] The HARNESS-029 invariant from BUG-060 holds in the Go implementation, proven by a test that
      fails against a deliberately naive append.
- [ ] Table-driven unit tests for the path encode/decode and the section-insertion logic.
- [ ] **No twin deleted, no caller repointed** — `git diff` touches only `cli/` and `specs/`.
- [ ] The shell scripts remain the canonical invocation; docs and skills unchanged.

## References

- Bitácora board: mlorentedev/dotfiles#490
- Parent ADR: `docs/adr/audit-007-cli-convergence-state.md` (row 5, PR5 of 12)
- ADR-020 §5 — strangler-fig on contact; this ticket is where crystallize's obligation lands
- Related: #672 (CLI-031, golden characterization tests), #850 / BUG-060 (the defect that made the
  drift concrete), CLI-023 (PR7, the cutover that deletes the cluster)
- Cross-ref, different surface: #400, #461 (vault template drift)
