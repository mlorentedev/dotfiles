---
id: "AI-046-pi-nan-provider"
type: spec
status: draft # draft | implementing | verifying | archived
created: "2026-09-26"
issue: "mlorentedev/dotfiles#1764"   # repo#NNN — GitHub issue / Project item that tracks this spec
tags: [spec, proposal, pi, nan]
template_version: "1.0"
---

# AI-046-pi-nan-provider

## Why

pi reaches NaN through a provider we maintain by hand in `ai/pi/models.json`: six model ids, their context windows, and `compat` flags. That list drifts from NaN's catalog. #1772 found four context windows disagreeing with opencode, `glm5.3` and `mimo-v2.6-flash` are missing, and nothing fails when NaN retires an id. `@gtrabanco/pi-nan-provider`, from a NaN community member, registers the same provider from a build-time snapshot merged with NaN's live `/models`. It also fixes two failures we would otherwise hit: a cross-model reasoning replay that overflows `qwen3.6`'s 262K window when you switch models, and truncated streams that stall with no error.

## What

After this change, the package owns pi's NaN provider, and every machine gets it the way it gets every other pi package:

- `ai/pi/packages.json` declares `npm:@gtrabanco/pi-nan-provider@0.7.0`, pinned like every entry there. `dotf pi packages apply` installs it.
- `ai/pi/models.json` no longer lists NaN models. At most it keeps per-model overrides the package documents as composable (pi `docs/models.md`, Per-model Overrides), and only when a measured need exists.
- The package's media MCP bridge (`npx -y nan-mcp-server@<version>`) is off by default through managed config (`NAN_MEDIA_MCP=0`). It spawns a second npm package at runtime, outside the pinned manifest. Turning it on is a later, separate decision.
- `NAN_API_KEY` reaches pi the way it does today, from the environment through the secrets facade (ADR-028). The package's key file and `/login` paths are not used.
- `ai/pi/settings.json` `enabledModels` and the README model list keep naming only ids NaN serves, now checked against the package's catalog.

## Out of scope

- `/nan-usage`. It reads a NaN CLI session token from `~/.config/nan/session.json`, which is a second credential outside `secrets/registry.yaml`. Quota visibility is AI-047 (#1766).
- The official NaN MCP bridge (`NAN_MCP_TOOLS`), and the media bridge beyond keeping it off.
- opencode's NaN provider, which stays hand-maintained. #1772's parity test keeps the two agents' context windows honest while it does.
- Moving any binding to `mimo-v2.6-flash`. It failed admission (#1763).

## Risks / open questions

- **Owner decision: who owns the `nan` provider id.** The package registers `id: "nan"`, the same id `ai/pi/models.json` defines. Options:
  - **(A) The package owns it, and models.json keeps only overrides. Recommended.** The live catalog ends the drift class, and the package's reasoning guard and retry only act on requests to the providers it registers.
  - (B) models.json keeps the full provider and the package is installed for its extras only. What pi does when two sources define one id is unmeasured. A shadowed provider would silently drop the guard.
  - (C) Do not adopt; keep hand-maintaining.

  Before implementing either A or B, measure: `pi --list-models nan` on pi 0.87.1, with both present and with each one alone.
- **Supply chain.** The package runs with full system access inside an agent that holds `NAN_API_KEY`, like every pi package (see the `packages.json` header). The pin plus review on every bump is the same control the other nine entries get. The runtime `npx` spawn is the part the pin does not cover, hence off.
- **pi version.** The peer range is `>=0.83.0 <1`, but the truncated-stream fix targets 0.87. #1774 raises the floor to 0.87.1.
- **Blast radius of the first apply.** `dotf pi packages apply` converges in both directions. On msi, the first apply after #1755 also removes pi-memory (HARNESS-139 AC8). Peers are told before and after; the apply is not run from this branch.
- **Live `/models` at startup.** If NaN is down, the package falls back to its snapshot, so pi still starts. Verify this with the network blocked, not by reading.

## Acceptance criteria

- [ ] AC1: `ai/pi/packages.json` declares `npm:@gtrabanco/pi-nan-provider@0.7.0`, and `tests/pi-config.bats` (the pinned-entry contract) passes.
- [ ] AC2: With the package installed and `ai/pi/models.json` deployed, `pi --list-models nan` lists every id in `ai/pi/settings.json` `enabledModels`, and `glm5.3`. The output is recorded in `verification.md`.
- [ ] AC3: `ai/pi/models.json` defines no NaN model the package already registers. A test fails if one comes back.
- [ ] AC4: `NAN_MEDIA_MCP=0` reaches pi on every managed launch path, and a test fails if the default flips.
- [ ] AC5: With `api.nan.builders` unreachable, pi starts and lists the snapshot models. Measured and recorded.
- [ ] AC6: A model switch from `deepseek-v4-flash` to `qwen3.6` in a session whose history exceeds 262K tokens including reasoning completes without a context overflow. This is the guard the package exists for. Measured once and recorded.

## References

- Issue: mlorentedev/dotfiles#1764
- Package: https://github.com/gtrabanco/pi-nan-provider (0.7.0, MIT)
- `ai/pi/packages.json` header: why every entry is pinned, and how apply converges
- ADR-028 (secrets facade); #1772 (opencode/pi context parity test); #1774 (pi floor 0.87.1); #1763 (mimo-v2.6 admission); #1766 (quota visibility)
