---
id: "ADR-032-cross-harness-agent-orchestration"
type: adr
status: accepted
owner: manu
date: "2026-08-09"
supersedes: []
extends: [adr-011-model-tier-policy, adr-020-tooling-cli-go-convergence, adr-027-cross-harness-agent-pipeline]
issue: mlorentedev/dotfiles#558
tags: [architecture, decision, agents, orchestration, harness, routing, model-tier, determinism, cross-agent]
created: "2026-08-09"
---

# ADR-032: Cross-harness agent orchestration — two levels, one executor seam, `dotf` as the common tool

> Fills the policy layer ADR-027 deliberately left out: **who decides to fan work out, which pool
> absorbs it, and what stops it from running away**. ADR-027 defined *what an agent is* and how the
> definition reaches each harness; this ADR defines *how work is delegated across harnesses and
> subscriptions*, and amends ADR-027 and ADR-011 where the ground has moved under them.

## Status

Accepted. Lands inside the existing **HARNESS-042** epic (#558) — not a new track.

"Not gated" refers to **acceptance and authoring**: no external precondition blocks recording this
decision or starting work, because the cross-machine deploy gate (`knowledge#120`) that ADR-027
inherited is closed. Individual implementation paths do have dependencies, and they are named in
Implementation — in particular the Orca backend waits on #462, #649 and #899. The subprocess floor
and level-1 native fan-out have no such dependency, which is why the design does not stall behind
them.

## Date

2026-08-09

## Context

Four things changed since ADR-027 was accepted (2026-06-21), and each invalidates an assumption
that shaped it.

**1. Every harness now has native subagent fan-out — Copilot included.** ADR-027 gave Copilot the
degraded `catalog` render kind because it had "no per-agent mechanism". Copilot CLI now ships
custom agents at `.github/agents/*.agent.md`, a `/fleet` command that splits work and dispatches
independent chunks to subagents in parallel (`@agent-name`), and *smarter subagent delegation* —
automatic delegation, at 100% of production traffic from v1.0.42.

| Harness | Native spawn | Parallel fan-out | Auto-delegates |
|---|---|---|---|
| Claude Code | `Task` tool | ~10 concurrent (Fleet mode: tens–hundreds, preview) | no |
| Copilot CLI | `.github/agents/*.agent.md` | `/fleet`, `@agent-name` | **yes, by default** |
| OpenCode | `mode: subagent` | task tool | no |
| pi | subagent extension (#895) | Single / Parallel / Chain | no |
| Codex | agents TOML | — | — |

The posture we want as a default already exists in one harness. The work is bringing the other
three up to it, not inventing it.

**2. The pools are two paid subscriptions, not a cheap tier and an expensive one.** Claude and NaN
are both flat subscriptions, so marginal cost is zero on both and routing between them is never a
cost decision. NaN is a GPU cluster serving open models over an **OpenAI-compatible** API —
*"if something accepts a base URL + API key, it works with NaN"* — which also means **Claude Code
cannot reach NaN without a gateway**, since it speaks the Anthropic Messages format.

The binding constraint is concurrency, not tokens: **60 rpm and 5 concurrent requests**, against
1.5M tokens/min per model. Those 5 slots are already shared by pi's TUI, the `qq` wrapper and
hive's embeddings. Two subscriptions are two independent lanes of parallelism — but the NaN lane
is narrow and shared, and that is a policy input.

**3. Orca is MIT and installable.** `github.com/stablyai/orca` is open source, with verified
install channels per OS already researched in #462 (brew cask, AUR, `.deb`, AppImage, silent
`.exe`), and `dotf tools` already reads a catalog carrying a `profile` field. Lock-out was never
the real risk; the real limit is that Orca is a GUI desktop application, which #462 itself puts
out of scope for headless/CI.

**4. Nobody upstream is building this.** `anthropics/claude-code#67898` — automatic model routing
by task complexity — is an **open feature request**. Of five references audited (Claude Code
Router, LiteLLM, `wshobson/agents` at 203 agents, `VoltAgent` at 154, Orca empirically), all five
treat the fan-out decision as prompting practice or as a proxy concern. None enforces it.

## Decision Drivers

- **Automatable, reproducible, agnostic, SSOT** — the user's four words, and the lens every option
  below was scored against.
- Exploit **both subscriptions plus the corporate Copilot seat** (and Codex if it is ever added)
  from one way of working.
- **Determinism over memory** (HARNESS-053, #816): doctrine no code enforces does not happen.
- **Honest degradation**: a machine that cannot do something says so; it does not silently do less.
- Reuse the ADR-027 engine; add no parallel mechanism.

## Decision

### 1. Two levels of delegation, and only the second one is new

**Level 1 — native, inside the session.** Each harness fans out with its own primitive (`Task`,
`/fleet`, task tool, subagent extension) using **the same rendered definitions**. Zero
infrastructure, identical personas everywhere. This is what makes a Copilot-only work machine a
first-class citizen.

**Level 2 — cross-pool, outside the session.** Reached only when the work needs *another pool*
(using NaN while Claude is busy — both lanes at once) or genuine worktree isolation. This level
exists precisely because Claude cannot speak NaN without a gateway.

Everything below is the machinery of level 2 plus the enforcement that makes level 1 a default
rather than a habit.

### 2. The common tool is `dotf`; Orca is a backend

The executor is a **neutral seam** — *"run role A on task T, isolated in W, return R"* — with
implementations selected by probe:

| Implementation | When | Notes |
|---|---|---|
| `orca` | interactive machine, Orca present | managed worktrees, supervised workers, DAGs, ask/reply |
| `subprocess` | the floor bootstrap guarantees | `pi -p …`, `claude -p …`; thin, always available |
| `hive.delegate_task` | headless / CI / text-only offload | already has fallback chains and a cost cap |

`dotf` owns the seam because it is the one thing already installed on every machine by bootstrap,
it is ours, and ADR-020 already assigns it this class of surface. Orca is the **default executor
when present**, never the substrate — that is what keeps C7 (reversibility) true without giving up
the coordination layer Orca already provides.

**Command surface, and why it is not `dotf harness`.** `dotf harness` is already assigned to
`refresh` / `deploy` / `check` by CLI-026 — the *compile* side, which turns definitions into
rendered artifacts. The executor is the *run* side and takes its own noun: **`dotf agent`**, with
`run` as the only verb v1 needs (`list` / `status` follow the dispatcher). Keeping them separate
is not cosmetic: one is idempotent and offline, the other spawns processes and consumes quota.

The minimal contract, fixed here because the three backends must be interchangeable:

| Element | Decision |
|---|---|
| Request | role, task text, working copy, tier override (optional), pool deny/allow from the machine |
| Result | exit status, captured output bounded by an output cap, the pool and model actually used, tokens/duration where the backend reports them |
| Timeout | required per dispatch; a backend that cannot enforce one is not eligible |
| Cancellation | the dispatcher must be able to abandon a worker and release its semaphore slot without waiting for it |
| Errors | distinguish *pool unavailable* (advance the chain) from *task failed* (do not advance) — collapsing them turns a bad answer into a silent retry on a different model |

**The wire schema, field names and exit codes are deliberately not fixed here** — that belongs to
the implementation spec under #558, not to a decision record. What is fixed is the noun, the
separation from `dotf harness`, and the five semantics above, because a backend that cannot honour
them cannot be plugged in later.

### 3. `harness/model-map.json` — pools, harnesses, tiers, chains

Four blocks, because they have four different consumers:

```jsonc
{
  "pools": {
    "nan":    { "auth": "subscription", "concurrency": 5, "rpm": 60,
                "reserve_interactive": 2,
                "shared_with": ["pi-tui", "qq", "hive-embeddings"],
                "probe": "env:NAN_API_KEY" },
    "claude": { "auth": "subscription", "concurrency": 10, "probe": "bin:claude" },
    "copilot":{ "auth": "seat", "concurrency": "fleet", "probe": "bin:copilot" },
    "opencode":   { "auth": "subscription", "probe": "bin:opencode" },
    "openrouter": { "auth": "payg", "escalation_only": true, "probe": "env:OPENROUTER_API_KEY" }
  },
  "harnesses": {
    "claude":   { "pools": ["claude"],                "render": "agent-md", "spawn": "task" },
    "pi":       { "pools": ["nan"],                   "render": "adapter",  "spawn": "subagent" },
    "opencode": { "pools": ["opencode", "openrouter"],"render": "agent-md", "spawn": "task" },
    "copilot":  { "pools": ["copilot"],               "render": "agent-md", "spawn": "fleet" },
    "codex":    { "pools": ["codex"],                 "render": "agent-yaml", "spawn": "—" }
  },
  "tiers":  { "mid": { "claude": "sonnet", "nan": "deepseek-v4-flash", "opencode": "qwen3.6-plus" } },
  "chains": { "mid": ["claude:sonnet", "nan:deepseek-v4-flash", "opencode:qwen3.6-plus"],
              "top": ["claude:opus"] }
}
```

- **`harnesses` is the binding pi made necessary.** pi is not a pool; it is the harness that
  reaches the NaN pool, and the only one with no declarative agent format — hence
  `render: adapter` (#563). `render` values reuse the manifest's existing render-kind vocabulary
  rather than inventing a second spelling.
- **`tiers` resolves at compile time** and feeds each harness's native render.
- **`chains` resolves at run time** and is read only by the level-2 dispatcher: a Claude `Task`
  cannot fail over to NaN, so cross-pool fallback exists only outside the session.
- **`concurrency` is a semaphore the dispatcher enforces**, not a comment. With
  `reserve_interactive: 2`, a fan-out claims at most 3 of NaN's 5 slots.
  **This bounds `dotf`-dispatched work only, and that limit must not be overstated:** a hand-run
  `qq`, a pi TUI turn or a hive embedding call consumes a slot the semaphore never sees. The
  reserve is therefore a *heuristic that makes starvation unlikely*, not a guarantee that it
  cannot happen. The honest guarantee is narrower and still worth having: **`dotf` alone will
  never be the cause of exhaustion.** Two consequences: a dispatch that hits a 429 treats it as
  *pool unavailable* and advances the chain rather than retrying into the wall, and any real
  guarantee would need the pool's own concurrency accounting, which NaN does not expose.
  **Adaptive reservation is explicitly out of scope** for v1: the reserve is a static number.
- **pi's own default is `qwen3.6`** (`ai/pi/settings.json`) while the mid tier resolves to
  `deepseek-v4-flash`. **The map wins after render**; pi's picker default is a user preference for
  interactive use and is left alone. `glm5.2` is not available on the current subscription and is
  absent from the map.
- **Model identifiers: alias where the harness self-rotates, canonical id where it does not.**
  ADR-011 records literal ids (`claude-opus-4-7`, `claude-sonnet-4-6`) precisely so a provider
  rotation is a one-line overlay edit. Claude Code additionally accepts the aliases
  `opus` / `sonnet` / `haiku`, which resolve to the current member of each family, so **for Claude
  the map emits the alias** and rotation cost drops from one line to zero. Every other pool has no
  alias mechanism and takes the **canonical id** (`deepseek-v4-flash`, `qwen3.6-plus`). The map is
  the place this distinction is declared; a consumer must never have to guess whether a string is
  an alias or an id, so the schema tags it.
- **Embeddings and other non-chat services belong in the map too.** `qwen3-embedding` and `rerank`
  are not chat tiers and do not fit `tiers`/`chains`, but hive's embedding endpoint is one of the
  four provider surfaces §8 makes the map responsible for generating. The schema therefore carries
  a `services` block (`embeddings`, `rerank`, and any future non-chat endpoint) alongside `tiers`;
  without it the generation step in #902 has no source for the value it must write.

### 4. The `top` tier has no fallback, on purpose

NaN's catalog has no opus-class model, so `chains.top` is a chain of one. When the top tier is
exhausted, the dispatcher **queues or escalates to the human — it never silently degrades to a mid
model**. Silent tier degradation is precisely the "mysterious quality drop" ADR-011 was written to
prevent.

### 5. Enforcement: presence and dispatch, never delegation gating

A new `harness/enforced/orchestration.md` record, compiled by the same machinery as
`definition-of-done`. Its emission is deliberately **not uniform**:

| Harness | Presence | Dispatch | Note |
|---|---|---|---|
| Claude | `SessionStart` | command → role | installs the posture |
| pi | `session_start` | `tool_call` | installs the posture |
| OpenCode | `chat.system.transform` | `tool.execute.before` | installs the posture |
| Copilot | — | — | already delegates; the block only declares roster and budget |

**The fan-out decision stays model-executed.** What code guarantees is that the policy is present
every turn, that the roster and budget are real, and that the dispatcher refuses to exceed a
pool's concurrency. Gating progress on "did you delegate?" is rejected: it is unenforceable
without a task classifier, and a false positive blocks the user for no benefit.

### 6. Two schema fields, and a depth limit

```yaml
surface: user | orchestrated | both   # user-facing persona, orchestrated worker, or both
delegates_to: [reviewer, scout]       # roles this persona may spawn; [] = leaf
```

| Field | Claude | OpenCode | Copilot | pi |
|---|---|---|---|---|
| `surface` | listing as a persona | `mode: primary\|subagent\|all` (1:1) | agent auto-selectability | adapter registration |
| `delegates_to` | `tools: Agent(role)` | `permission.task` globs | `@agent-name` availability | adapter allowlist |

Both pass the "who reads it at runtime" test in four harnesses, which is why they belong in the
neutral core and `temperature` / `maxSteps` / `permissionMode` do not.

**Delegation depth is 1 for v1**: orchestrator roles may delegate; delegated workers are always
leaves. Without a depth rule, two non-leaf roles pointing at each other is an infinite loop that
the semaphore only converts into a deadlock. Deeper DAGs are Orca's job at level 2.

**Depth-1 is enforced where a native target exists and declared where it does not** — Claude via
`tools: Agent(role)`, OpenCode via `permission.task` globs, pi via the adapter allowlist. **On
Copilot it is declared, not enforced**: its delegation is model-driven and `.agent.md` has no
field that forbids a subagent from delegating further. Two mitigations, neither pretending to be
the missing one: the deployed worker agents carry an explicit instruction not to delegate, and the
Copilot pool's fan-out is bounded by its own `/fleet` accounting rather than by ours. This is the
honest-degradation rule of C1 applied to safety, and it belongs in the risk column, not hidden in
a uniform-sounding sentence.

### 7. Machine facts are probed, never stored — with one declared exception

`dotf` discovers which pools exist on the machine it is running on. A per-machine inventory file
would rot; a probe cannot lie. **The exception is denial**: a corporate machine may need to forbid
personal pools, declared in `machine.json` as `pools.deny: [claude, nan]`. Deny is evaluated **at
dispatch time on the machine where the work runs**, not at config time — otherwise a level-2
dispatch from a personal session could route work content through a personal pool on behalf of a
denied machine.

### 8. Replication is a first-class requirement, on three axes

The design is only worth having if it survives the three ways this setup actually changes hands.

**Axis 1 — the machine is gone.** Rebuild is `install-dotf` + `setup`, nothing else. Almost all
local state this design creates is *probed*, never written, so there is nothing to back up and
nothing to drift. API keys come back through the secrets tier (ADR-028: Bitwarden SSOT with the
age floor), so the pool probes pass again as soon as secrets materialize. **Acceptance: a fresh
machine reaches an identical orchestration posture with zero manual steps and zero GUI clicks.**

**The one exception is `machine.json`, and it is the security-relevant one.** `pools.deny` lives
there (§7), so a rebuilt corporate machine that has not yet restored it would probe successfully
for personal pools and **default to allowing them** — the failure would be silent and in the
wrong direction. Two rules follow. First, `machine.json` is part of the restored state, not
regenerated from probes, and its restoration is ordered **before** the first dispatch is possible.
Second, denial fails safe: **a machine whose identity cannot be established denies every
non-local pool** until it is declared, so the unknown case degrades to "no cross-pool dispatch"
rather than to "all pools allowed".
That is what makes #899 (broken CLI wrapper) and #462 (Orca not installed by bootstrap) blockers
rather than conveniences — an executor that needs a button pressed in an app is not replicable.

**Axis 2 — a different project.** Deploy surfaces are user-level — `~/.claude/agents/`,
`~/.config/opencode/`, `~/.pi/agent/agents/` and **`~/.copilot/agents/*.agent.md`** — so every repo
inherits the roster, the maps and the doctrine with no per-repo setup. `dotf init` scaffolds
optional repo-level overrides; defaults never require them.

One precedence detail worth deciding deliberately rather than discovering: for Copilot, a
same-named agent in `~/.copilot/agents/` **wins over** the repo's `.github/agents/`. That is the
behaviour we want for a personal roster that must be identical everywhere, but it means a repo
cannot override a personal persona by name — a repo that needs different behaviour must use a
different name. Recorded here so the collision is a choice, not a surprise.

**Axis 3 — a different model provider.** This is the axis the first draft of this decision
underweighted, on the reasoning that NaN is swappable in theory but not in practice. Treating it
as real changes one thing materially: **today a provider's identity lives in four places** —
`harness/model-map.json`, `ai/pi/settings.json` (`enabledModels`), `ai/opencode/opencode.jsonc`
(catalog) and hive's `HIVE_EMBED_BASE_URL`. Four copies is not an SSOT, it is four SSOTs, and a
provider swap would be a four-file archaeology exercise with no guard.

**Decision: `model-map.json` becomes the single source for provider identity and generates the
rest** at compile/deploy time, with a drift assertion (ADR-012 shape) proving the generated
configs still match the map. Swapping provider then means editing one file and re-running setup.
The neutral definitions are untouched by a provider swap by construction — they carry a tier, not
a model id — which is exactly why the tier abstraction earns its keep even though the provider
"was never going to change".

This generation step is new scope on top of #560 and is called out in Implementation below.

## Amendment to ADR-027

1. **Copilot is promoted** from the `catalog` render kind to a full `agent-md` target
   (`.github/agents/<name>.agent.md`). Emit `model:` as a **string, never an array** — VS Code
   accepts arrays, the Copilot CLI rejects them, and one definition must load in both.
2. **The neutral schema gains `surface` and `delegates_to`** (§6).
3. **`model-map.json` gains the `pools` and `harnesses` blocks** (§3). ADR-027 §2 specified a
   tier→id map; that is necessary and not sufficient once more than one pool is reachable.
4. ADR-027 §5's "pi is first-class via one reusable adapter" is unchanged and reinforced: the
   `harnesses` block names `adapter` as pi's render kind.

**Migration of the contracts these amendments change.** Both are additive, and the defaults are
chosen so nothing breaks between the amendment landing and the roster being rewritten:

| Change | Existing state | Rule |
|---|---|---|
| `surface` added | `curator` (the only definition today) declares none | absent ⇒ `both`, which is today's effective behaviour; the schema keeps it optional until the roster (#562) sets it explicitly |
| `delegates_to` added | none declared | absent ⇒ `[]` (leaf). **Fails closed**: an un-migrated definition cannot delegate, so the depth rule holds during migration rather than after it |
| Copilot `catalog` → `agent-md` | Copilot receives an injected catalog block today | the catalog injection is removed **in the same change** that starts writing `~/.copilot/agents/`, never both at once; the harness deploy's prune path (#843) removes the stale block |
| `model-map.json` appears | tier is dropped at render today | until the map exists, render behaviour is unchanged; the map is additive and the drift gate only starts asserting once it ships |

No migration script is needed: one definition exists, and the fail-closed defaults mean the
transitional state is safe rather than merely brief.

## Addendum to ADR-011

ADR-011 §2 forbids auto-switching models — "propose, don't force" — on two rationales: bill shock
and silent capability drops. Both were written about **the interactive session's own model**, and
that prohibition stands unchanged.

**Scope clarification:** an orchestrator selecting a tier and a pool for a *delegated* subtask,
inside a declared budget, is a different plane. The bill-shock rationale does not apply — both
pools are flat subscriptions and per-token spend (OpenRouter) is escalation-only, never a default.
The silent-quality-drop rationale **does** apply and is honoured by §4: the top tier never
degrades silently.

## Options Considered

| Option | C1 neutral | C3 default-on | C7 reversible | C8 zero marginal | C9 headless | C11 bootstrap | C13 least-priv |
|---|---|---|---|---|---|---|---|
| **A** Gateway first (CCR / LiteLLM) | partial — needs base-URL override | partial — routes models, does not fan out | **no** — proxy in the hot path | **no** — takes Claude off plan | yes | partial | neutral |
| **B** Orca mandatory | yes | partial | yes (MIT) | yes | **no** — GUI only | partial — #649, #899 | no, today |
| **C** Executor seam (chosen) | yes | yes, via `enforced` | yes | yes | yes — floor + hive | yes — floor is guaranteed | yes, via capability-map |
| **D** Null: finish ADR-027, leave fan-out to prose | yes | **no** | yes | yes | yes | yes | yes |

Discarded alternatives and their reasons are recorded in the vault session protocol's rejection
list (gateway-as-default, Orca-mandatory, `model:` on skill frontmatter, a seventh orchestrator
persona).

## Consequences

**Positive**

- One way of working across Claude, Copilot, pi, OpenCode and Codex, with the same personas.
- Both subscriptions become two parallel lanes instead of an either/or.
- The corporate Copilot machine is first-class, and its policy limits are declarable.
- Runaway fan-out has two real brakes — the concurrency semaphore and depth-1 delegation.
- No proxy, no per-token spend, no vendor substrate on the default path.

**Negative / debt**

- `dotf` grows an executor surface with three backends; the seam is a new drift axis.
- The `enforced` emission is non-uniform (three harnesses install a posture, one already has it) —
  a maintenance asymmetry that must be documented rather than smoothed over.
- Concurrency accounting is only as good as the probe; a pool consumed outside `dotf`
  (a hand-run `qq`) is invisible to the semaphore. The static `reserve_interactive` is the
  mitigation, and it is a guess until measured.
- Routing quality remains unmeasured until #539 (eval harness) exists.

**Neutral**

- The gateway hatch (CCR / LiteLLM) stays documented but unbuilt.
- NaN's remote MCP server (web search on the same key) is an orthogonal capability add.

## Implementation

Inside epic #558, in this order:

1. **#560 extended** — `model-map.json` with `pools` / `harnesses` / `tiers` / `chains` plus
   `capability-map.json`; the semaphore lands with the dispatcher, not before it.
1b. **Provider-identity generation (new, from §8 axis 3)** — the map generates
   `ai/pi/settings.json` `enabledModels`, the opencode catalog and hive's embed base URL, with a
   drift assertion on the generated files. Until this exists, "one file to swap provider" is a
   claim, not a property. Needs its own ticket under #558.
2. **#563** — Copilot promoted to `agent-md`; the pi adapter; #895 folded in as its empirical input.
3. **#562** — the roster, now carrying `surface` and `delegates_to`.
4. **`enforced/orchestration.md`** + dispatch hooks (#561). **Held behind #881.** That issue's
   audit found the `harness/enforced/` layer carries six rule ids and not one skill, and set a
   hold: do not add enforcement layers until the first one has been exercised end to end.
   Orchestration would be the seventh id, so it inherits the hold. As of 2026-08-09 the gate
   #881 waits on has still produced **no** `review.md` artifact, so the hold is live — the
   tickets closing (#875, #879) is not the same as the mechanism running.

   The record is also **written pointer-shaped** (~500 characters: the trigger, the two hard
   boundaries, and a link to the `dispatching-parallel-agents` skill for the detail), because the
   injected set is 2,532 characters against a GOV-004 (#673) target of ~9,000 for the whole
   `AGENTS.md` — 28% already, and 41% with a `definition-of-done`-sized addition. The character
   budget proposed to bound that is folded into #881 rather than tracked here, since it is the
   same governance question from the cost side.

   Nothing else in this ADR depends on step 4: levels 1 and 2, the maps and the schema all ship
   without it. What is lost while it is held is only the *default-on* property (C3) — the
   capability works, it just has to be asked for. That is the honest trade, and it is preferable
   to shipping a seventh record onto a layer with no evidence it changes behaviour.
5. **#462** is reprioritized from optional to prerequisite for the Orca backend, with **#649**
   (installer must handle `.deb` / AppImage / installers, not just binaries) and **#899** (the CLI
   wrapper is broken outside Orca's own terminals) as named blockers.

Per C10, each step ships its guard in the same PR: schema validation for the new fields, a drift
assertion on the committed render, and a probe test for the semaphore.

## References

- Extends: ADR-027 (cross-harness agent pipeline), ADR-011 (model tier policy), ADR-020 (`dotf`
  absorbs tooling surfaces). Related: ADR-025 (cross-machine paths), ADR-013 (deploy engine).
- Epic #558; tickets #560, #561, #562, #563, #895, #462, #649, #899, #539.
- External references audited 2026-08-09: Claude Code Router; LiteLLM proxy and auto-routing;
  `wshobson/agents` (203 subagents, per-harness transpilation); `VoltAgent/awesome-claude-code-subagents`
  (154 subagents, tier-by-complexity); Orca CLI 1.4.177 (empirical, this machine);
  `anthropics/claude-code#67898` (automatic complexity routing — open upstream).
- Constraint table and rejection list: vault `10_projects/dotfiles/30-architecture/session-protocol.md`.
