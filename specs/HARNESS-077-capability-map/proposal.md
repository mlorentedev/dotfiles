---
id: "HARNESS-077-capability-map"
type: spec
status: implementing # draft | implementing | verifying | archived
created: "2026-08-22"
issue: "mlorentedev/dotfiles#560"   # repo#NNN — GitHub issue / Project item that tracks this spec
tags: [spec, proposal, harness, agents, capabilities, agnosticism]
template_version: "1.0"
---

# HARNESS-077-capability-map

## Why

`harness/agents/curator/AGENT.md` declares `capabilities: [read, search, edit, shell]` and the
deployed `~/.claude/agents/curator.md` carries none of it — `render_agent` drops the field. That is
*exactly* the state `model:` was in before #1165: a neutral declaration nothing reads, promised by
`harness/agent-frontmatter.schema.json` and not kept.

It is also the half of ADR-027 §2's agnosticism pair that never shipped. #1165 built the compile-time
consumer for the model tier; this is the same seam, one field over. Until it exists, a persona's
declared capabilities are decoration, and every harness gets whatever tool access it defaults to.

## What

The neutral capability vocabulary a persona declares reaches the file the harness actually loads.

`harness/capability-map.json` maps each neutral capability to each harness's native representation.
`dotf harness resolve-capabilities <cap>[,<cap>…] --harness <name>` answers with that harness's
rendered value, read through a validated loader. `render_agent` emits it, exactly as it now emits
the resolved `model:` line.

**The map's value is per-harness and shaped for that harness, because the natives genuinely differ.**
Verified 2026-08-22 against the shipping tools, not assumed:

| Harness | Field | Shape | Semantics |
|---|---|---|---|
| `claude` | `tools:` | comma-separated native tool names — `Read, Glob, Grep, Bash, Edit` | **allow-list**: a tool not named is unavailable |
| `opencode` | `permission:` | object keyed `read`/`edit`/`glob`/`grep`/`list`/`bash`/`webfetch`/`websearch`/… with values `ask`\|`allow`\|`deny` | **decision map**: an unnamed key falls back to the config default |

`opencode`'s `tools` field exists but its own schema marks it *"@deprecated Use 'permission' field
instead"*, so mapping onto it would ship a deprecation on day one.

**That asymmetry is the whole reason this is a map and not a rename.** Naming a capability in an
allow-list grants it and denies the rest; naming it in a decision map grants it and says nothing
about the rest. So the map declares the *complete* native value for a capability set, per harness,
rather than a per-capability token the render concatenates blindly.

## Out of scope

- Harnesses whose native representation is not verifiable from this machine. `copilot` is not
  installed (#1170 already holds its model-tier half); `pi` and `agy` are `adapter` harnesses that
  take runtime flags, not rendered definitions (#1162). Declaring guessed native names would put a
  route to nowhere into the map that validates cleanly — the class ADR-035 deleted the phantom
  `codex` pool to avoid.
- Changing the neutral vocabulary itself. `[read, search, edit, shell, web]` comes from
  `harness/agent-frontmatter.schema.json` and is taken as given.
- Runtime permission enforcement. This renders a declaration; nothing here checks what a running
  agent actually does.
- `chains`, the dispatcher, `dotf agent run`. Untouched, still level-2 work.
- **The `dotf doctor` check over this registry**, and #1164's check that a record's declared tier
  agrees with `model-map.json`. Both are diagnostics over registries rather than parts of the
  render, they live in the same `cli/internal/doctor/` neighbourhood, and folding them in pushed
  this diff past ADR-017's ~300-line atomic cap. They ship together as the immediate follow-up.
- **Unifying the schema custom-rule plumbing.** `model-map`'s `customRules` is a package-level
  closed set bound to one schema, so a second registry cannot declare an `x-` rule without a
  refactor of a file hardened over six adversarial review rounds. This registry's one cross-block
  invariant runs in its loader instead, and the refactor is filed separately so it is deliberate
  rather than a side effect.

## Risks / open questions

- **`opencode` has no agent-definition deploy target today.** `manifest.json`'s `agents.deploy`
  holds one entry (`claude`), and opencode receives only a presence-injected instructions file. So
  the opencode column is declared and unexercised by any deploy until that set grows. It is
  included anyway because the whole complaint against the current state is that the pipe is neutral
  by mechanism and Claude-only by data — declaring a second harness with a *verified* native shape
  is the cheapest honest step against that, and it is what makes the per-harness-shape decision
  above concrete rather than theoretical.
- **An allow-list that is wrong denies silently.** If `read` maps to `Read` but the persona also
  needs `Glob`, the deployed Claude agent simply cannot glob, with no error anywhere. This is the
  inverse of the model failure #1165 fixed (which was loud once resolved) and it argues for
  generous, explicitly-reasoned capability groupings rather than minimal ones.
- **Failure semantics are inherited, not re-decided.** #1165 established the split and it applies
  unchanged: an unmappable capability or an unreadable map **fails the render** (C15); an absent or
  too-old `dotf` **warns and degrades** to the prior behaviour, because that is a bootstrap state.
  The capability probe from #1165 (`dotf harness --help`) generalises — it must name this
  subcommand too, pinned from the Go side.

## Acceptance criteria

- [x] `dotf harness resolve-capabilities read,search,edit,shell --harness claude` prints a
      `tools:`-ready value naming the native Claude tools, exit 0, nothing on stderr.
- [x] The same for `--harness opencode` prints its `permission:` object form.
- [x] An undeclared capability exits non-zero naming the capability and the harness, and prints
      nothing to stdout.
- [x] A harness the map does not cover exits non-zero naming it, rather than resolving to empty.
- [x] An absent or schema-invalid `harness/capability-map.json` fails rather than defaulting (C15).
- [x] `curator` declaring `capabilities: [read, search, edit, shell]` deploys a `curator.md` whose
      frontmatter carries the resolved `tools:` line alongside `name`, `description`, `model` and
      `generated_*`.
- [x] A record declaring no `capabilities` renders without the field, unchanged.
- [x] An absent or too-old `dotf` warns and renders without the field rather than failing the
      harness deploy, matching #1165.
- [x] Go table tests for the resolver, bats for the render and both degradation shapes, and a
      real-binary case in `tests/compile-harness-real.bats`.

## References

- Bitácora board: mlorentedev/dotfiles#560 (re-scoped to this half by ADR-035 decision 1)
- `docs/adr/adr-027-cross-harness-agent-pipeline.md` §2 — the agnosticism pair this completes
- `docs/adr/adr-035-model-map-routing-registry.md` — shape C (`$comment` + `version` + schema +
  doctor check), the precedent this follows
- `specs/archive/HARNESS-076-model-map-tier-render/` — the seam this mirrors, one field over
- Native formats verified 2026-08-22: Claude Code agent frontmatter `tools:` (installed plugin
  agents); opencode `permission:` enum `ask|allow|deny` and `tools` deprecation
  (https://opencode.ai/config.json)
- Related: #1170 (copilot's model half, same not-verifiable-here reason), #1162 (adapter harnesses)
