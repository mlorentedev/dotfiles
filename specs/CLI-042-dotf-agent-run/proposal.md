---
id: "CLI-042-dotf-agent-run"
type: spec
status: draft # draft | implementing | verifying | archived
created: "2026-08-23"
issue: "mlorentedev/dotfiles#1190"   # repo#NNN — GitHub issue / Project item that tracks this spec
tags: [spec, proposal]
template_version: "1.0"
---

# CLI-042-dotf-agent-run

> **Naming**: file lives at `<repo>/specs/CLI-042-dotf-agent-run/proposal.md`. `CLI-042-dotf-agent-run` is `AREA-NNN-slug` (e.g. `TOOL-001-secret-drift`).

## Why

<!-- from issue #1190: CLI-042: dotf agent run — the executor seam ADR-032 defines and nothing implements -->

The orchestration stack has a policy layer and no execution layer. `harness/model-map.json` declares
pools, harnesses, tiers and chains, and `chains` — which resolves at run time and is read only by the
level-2 dispatcher — has **no reader at all**, because `dotf agent` is `unknown command`. Without this,
cross-pool delegation is prose: nothing runs a role on a task in another pool, nothing enforces the
concurrency reserve the map declares, and `pools.deny` is a field no code evaluates. What breaks if we
do not ship it: the map stays a declaration rather than a layer, the six remaining roster personas
would be defined with no consumer, and the concurrency semaphore ADR-035 §3 places at level 2 is
unimplementable because there is no launcher.

## What

`dotf agent run` exists and dispatches. Three observable outputs:

1. **A result record** carrying exit status, output bounded by a cap, and **the pool and model actually
   used** — resolved by walking `chains` for the requested tier through the existing loader
   (`cli/internal/harness/model_map.go`, `ResolveChain` / `ResolveTier` / `DeclaredBudget`), never a
   second parse of the map.
2. **Distinct exit codes for _pool unavailable_ and _task failed_**, so the first advances the chain
   and the second does not. Collapsing them turns a bad answer into a silent retry on another model.
3. **A loud, non-zero refusal** when the concurrency counter cannot be read or the machine's identity
   cannot be established — in the latter case every non-local pool is denied (ADR-032 §8).

Three contract decisions taken deliberately, recorded here so they are not re-derived:

| Decision | Choice | Why |
|---|---|---|
| Dispatch mode | **Synchronous**: blocks, returns R | ADR-032 §2's seam is literally *"run role A on task T, isolated in W, return R"*. Local evidence: `dotf spec review` launches **detached**, which is exactly why GUARD-005 exists — a detached launcher cannot observe its own verdict, so it needed a sidecar plus a downstream assertion. Starting synchronous avoids reopening that hole; `--detach` can be added later, never the reverse. |
| Backend selection | **Probe, with `--backend` as an override** | ADR-032 §7: machine facts are probed, never stored — an inventory file rots, a probe cannot lie. Consistent with how the map already declares `probe: bin:claude` / `env:NAN_API_KEY`. The flag exists to force a backend and to make tests deterministic. |
| stdout | **JSON always**; logs to stderr | The consumer of `agent run` is a dispatcher, not a person — the verb exists to be composed. Precedent: `dotf env path` and the agent-mode contract of #700. |

**Backends: two, and the second one has to be repaired first.** Decided 2026-08-22 after the
measurement in Risks below: a seam validated against a single implementation is a speculative
interface rather than a seam, so hive stays in v1 and hive's own inability to reach a model enters
this spec's critical path rather than being deferred. Two further decisions follow from it:

| Decision | Choice | Why |
|---|---|---|
| Where the fallback lives | **In the dispatcher. hive is single-shot** — it takes one model as a parameter and either answers or fails. | ADR-032 §2 assigns *pool unavailable → advance the chain* to the dispatcher. A backend that falls back internally bypasses that semantic and makes "the pool and model actually used" unreportable, and it is a second routing authority — the "map, never a director" drift ADR-035 §4 exists to kill. `chains.mid` already declares NaN→NaN (`deepseek-v4-flash` → `mimo-v2.5`). Concrete gain: fallback becomes cross-**backend**, not merely cross-model — if NaN is saturated the chain can reach `claude:sonnet` through subprocess, which an internal hive chain could never do. |
| How `dotf` reaches hive | **A dispatch verb on the hive side, launched as a subprocess** — not an MCP client in Go. | The seam's hard semantics (kill on timeout, release the slot without waiting) are exactly what the subprocess backend must already implement for `claude -p` / `pi -p`; the hive verb rides that code path verbatim. One mechanism, N providers. An MCP client would add a Go dependency — itself a Discipline Gate trigger, and stdlib-first says no — a protocol handshake that can drift, and a surface bats cannot smoke. **This does not reduce the backend count**: hive stays a probed, first-class backend; argv is only its transport. |

