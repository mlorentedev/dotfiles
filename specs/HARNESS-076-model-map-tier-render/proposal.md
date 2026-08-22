---
id: "HARNESS-076-model-map-tier-render"
type: spec
status: implementing # draft | implementing | verifying | archived
created: "2026-08-22"
issue: "mlorentedev/dotfiles#1161"   # repo#NNN — GitHub issue / Project item that tracks this spec
tags: [spec, proposal, harness, orchestration, routing, model-map]
template_version: "1.0"
---

# HARNESS-076-model-map-tier-render

## Why

`harness/model-map.json` merged as a declaration nobody reads. Measured 2026-08-21: `harness.ResolveTier`
and `harness.ResolveChain` have zero callers outside tests, and the registry's only reader is `dotf doctor`,
which reports on it and routes nothing with it. One layer down, the same failure repeats concretely —
`harness/agents/curator/AGENT.md` declares `model: top`, `render_agent()` drops the field, and the deployed
`~/.claude/agents/curator.md` carries no `model:` at all, so the roster's only populated persona runs on
whatever model the invoking session happens to be on. `harness/agent-frontmatter.schema.json` already
promises the opposite behaviour in prose, which makes this a contract the repo states and does not keep.

## What

The neutral tier a persona declares reaches the file the harness actually loads.

`dotf harness resolve-tier <tier> --harness <name>` answers which model id a neutral tier means for one
harness, reading `harness/model-map.json` through the validated loader. `render_agent()` in
`scripts/compile-harness.sh` calls it and emits `model: <resolved id>` into the rendered frontmatter instead
of dropping the field. After this change `model: top` on `curator` deploys as `model: opus`.

Failure splits by cause, and the split is the design:

- **The map cannot answer** — an undeclared tier, a harness with no entry in it, or a map that is absent,
  non-JSON or schema-invalid. The **agent render fails, naming the harness and the tier**. This is C15,
  which governs a map that cannot be *read*: it must never resolve to empty and never to a permissive
  default.
- **The resolver is unavailable** — no `dotf` on PATH, or a `dotf` predating this subcommand. The render
  **warns loudly and emits no model line**. This is not C15: an absent binary is a bootstrap state, not an
  unreadable map. `setup-linux.sh` installs `dotf` best-effort (`install_dotf || log_warning`), so treating
  this as fatal would promote a warned-past dependency into a hard prerequisite of the entire harness
  deploy. The degraded output is exactly what the script produced before this change, so the path is never
  worse than the status quo — and it still *says so*, which is ADR-032's honest-degradation bar.

