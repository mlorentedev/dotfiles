---
id: "HARNESS-067-model-pin-drift"
type: spec
status: implementing # draft | implementing | verifying | archived
created: "2026-08-27"
issue: "mlorentedev/dotfiles#902"   # repo#NNN — GitHub issue / Project item that tracks this spec
tags: [spec, proposal]
template_version: "1.0"
---

# HARNESS-067-model-pin-drift

> The `created:` date above reads 2026-08-27 for work done on the evening of
> 2026-08-26. `dotf spec init` stamps that field from a UTC clock — **#1225**,
> still open, and left uncorrected here so a second instance of the evidence
> survives the fix.

## Why

A model id is pinned in roughly seven live places and `harness/model-map.json`
binds none of them. The map has a schema, a validating loader and real
compile-time consumers (`ResolveTier`, `dotf agent` dispatch, the doctor tier
check) — what it does not have is any mechanism making a leaf agree with it.

The gap is not theoretical. Measured 2026-08-26:

- **One model, three spellings.** `nan:mimo-v2.5` in the map, `openai/mimo-v2.5`
  in `.pr_agent.toml` (litellm treats NaN as OpenAI-compatible), bare
  `mimo-v2.5` in `ai/pi/models.json`.
- **A dead id in a deployed file.** `~/.pi/agent/settings.json` carries
  `nan/deepseek-v4-flash-0731`, and pi prints
  `Warning: No models match pattern "nan/deepseek-v4-flash-0731"` on *every*
  start. It also carries three `openrouter/*` entries for a provider this repo
  retired in August 2026. Nothing detects either, and nobody noticed until an
  unrelated investigation read the file.
- **The adversarial-review gate's independence arm is unroutable.**
  `harness/reviewer-pool.json` routes `agy/gemini-3.1-pro-high` as its
  provider-diverse fallback. The map declares pools `nan`, `claude`, `copilot`,
  `opencode`, sets `harnesses.agy.pools: ["nan"]`, and the string `gemini`
  appears nowhere in it.

## What

A declared **pin-site registry** plus two detectors, in the layering #1248 just
established:

1. `harness/model-pins.json` — one entry per pin site: the file, which pins in it
   are **routing** pins, and the **normalization** rule for turning that site's
   spelling into a map-resolvable `pool:id`.
2. **Repo pins vs the map** — a Go test over committed files. Runs anywhere,
   including CI.
3. **Deployed drift** — a `dotf doctor` check. Machine state, which CI never sees.

## Out of scope

- **Generation.** #902 asks for it and it is right for the surfaces the pipeline
  owns (`ai/pi/models.json`, `ai/opencode/opencode.jsonc`, the shell rc files).
  Detect first: a generator whose correctness nothing checks is the same class of
  unguarded claim this issue exists to end. Phase 2, recorded on #902.
- **`ai/pi/settings.json` will never be rendered.** It is seed-if-missing because
  pi rewrites it at runtime, a contract #754 established after the previous shape
  reset the user's theme and default model on every setup run. A generated
  `enabledModels` reaches a fresh machine and never an existing one — the
  opposite of the requirement. Reported, never written.
- **Repairing deployed drift.** The doctor check WARNs; there is no `--fix`.
  `~/.pi/agent/settings.json` is pi's own runtime state, and writing into a
  tool-owned file is the same disposition question the extension symlinks raised
  in #1243. It gets asked, not defaulted.
- **`harness/reviewer-pool.json`.** Separate from the map **by decision** — its
  own `$comment` records that it was kept out so H-044 need not migrate it. The
  `agy/gemini-3.1-pro-high` gap above is real, but it is a question about the
  *map's* completeness, not about drift. Filed as **#1253**.
- **`cli/internal/agent/backends.go`.** Go rather than JSON on purpose; its own
  comment says so.

## Risks / open questions

- **Catalog is not routing, and conflating them would fire on a recorded
  decision.** `qwen3.8-flash` and `glm5.3-flash` are in pi's and opencode's
  catalogs and deliberately absent from the map — picker availability on a
  promotional allocation pending a community vote (**#1244**). A guard reading
  "leaf offers a model the map does not know" as drift would flag them.
  **Resolved**: only *routing* pins are checked — `defaultModel`, tier entries,
  `.pr_agent.toml`'s `model` / `model_weak`, the `qq`/`qf` wrapper arguments.
  A catalog superset is never drift.
- **Normalization cannot be inferred.** Three spellings of one model, each
  correct for its consumer. **Resolved**: the rule is declared per pin site in
  the registry, as data, and the check reads it rather than guessing.
- **Rotation costs nothing where a harness self-rotates.** Claude Code resolves
  `opus`/`sonnet`/`haiku` to the current family member, which is why the map
  pins aliases there and canonical ids everywhere else. A check that demanded
  canonical ids uniformly would break that deliberate asymmetry.
- **Open**: whether `HIVE_EMBED_BASE_URL` (#1231's eight-place base URL) belongs
  in this registry or stays a separate axis. It is an endpoint, not a model id.
  Left out of this PR; #1231 stays open.

## Acceptance criteria

- [ ] **AC1** — `harness/model-pins.json` declares every routing pin site, each
      with its normalization rule and a non-empty `why`, and is schema-validated.
- [ ] **AC2** — every routing pin in a committed file resolves to a pool/model
      the map declares, proven by a test that runs in CI.
- [ ] **AC3** — the guard is shown to **fail** on an injected bad pin, so it
      cannot pass vacuously.
- [ ] **AC4** — a catalog entry absent from the map is **not** reported as drift
      (`qwen3.8-flash`, `glm5.3-flash`), and the test asserts that explicitly.
- [ ] **AC5** — `dotf doctor` reports a deployed routing pin that no longer
      resolves, demonstrated against the live `nan/deepseek-v4-flash-0731`.
- [ ] **AC6** — the deployed check reports a pin naming a retired provider
      (`openrouter/*`), distinctly from an unresolvable model id.
- [ ] **AC7** — an unreadable or malformed registry fails loudly and is never
      read as "no pin sites declared" (constraint C15).
- [ ] **AC8** — the doctor check performs no writes, asserted rather than
      assumed.

## References

- Bitácora board: `mlorentedev/dotfiles#902`; consolidates #1231, #1170, #1244.
- The layering precedent: #1248 / AI-030 AC11 — repo-level facts in CI, machine
  state in doctor, because CI never sees this machine.
- The contract this design works around: `ai/pi/README.md`, `tests/pi-config.bats`
  and #754 on seed-if-missing.
- The registry this checks against: `harness/model-map.json`, ADR-032, ADR-035.
- Prior art for "declarative table, one behaviour": `ai/pi/packages.json`
  (AI-030) and `ai/deploy.json` (CLI-039).
