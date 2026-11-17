---
id: "AI-054-review-budget-guard"
type: spec
status: archived # draft | implementing | verifying | archived
created: "2026-11-10"
issue: "mlorentedev/dotfiles#3281"   # repo#NNN — GitHub issue / Project item that tracks this spec
tags: [spec, proposal]
template_version: "1.0"
owner: manu
---

# AI-054-review-budget-guard

## Why

Two settings in the pr-agent reviewer are arithmetic that cannot fire: `max_model_tokens` equals the model ceiling it is compared against, so the token clip has no reachable subject, and two of the three review steps declare a ceiling their own call budget can outlast. Both read as protection and assert nothing. dotfiles has survived on the job-level ceiling by accident; the same configuration, ported to `mlorentedev/svqtriana.github.io` without the `CONFIG__AI_TIMEOUT` override, killed run `42562380120` — the job exhausted its budget and published no review. A guard that fails when the configuration stops protecting anything is the fix; the values are only the first thing it finds.

## What

After this change `dotf harness check` fails when either of two invariants stops holding on the reviewer's configuration, and passes on the corrected values:

1. **Token clip.** `max_model_tokens + <largest declared output cap> < custom_model_max_tokens`. The clip must leave room for the answer inside the model window, or it clips nothing. Checked on the file **and** on any step that re-declares the clip in its env, because the workflow's env overrides the file.
2. **Step ceiling.** `chain × attempts × ai_timeout ≤ timeout-minutes × 60`, per review step. A fallback the step cannot afford is not a fallback. `attempts` is 1 when the step declares `CONFIG__RETRY_SAME_MODEL_ON_TIMEOUT: "false"` and 2 otherwise, so the conservative bound applies to any step that leaves the retry at upstream's default. A step that declares no `ai_timeout` is its own finding: it runs the image's default, a number this repository never chose.

**Why the job ceiling is not in the guard.** It needs the probes' seconds and the publication calls' count as well as the pair of steps the route can draw, and modelling that route's `if:` guards from Go would be fragile. The first attempt at it — the two largest step ceilings plus setup — reported a pass at 34 minutes on a job that needs 37, which is precisely the defect this spec exists to remove; a guard that agrees with a wrong ceiling is worse than none. `tests/pr-agent-config.bats` already computes the job's sum exactly, from the probe timeouts where they are set, and **it is what caught the 34-minute ceiling** the moment a step ceiling grew. That test stays the SSOT for the job; this spec raises the number it was failing on.

Measured on the configuration before this change:

| Step | ceiling | chain | attempts | `ai_timeout` | needed | verdict |
|---|---|---|---|---|---|---|
| `PR-Agent on Anthropic, drawn first` (442) | 8 min = 480 s | 1 (`FALLBACK_MODELS: []`) | 2 | 360 | 720 s | **fails, 240 s short** |
| `PR-Agent` (514) | 12 min = 720 s | 3 (model + 2 fallbacks) | 1 | 120 — **not declared** | 360 s | arithmetic passes on an inherited default |
| `PR-Agent on Anthropic, after NaN published nothing` (673) | 8 min = 480 s | 1 | 2 | 360 | 720 s | **fails, 240 s short** |
| job (183) | 34 min | — | — | — | `25 + 5 probes + 3 setup + 3 calls = 36 min` | **fails once the step ceilings are fixed** |

and the token clip: `200000 + 40000 = 240000 > 200000`, **fails**.

The corrected configuration changes no behaviour on any path that succeeds today:

- `max_model_tokens` 200000 → **150000**, so `150000 + 40000 = 190000 < 200000`: the clip now bites before the window, with 10000 to spare. The largest review prompt measured on this repo is ~97000 tokens (glm5.3-flash, AI-045 AC9), so no review that runs today gets clipped. Two alternatives were rejected: **raising `custom_model_max_tokens`** to 260000 is arithmetically fine (inside each model's 1M context, and it matches the workflow's own worst-case cost note) but amends a documented owner decision — the toml says 200000 is "a deliberate floor well inside each, raised only against measured need"; **120000** clips 30000 earlier for no benefit, since the answer cap is already reserved above the clip rather than out of it.
- The two Anthropic ceilings 8 → 13 minutes, so each can afford its own two attempts.
- `CONFIG__AI_TIMEOUT: "120"` stated on the NaN step, whose effective value it already is; an inherited default is the exact defect that killed svqtriana.
- Job ceiling 34 → **38**. Growing a step ceiling spends the job's slack, so the job must grow with it: `25 + 6 + 3 + 3 = 37` (the route may probe two Anthropic models on a risky PR, found by the independent audit), and AI-045's AC7 requires the bound to exceed the sum rather than equal it. 38 is the smallest value that holds.
- `tests/pr-agent-config.bats` updated where it pinned the old numbers: the two Anthropic `timeout-minutes` in the routing test, and the prompt-cap parity — which used to assert the inert `cap == window` and now asserts cap-for-cap across providers plus the room the answer needs.