**hive's worker becomes NaN-only.** Ollama and OpenRouter are dropped rather than deprecated
(owner's decision, 2026-08-22), which aligns the worker with `model-map.json`, whose `$comment`
already records `openrouter` as retired upstream. This makes the repair a *removal plus a re-route*
rather than an addition: hive keeps one transport, the OpenAI-compatible one it already uses for
embeddings and synthesis.

## Out of scope

Things this PR explicitly does NOT include. Forces a sharp boundary and prevents scope creep.

- **The `orca` backend** — *deferred, not rejected*. ADR-032 §2 places three implementations behind
  this seam and they occupy different niches rather than competing: `orca` on an interactive machine
  (managed worktrees, supervised workers, DAGs, ask/reply — the **default executor when present**,
  never the substrate), `subprocess` as the floor bootstrap guarantees, `hive` for headless/CI. Orca
  is out of v1 on three measured blockers: **#899** (the CLI wrapper is unusable from a plain shell —
  the AppImage injects `--no-sandbox` into the electron-as-node path), **#462 / #649** (bootstrap does
  not install it; the installer does not handle `.deb` / AppImage), and **C9** (a GUI desktop
  application is out of scope for headless/CI). The sentence that makes those blockers rather than
  conveniences is ADR-032 §8: *an executor that needs a button pressed in an app is not replicable.*
  The seam is nevertheless designed for three from the start — had Orca been the first backend built,
  the contract would have been shaped around GUI-resident state and the other two would not fit later.
- **Ollama and OpenRouter support in hive** — dropped outright, not deprecated (owner's decision,
  2026-08-22). Consistent with `model-map.json`'s `$comment`, which already records `openrouter` as
  retired upstream in August 2026.
- **A Gemini / `agy` arm inside hive** — unroutable by the declared map: no `gemini` pool exists,
  `agy`'s OAuth-login auth does not fit the `env:` / `bin:` probe shape, and hive's assigned niche is
  headless/CI, where `~/.gemini` does not exist. Reaching Gemini at dispatch level is a **map** change
  (a new pool plus a new auth kind for probes) and gets its own ticket.