Telling those two apart needs a **capability probe**, not an exit status. Measured 2026-08-21: a `dotf`
predating the subcommand answers with `unknown flag: --harness` and exit 1, identical in status to a
genuine routing refusal (#1158). The render therefore greps `dotf harness --help` for the subcommand name,
and a Go test pins that string so the probe cannot rot into always-answering "too old".

## Out of scope

- `chains` and the run-time dispatcher. ADR-035 splits consumption into two cadences over one file; this is
  the compile-time half only. `dotf agent run` stays an unknown command.
- Budget enforcement. `DeclaredBudget` still enforces nothing; level 2 remains deferred.
- `harness/capability-map.json` and the capability half of the render (#560, #1163).
- Fixing the `agy` render/`$comment` contradiction (#1162). This spec surfaces it and leaves it declared.
- Adding tier entries for harnesses that lack them. The map is read as it is, not edited.

**One declared exception to the scope.** `ai/claude/settings.json` gains `"outputStyle": "Concise"`,
a change the repo owner made and asked to carry in this PR. It is unrelated to the tier render and
is recorded here so the diff is self-explaining rather than looking like drift. Reviewing it
surfaced a real defect and its fix, which *is* in scope for a settings change: the
`merge_claude_settings` policy is an explicit **allow-list**, so a key added to the template and
not named there is a silent no-op on every existing installation — verified on this machine, where
the deployed `~/.claude/settings.json` had no `outputStyle` at all. Both `setup-linux.sh` and
`setup-windows.ps1` now name it, and a test asserts every dotfiles-owned scalar key appears in both
policies so the next one cannot fail silently.

## Risks / open questions

- **This makes `compile-harness.sh` depend on `dotf` for the first time.** The script had zero `dotf`
  invocations. **Decision: accept the dependency, and make its absence non-fatal** (see What, above). The
  rejected alternative was parsing the map with `jq` in shell: it duplicates the resolution rules in a
  second place and bypasses the schema validation entirely, so a map the Go validator rejects would render
  clean — trading a loud, narrow failure for a silent, broad one. **Reversible at one seam** if the reviewer
  disagrees: `resolve_model_tier` is a single function.
  *Resolved during implementation, by evidence.* The first cut made a missing `dotf` fatal and broke 17
  tests in `tests/skills-pipeline.bats`, which drive the real deploy — because the `dotf` deployed on the
  dev box predates the subcommand (#1158). That is the exact scenario the fatal branch would have inflicted
  on any under-provisioned machine, so it was narrowed rather than tolerated.
- **The fail-loud path is unreachable through the current deploy set, by construction.** `manifest.json`'s
  `agents.deploy` holds one entry (`claude` / `agent-md` / `.claude/agents`) and `claude` is the only
  harness with all three tiers declared. It becomes reachable the moment a harness without a tier is added,
  which is exactly when it should fire. The tests cover the path directly rather than waiting for a deploy
  to reach it.
- **Only `claude` consumes a rendered `model:` field.** `opencode` declares `mid` only, and `copilot`, `agy`
  and `pi` declare no tier at all. This spec does not widen `agents.deploy`, so that asymmetry is recorded,
  not resolved.
- **A record's tier and the map can still disagree in the repository**, and only a deploy notices. `--check`
  is deliberately not the place: it runs in the CI `lint` job, which installs no Go, so resolving there
  would report drift on a good record whenever the machine lacks `dotf`. Filed as **#1164** for a
  `dotf doctor` check, which already loads the map and by definition runs where `dotf` exists.

## Acceptance criteria

- [x] `dotf harness resolve-tier top --harness claude` prints `opus` and exits 0.
- [x] `dotf harness resolve-tier top --harness copilot` exits non-zero with a message naming both the tier
      and the harness, and prints no model id on stdout.
- [x] `dotf harness resolve-tier` fails non-zero when `harness/model-map.json` is absent or schema-invalid,
      rather than resolving to a default (C15).
- [x] `harness/agents/curator/AGENT.md` declaring `model: top` renders to a `curator.md` whose frontmatter
      carries `model: opus`, alongside the existing `name`, `description` and `generated_*` keys.
- [x] An agent record whose tier the map cannot answer fails the agent render with a non-zero exit; the
      failure names the harness and the tier, and the previous definition is left intact rather than
      truncated.
- [x] An absent or too-old `dotf` warns, naming the tier and the harness, and renders without a model line
      rather than failing the harness deploy.
- [x] Skill deploy still succeeds when the agent render fails.
- [x] `render_agent` keeps dropping `kind`, `capabilities`, `skills` and `targets` — only `model` changes.
- [x] bats covers the render, the unresolvable path and both unavailable-resolver shapes; Go table tests
      cover the subcommand; a real-binary sibling suite covers the capability probe and the stdout contract.

## References

- Bitácora board: mlorentedev/dotfiles#1161
- `docs/adr/adr-035-model-map-routing-registry.md` — the two consumer classes and the level-1/level-2 split
- `docs/adr/adr-027-cross-harness-agent-pipeline.md` §2 — the render-time map this implements
- `docs/adr/adr-032-cross-harness-agent-orchestration.md` §3 — the registry's contents; honest degradation
- `docs/adr/adr-020-tooling-cli-go-convergence.md` — why the resolver is Go and the caller is shell
- `specs/archive/HARNESS-075-model-map-routing-registry/` — the registry this consumes
- Related: #1162 (agy render contradiction), #1163 (#560 board drift), #560 (capability-map)