## Out of scope

- Re-routing the review pool, changing which models are drawn, or the price-step arithmetic in `docs/runbooks/guide-pr-agent-reviewer.md`. The pool is unchanged; only what a step can spend changes.
- Changing `ai_timeout` on any step, or the output cap on any step. Both stay as they are; the NaN step's value is only made explicit.
- The svqtriana repository's copy of the guard. It is already fixed there; this spec ports the invariant here.
- `pr-agent.yml`'s triggers, probes, publication guards, or the review-attestation path.

## Risks / open questions

- **A 13-minute ceiling doubles the wall clock a stuck Anthropic attempt can hold a runner.** Mitigation: a ceiling only binds when something is already wrong, and the measured worst review on the pool is 10.5 minutes; the job ceiling (38) still bounds the pair. The alternative — declaring `retry_same_model_on_timeout: false` to keep 8 — removes a retry that may be absorbing transient Anthropic errors, which is a behaviour change and not this PR's business.
- **`max_model_tokens` is also declared as `CONFIG__MAX_MODEL_TOKENS` in the workflow, which overrides the file.** The guard reads both: a fix that lowers only the toml leaves the run clipped by the number nobody changed. AC2 fails on that case, and a test pins it.
- **`chain` is read as `1 + len(fallback_models)` from `.pr_agent.toml` whenever the step's `CONFIG__FALLBACK_MODELS` is a GitHub expression rather than a literal.** The route can only draw from the toml's pool, so this over-estimates rather than under-estimates — the safe direction.
- **`attempts = 2` for a step that does not declare the retry is a conservative bound, not a measurement.** If upstream's default ever changes, the guard stays safe in the direction that matters (it over-estimates), and this spec records why.
- **The job ceiling is checked by `tests/pr-agent-config.bats`, not by the Go guard, so the two surfaces are not symmetric.** A Go copy of that arithmetic was written and deleted: at one point it computed the job as the two largest step ceilings plus setup, passed at 34 minutes on a job that needs 36, and so blessed the very defect this spec exists to catch. The bats test reads the probe timeouts and the publication count where they are set, so it cannot drift from them; keeping both would also leave two answers one minute apart. AC6 pins the guard's silence on the job so nobody re-adds the weak bound believing it helps.
- **Raising the Anthropic ceiling spends the job's slack, and the slack was already nearly gone.** The pair needs `12 + 13 = 25` of the 34 minutes available; the probes, setup and publication calls take the other 11. The job ceiling had to move in the same change as the step ceiling, which is exactly the coupling that made this the second find.
- **Both files are parsed with the libraries already in `go.mod`** (`go-toml/v2`, `go.yaml.in/yaml/v3`), so a reworded comment or a reordered key cannot silently stop the check matching. What can still rot is the marker that identifies a review step (`uses: docker://pragent/pr-agent`); the guard fails closed when it finds none.

## Acceptance criteria

- [ ] AC1: `dotf harness check` reports a problem when `max_model_tokens + <largest output cap> >= custom_model_max_tokens`, and on any review step that re-declares the clip in its env; the committed `.pr_agent.toml` and every committed review step satisfy it.
- [ ] AC2: `dotf harness check` reports a problem for any review step whose `chain × attempts × ai_timeout` exceeds its `timeout-minutes × 60`, and for any step that declares no `ai_timeout` at all; every committed review step satisfies it.
- [ ] AC3: The job ceiling is raised to cover the fixed step ceilings, under the arithmetic `tests/pr-agent-config.bats` already enforces — the pair plus the probes, the setup and the publication calls — and that test passes at the new value. `dotf harness check` deliberately stays silent on the job; a test pins that silence so a weaker duplicate cannot be reintroduced.
- [ ] AC4: The guard fails closed when the workflow parses but contains no review step, and when either file is present but unparseable. A checkout with neither file reports nothing.
- [ ] AC5: `docs/runbooks/guide-pr-agent-reviewer.md` states the 13-minute Anthropic ceiling, the declared NaN `ai_timeout`, the 150000 token clip and the 38-minute job, replacing the stale 8-minute, 200000 and 31-minute figures.
- [ ] AC6: `tests/pr-agent-config.bats` and the guard agree on every number they both read: the two Anthropic ceilings, the prompt cap, and the room the answer needs inside the window.

## References

- Bitácora board: mlorentedev/dotfiles#3281
- Related ADR: `docs/adr/adr-020-tooling-cli-go-convergence.md` (the guard lives in Go, not as a new `.mjs` twin)
- Related patterns: `00_meta/patterns/pattern-spec-driven-development.md`
- Precedent: `mlorentedev/svqtriana.github.io` run `42562380120`; `docs/lessons/lesson-371-a-paid-model-is-bounded-by-every-layer-that-supplies-a-default.md`