- **An MCP client in `dotf`** — deferred with a named trigger: *a second MCP-only backend appears*.
- **The remaining roster personas** (#562). Executor before personas: six speculative definitions
  ahead of their first consumer is the trap this arc already walked into once.
- **`harness/enforced/orchestration.md` and the dispatch hooks** (#561) — held behind #881 until the
  first `enforced` record has been exercised end to end.
- **The `list` and `status` verbs** — they follow the dispatcher, and a synchronous `run` does not need
  them.
- **Adaptive reservation** — ADR-032 §3 fixes `reserve_interactive` as a static number for v1.

## Risks / open questions

Failure modes, dependencies, and unknowns to clarify before implementation. If any item here is unresolved, do not move to `tasks.md` yet.

- **[MUST RESOLVE BEFORE CODE] hive's worker reaches zero models on this machine.** Measured
  2026-08-22 via `mcp__hive__worker_status`: *Ollama: offline / unavailable · OpenRouter: no API key ·
  Available Models: none*. Its provider vocabulary (`auto` / `ollama` / `openrouter-free` /
  `openrouter`) does not intersect the map's pools (`nan`, `claude`, `copilot`, `opencode`), and
  `openrouter` is retired upstream (August 2026, per `model-map.json`'s own `$comment`). So the
  backend cannot consume `chains` and cannot honestly report "the pool and model actually used".
  Sharpest form: every acceptance criterion must be verifiable by test or smoke check, and a backend
  with no reachable provider **cannot pass a smoke test today** — unverifiable by construction, not
  merely risky.
- **[MUST RESOLVE BEFORE CODE] The repair is a change in another repo.** `hive` is
  `mlorentedev/hive` (`~/Projects/hive`), shipped as the `hive-vault` PyPI package and installed as a
  `uv` tool. Fixing the worker is a PR there, gated by its own board, and this spec depends on the
  version that lands. Knowledge placement holds: the hive change is documented in the hive repo, and
  only the seam contract lives here.
- **[RESOLVED 2026-08-22] What the fallback arm actually is — it does not live in hive at all.**
  Recorded because the reasoning is the artifact. Two different fallback shapes already exist in this
  repo, for different reasons, and the resolution was not to pick one but to notice that neither
  belongs inside a backend:

  | Registry | Primary | Fallback | Rationale |
  |---|---|---|---|
  | `.pr_agent.toml` (CI) | `nan/mimo-v2.5` | `nan/deepseek-v4-flash` | **Lane separation.** NaN's limit is per model, so two models are two buckets of five. That file's own comment records that a third-provider tier was *considered and dropped* — no Google credential in the registry, and OpenRouter would spend a finite paid balance per PR. |
  | `harness/reviewer-pool.json` (gate) | `nan/deepseek-v4-flash` | `agy/gemini-3.1-pro-high` | **Provider diversity**, not merely a different model. |

  The Gemini arm is **not reachable over HTTP with a key**: `agy` authenticates by OAuth login
  (`~/.gemini/antigravity-cli/google_credentials.json`, #1156) and the secrets registry has never
  declared `GEMINI_API_KEY`. A Python HTTP client inside hive cannot use that token — reaching Gemini
  would mean hive shelling out to the `agy` binary, as `cli/internal/spec/review_launch.go` does.
  **Resolution: hive is NaN-only and single-shot; the dispatcher owns the chain** (see What). The
  Gemini arm is a map-level question, not a backend-level one.

- **[OPEN — must be answered in `tasks.md`, not in code review] Which backend serves a `nan` chain
  entry when two can.** After the repair, both `subprocess` (via `pi --model <id>`) and `hive` can
  serve `nan:deepseek-v4-flash`. The probe order and the tie-break are a contract decision, not an
  implementation detail: they determine which process consumes the pool's slot and what
  `pool`/`model` reporting means when two transports reach the same model. Proposed default, to be
  confirmed with the task breakdown: **prefer `subprocess` on an interactive machine and `hive` where
  no harness binary is present**, with `--backend` overriding both.

- **[OPEN — SRE, and the sharpest detail of the repair] How `NAN_API_KEY` reaches the hive daemon.**
  With Ollama and OpenRouter gone, the worker needs a NaN credential, and a plaintext key in
  `~/.config/environment.d/` would violate the secrets doctrine (ADR-028: Bitwarden SSOT, `dotf
  secrets run` as the only sanctioned way to hand a secret to a process). The sanctioned shape is the
  **service unit wrapping `dotf secrets run -- hive serve`**, so the value is injected into the child
  process and never lands in a file or a transcript. This is an acceptance criterion below, not a
  deployment note.
- **hive already speaks NaN, just not on the worker path.** `hive/config.py` carries an
  OpenAI-compatible surface — `embed_base_url` / `embed_api_key` / `synth_model` — whose comment reads
  *"default config points at NaN"*, plus `http_timeout` and `tool_timeout` (60s each), so deadline
  machinery exists. On this machine `HIVE_EMBED_BASE_URL` is **unset**: the deployed
  `~/.config/environment.d/10-hive-vault.conf` carries only `HIVE_VAULT_PATH` and `VAULT_PATH`. This
  lowers the repair from "add a provider" to "route the worker through the transport hive already
  has, and deploy its configuration as IaC".
- **`delegate_task` exposes neither `timeout` nor cancellation**, and ADR-032 §2 rules a backend that
  cannot enforce a timeout ineligible. A client-side deadline that kills the process and releases the
  slot satisfies semantics 3–4 *for the dispatcher*, but not for the pool: the remote worker keeps
  consuming a NaN slot the semaphore has already handed back, so the reserve silently over-counts what
  is free. **Resolution: the hive verb must propagate cancellation and terminate the worker**
  (`mlorentedev/hive#384`), and AC3 is written so the weaker form fails the criterion rather than
  redefining it.
- **[RESOLVED 2026-08-22] hive has no dispatch CLI verb.** `hive --help` offers only the stdio MCP
  server, the daemon, the client shim, `service` and `self-upgrade`. Resolution: **add the verb on the
  hive side** rather than an MCP client in `dotf` (see What). Hive's `client` shim already routes to
  the daemon, so the verb is cheap there.

- **ADR-032 §2's backend table is falsified on one row, and the amendment is recorded here rather
  than edited into an accepted ADR** (the ADR-035 convention). The row reads *"`hive.delegate_task` —
  already has fallback chains and a cost cap"*. Both halves no longer hold: the fallback chain points
  at providers that do not answer, and the cost cap is meaningless against a flat subscription, where
  marginal cost is zero and the binding constraint is concurrency.

- **This will exceed the ~300 executable-LOC atomic-PR cap and must ship as a sequence.** Executor,
  semaphore, probes, two backends and their tests do not fit one PR, and "first step of a multi-PR
  sequence" is itself a Discipline Gate trigger. `tasks.md` names the split; no PR in it silently
  absorbs the next.

## Acceptance criteria

Observable outcomes. Each must be testable.

- [ ] **AC1 — the machine contract holds.** `dotf agent run --role <r> --task <t> --tier mid` writes a
      single JSON object to **stdout** carrying at least `status`, `pool`, `model`, `exit`,
      `duration_ms` and `output`, with every log line on **stderr**. Verified by a table-driven Go test
      and a bats smoke that pipes stdout through a JSON parser with stderr attached.
- [ ] **AC2 — the two error classes stay distinct.** A backend reporting *pool unavailable* advances
      to the next `chains` entry for the tier; a backend reporting *task failed* does not, and the
      emitted record names the pool and model that actually answered. Table-driven test over a fake
      backend with scripted responses, including the case where the chain is exhausted.
- [ ] **AC3 — timeout and cancellation are real, and the guarantee is stated at the strength it
      actually holds.** A dispatch that exceeds its timeout kills the worker, releases its semaphore
      slot **without waiting for it**, and reports a non-zero exit with a `timeout` status. Test with
      a fake backend that outlives its deadline, asserting the slot is free before the worker is
      reaped.
      **Backend-specific condition, not a caveat to be dropped:** for `subprocess` the killed process
      *is* the consumer, so the guarantee is exact. For `hive` it is not — killing the local
      `hive delegate` process does not by itself stop a worker running behind the daemon, which would
      keep consuming a NaN slot the semaphore has already released. `hive delegate` must therefore
      **propagate cancellation to the worker and terminate it**; that requirement is stated in
      `mlorentedev/hive#384` and is part of what makes the backend eligible under ADR-032 §2. Until it
      is met, the honest statement is *the slot is released locally and the remote worker may outlive
      it* — and a backend that can only offer that is one whose reserve accounting is a guess. **AC3
      is not satisfied by the weaker form**: if the hive verb ships without cancellation propagation,
      the correct outcome is that hive fails the eligibility test, not that AC3 is reworded down.
- [ ] **AC4 — it fails closed and loudly, never permissively.** With an unreadable concurrency
      counter, `dotf agent run` exits non-zero rather than treating it as zero in use; with a machine
      whose identity cannot be established, every non-local pool is denied. Both cases tested, and
      both messages state the narrow guarantee — *`dotf` alone will never be the cause of exhaustion* —
      rather than promising that exhaustion cannot happen.
- [ ] **AC5 — the top tier never degrades silently.** With `claude:opus` unavailable, a `--tier top`
      dispatch escalates and exits non-zero; it never falls through to a mid-tier model. Tested.
- [ ] **AC6 — the hive backend passes a real smoke check.** `dotf agent run --backend hive --tier mid`
      returns an answer served by NaN and reports `pool=nan` with the model the chain resolved. This is
      the criterion hive's repair exists to unblock, and it is the one that cannot pass today.
- [ ] **AC7 — the NaN credential never lands in a file.** The hive service unit invokes
      `dotf secrets run -- hive serve`, and the deployed `environment.d` fragment contains no
      credential. Asserted by a test reading the rendered unit and grepping the deployed fragment,
      verified by consequence (the daemon answers) rather than by printing the value.
- [ ] **AC8 — the drop of Ollama and OpenRouter is complete, not partial.** No configuration this repo
      deploys still names them for hive's worker, and `ai/agy/mcp_servers.json`'s `HIVE_OLLAMA_ENDPOINT`
      is removed in the same change that removes the provider.
- [ ] **AC9 — "zero reachable models" is caught at diagnostic time, not under dispatch load.**
      `dotf doctor` reports a backend that probes present but can serve nothing, and fails rather than
      passing quietly. This is the incident-to-guard emission for the defect that shaped this spec:
      hive was a declared backend with no reachable provider **for an unknown length of time, and
      nothing noticed** — the check must be one that would go red if the thing it checks broke again.

## References

- Bitácora board: the GitHub issue / Project item tracking this spec (see the `issue:` frontmatter field)
- `docs/adr/adr-032-cross-harness-agent-orchestration.md` — §2 the seam and its five semantics, §3
  `chains` and concurrency, §4 no top-tier fallback, §7 probes and `pools.deny`, §8 replication and
  fail-safe deny
- `docs/adr/adr-035-model-map-routing-registry.md` — §3 budget enforcement levels and fail-closed,
  §4 a declared map, never a director model
- `harness/model-map.json`, `harness/reviewer-pool.json`, `.pr_agent.toml` — the registries this
  consumes or mirrors
- Epic #558; #1124 (the map this consumes); #561, #562, #563 (siblings, out of scope here)
