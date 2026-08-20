---
id: "CLI-039-dotf-deploy"
type: spec
status: archived # draft | implementing | verifying | archived
created: "2026-08-16"
issue: "mlorentedev/dotfiles#1023"
tags: [spec, proposal, cli, deploy, adr-020]
template_version: "1.0"
review: waived
review_waived_reason: "Work shipped and merged under #1023; the issue was then closed by hand, so the archive-on-merge gate (keyed on a PR closing keyword) never saw it and the spec was left active. A retroactive adversarial review cannot gate code already on main, so the waiver is recorded instead of manufacturing one. Backlog reconciliation 2026-08-19."
---

# CLI-039-dotf-deploy

## Why

Deploying an agent config — copy a source, substitute secret placeholders,
compare against the installed copy, install atomically — is implemented **twice**,
once per OS, inside the bootstrap scripts:

| config | `setup-linux.sh` | `setup-windows.ps1` |
|---|---|---|
| MCP servers | ~464 | mirrored |
| `opencode.jsonc` | ~723 | ~1078 |
| pi `models.json` | ~835 | ~1201 |

Two implementations of one behaviour, in two languages, kept in step by hand.
The scripts are 1513 and 2120 lines and grow every time this logic is touched.

ADR-020 C7 leaves shell exactly one job: *"thin bootstrap — detect OS/arch, fetch
the right binary, put it on PATH — plus profile/env."* Substituting secrets into
a config and installing it atomically is none of those; it is user-facing tooling
logic, which the same ADR assigns to Go. The strangler-fig rule is explicit:
**when a `.sh`/`.ps1` twin is next touched, port it to `dotf` in that same PR and
delete the pair.** That has not happened here, so the scripts accrete.

#987 is what surfaced it: the obvious-looking fix was to patch the same block in
both scripts, growing both in the direction this epic exists to reverse.

## What

`dotf deploy [name]` — one implementation, driven by a declarative manifest, in
the idiom this repo already uses for exactly this kind of table (`packages.json`
for tools, `registry.yaml` for secrets, `env-contract.json` for paths):

```jsonc
// ai/deploy.json
{ "version": 1,
  "configs": [
    { "name": "pi", "src": "ai/pi/models.json",
      "dst": "{PI_AGENT_DIR}/models.json", "render": true, "mode": "0600" }
  ]
}
```

- `dotf deploy` deploys every declared config; `dotf deploy pi` just that one.
- Destinations resolve through `dotf env` (ADR-025) — never a hardcoded path.
- Rendering **calls** `dotf secrets render`; it does not reimplement substitution.
- Idempotent: identical content is reported and not rewritten.
- `--dry-run` reports what would change without touching anything.

Both setup scripts lose their pi block and call the command. **They get shorter.**

## Slicing

The atomic-PR cap applies, and there is a seam where every intermediate merge is
functional. **This spec covers slice 1 only.**

1. **(this spec)** the command, the manifest, and **pi**; both setups replace
   their pi block with the call.
2. opencode — gated on an empirical answer to whether opencode resolves
   `{env:VAR}` itself. The setup comment claims both tools self-resolve, and pi's
   identical claim was disproven only by experiment (#987). Not assumed twice.
3. MCP registration, the largest and least uniform block.

## Out of scope

- Slices 2 and 3 above.
- Changing `dotf secrets render`'s behaviour or contract.
- A `setup-macos.sh`. `dotf` ships darwin binaries and this command is
  platform-neutral, but the bootstrap gap is pre-existing (README: planned) and
  is not silently closed here.
- Deploying anything that is not an agent config (symlinks, hooks, RC files).

## Risks / open questions

- **A deployed config is a separate artifact with its own lifecycle** (ADR-030).
  This command writes the deployed copy from the checkout; it must not be
  mistaken for a sync of the reverse direction.
- **Secret-bearing destinations must be 0600.** The mode is declared per config
  rather than inferred, because inferring it means guessing which configs hold
  secrets — the guess that has already gone wrong once (#987).
- **Manifest scope creep.** Five fields, and more only when a second real case
  demands them. Designing for agents that do not exist yet is how a manifest
  becomes a configuration language.
- **Bootstrap ordering.** `dotf deploy` needs `dotf` on PATH, so the call sits
  after the install step, exactly where the current blocks already sit. A machine
  whose `dotf` install failed must still complete setup with a clear message
  rather than a hard stop — the existing blocks degrade that way and so must this.

## Acceptance criteria

- [ ] `dotf deploy pi` performs the full deploy — resolve, render, compare,
      install — on Linux and Windows with identical observable behaviour.
- [ ] Both setup scripts are **net shorter**: the pi block is deleted, not
      wrapped.
- [ ] Destinations resolve through `dotf env`; no hardcoded path.
- [ ] Rendering calls `dotf secrets render`; no second substitution implementation
      exists in the tree.
- [ ] A secret-bearing destination is written 0600, asserted by a test.
- [ ] Idempotent: a second run with no source change reports no change and does
      not rewrite the file (asserted on mtime or content, not just output).
- [ ] `--dry-run` changes nothing on disk.
- [ ] A missing source, an unresolvable destination and a failed render each
      produce a specific error naming the config — never "deploy failed".

## References

- Bitácora board: `mlorentedev/dotfiles#1023`
- [ADR-020](../../docs/adr/adr-020-tooling-cli-go-convergence.md) — the convergence and C7 boundary this implements
- [ADR-025](../../docs/adr/adr-025-cross-machine-paths.md) — path resolution the destinations use
- [ADR-030](../../docs/adr/adr-030-checkout-vs-deployed-copy.md) — the two-artifact lifecycle
- #909 (CLI-035) — port the harness engine; same pattern, same ADR
- #793 (REFACTOR-015) — triplicated hooksPath wiring; same family
- #987 — the change that surfaced this
