---
id: "GUARD-006-agent-tier-agreement"
type: spec
status: implementing # draft | implementing | verifying | archived
created: "2026-08-22"
issue: "mlorentedev/dotfiles#1164"   # repo#NNN — GitHub issue / Project item that tracks this spec
tags: [spec, proposal, doctor, harness, guard, model-map]
template_version: "1.0"
---

# GUARD-006-agent-tier-agreement

## Why

Two **committed** files can disagree and nothing notices. An agent record under `harness/agents/`
declares a neutral tier (`model: top`), and `harness/model-map.json` decides which harnesses that
tier covers. Nothing compares them.

Before #1165 the disagreement was invisible because `render_agent` dropped the field entirely.
#1165 made it a **hard render failure** — correctly, per C15 — which means a drift that used to be
silent now costs a deploy, and it is discovered on whichever machine happens to run one. The gap is
between two files in the repository; it should be caught in the repository.

## What

`dotf doctor` reports, for every harness `manifest.json`'s `agents.deploy` renders to, whether every
committed agent record's declared tier resolves. A record whose tier the map cannot answer FAILs,
naming the record, the tier and the harness — the same three facts the deploy-time failure names, so
the operator reads the same sentence in both places.

It runs inside the existing *Routing registry* section, after the map is known to load — a check
about a map that could not be read would be noise on top of a failure already reported.

## Out of scope

- **Making `compile-harness.sh --check` do this.** That mode is the offline drift gate and runs in
  the CI `lint` job, which installs no Go and has no `dotf`. Resolving tiers there would report
  drift on a perfectly good record purely because the machine lacks the resolver — conflating a
  property of the deploy **environment** with a property of the committed **record**. `dotf doctor`
  has neither problem.
- **Tier gaps for harnesses nothing deploys to.** `copilot` declares `render: agent-md` with no
  tier; that is a real open question and it is **#1170**, not drift. Reporting it here would train
  the reader to skip this line.
- **The capability half.** `capability-map.json`'s own doctor check rides with #1172, whose registry
  does not exist on `main` yet.
- **`--fix`.** There is no safe automatic repair: the fix is either to declare a tier or to change
  the record, and which one is right is a judgement about intent.

## Risks / open questions

- **A false positive is worse than the gap it closes.** A diagnostic that fires on correct data gets
  skipped, and then the true positive is skipped with it. Two narrowings guard against that: the
  check is scoped to `agents.deploy` targets, and it honours each record's `targets:` list. The
  second is the sharp one — an **absent** `targets:` means *every* harness, and inverting that
  default would make every scoped persona fail against every other harness. It has its own table
  test for exactly that reason.
- **Missing inputs are not this check's failures.** An absent manifest or record dir is
  `checkCompileHarnessDrift`'s diagnosis. Printing two failures for one cause makes both worth less,
  so this one stays silent and returns.
- **The frontmatter reader is deliberately minimal.** It answers only what this check asks
  (single-line values in the first frontmatter block) rather than becoming a second YAML parser
  competing with the render pipeline. A record using block-style YAML is out of its reach — and
  #1172 makes that shape a loud failure in the renderer, which is the right place for it.

## Acceptance criteria

- [x] A record whose tier resolves for every deploy target passes, and the pass line says how many
      pairs were checked.
- [x] A record whose tier one deploy target cannot answer FAILs, naming the record path, the tier
      and the harness.
- [x] A tier no `tiers` block declares at all FAILs the same way.
- [x] A record declaring no tier is not drift and produces no finding.
- [x] A record scoped by `targets:` is judged only against the harnesses it targets.
- [x] An absent `targets:` is treated as every harness (its own test, because inverting it turns
      correct data into noise).
- [x] An absent manifest, an unparseable one, or one with no deploy targets produces no FAIL.
- [x] Verified against the real tree, not only fixtures.

## References

- Bitácora board: mlorentedev/dotfiles#1164
- `specs/archive/HARNESS-076-model-map-tier-render/` — the change that made this drift expensive
- `docs/adr/adr-035-model-map-routing-registry.md` — the registry and its doctor-check precedent
- Related: #1170 (copilot's tier gap, deliberately not reported here), #1172 (the capability half)
