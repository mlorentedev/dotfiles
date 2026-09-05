---
id: "CLI-063-dotf-deploy-claude"
type: spec
status: draft # draft | implementing | verifying | archived
created: "2026-09-04"
issue: "mlorentedev/dotfiles#1339"   # repo#NNN — GitHub issue / Project item that tracks this spec
tags: [spec, proposal]
template_version: "1.0"
---

# CLI-063-dotf-deploy-claude

## Why

The Claude-side deploy — `.claude.json` snapshot guard, MCP registration, plugin installation,
`settings.json` merge — is implemented twice, in jq and in PowerShell. **Claude is the only agent
still deployed that way.** `ai/deploy.json` already routes `pi`, `orca-keybindings`,
`copilot-settings`, `copilot-config`, `copilot-mcp` and `agy-settings` through `dotf deploy`; the
four Claude capabilities are the last twins of their class.

The part that hurts is not the line count. It is that the merge policy is **data wearing code's
clothing**. `merge_claude_settings()` decides per key whether the template wins, the objects merge,
or the arrays union. That policy exists in three places: a prose comment in `setup-linux.sh`, a
prose comment in `setup-windows.ps1`, and two implementations meant to agree with both. Adding a key
to `ai/claude/settings.json` requires editing all three, and `setup-linux.sh` records what happens
when someone does not — `outputStyle` sat in the template while the deployed file had none,
*"silently a no-op on every existing installation, reaching only machines bootstrapped from
scratch"*, found by measuring this repo's own box rather than by any test.

**Measured 2026-09-05 against `main` @ `568d73b`** (the line numbers in #1339's body are from
2026-08-27 and have moved; these are located by content):

| Capability | `setup-linux.sh` | `setup-windows.ps1` |
|---|---|---|
| Snapshot guard | `snapshot_claude_json`, `restore_claude_json_if_truncated` | `Backup-AndRestoreClaudeJson` |
| MCP registration | the `claude mcp add` block | its PowerShell twin |
| Plugin list + install | hardcoded id list + install loop | hardcoded id list + install loop |
| `settings.json` merge | `merge_claude_settings()`, ~100 lines of jq | its PowerShell twin |

**The twins are already behaviourally divergent**, which is what makes "port the twin" ambiguous
rather than mechanical: `setup-windows.ps1` increments `$pluginsAdded` outside the success check and
swallows the failure, so a plugin that failed to install is reported as added (**#1491**). No test
pins the counter on either side.

## What

**Extend `dotf deploy` to cover Claude**, rather than build a new command. Nothing repointed,
nothing deleted, no caller edited: after this work both paths exist and agree.

### Why not `dotf agent claude sync`, as #1339 proposes

Three of that framing's premises are false, and each was checked against the code rather than the
ticket:

1. **`dotf agent` is the wrong noun, by an accepted ADR.** ADR-032:122-124 fixes the boundary:
   *"`dotf harness` is already assigned to `refresh` / `deploy` / `check` — the compile side… The
   executor is the run side and takes its own noun: `dotf agent`… Keeping them separate is not
   cosmetic: one is idempotent and offline, the other spawns processes and consumes quota."* Config
   sync is idempotent and offline. It belongs on the side ADR-032 assigns to deploy, and the ADR
   states that *the noun* is the part it fixes. That `dotf agent` already exists is not a licence to
   use it — it is the reason the boundary needed writing down.
2. **"`dotf deploy` has no merge kind" is false.** `strategy: merge` has existed since AI-039
   (#1322) and is in production on three configs. Its documented semantics are precisely the policy
   Claude needs at the top level: *"a managed key is owned by the repo, whole value, and
   `dotf deploy <name>` puts it back when the tool changed it; an unmanaged key is the box's."*
3. **This work is already on `dotf deploy`'s roadmap.** `ai/deploy.json`'s own `$comment` says:
   *"opencode and the MCP registration blocks are deliberately absent — they are slices 2 and 3."*
   CLI-063 is the continuation of CLI-039, not a new surface beside it.

A separate `dotf agent claude sync` would build a **second, Claude-only deploy path parallel to the
one every other agent already uses** — the twin shape ADR-020 exists to collapse, rebuilt under a
noun reserved for something else.

### What is genuinely missing, measured

`mergeInto` (`cli/internal/deploy/deploy.go:433`) writes the source's **top-level** keys into the
destination. Claude's six policies split cleanly against that:

| Key | Policy in `merge_claude_settings()` | Covered by today's `strategy: merge`? |
|---|---|---|
| `attribution` | whole object, template wins | **yes** |
| `autoCompactEnabled` | template wins | **yes** |
| `precomputeCompactionEnabled` | template wins | **yes** |
| `env` | **nested object merge** (`(.env // {}) + $tmpl.env`) | no — top-level replace loses the box's keys |
| `enabledPlugins` | **nested object merge** | no — same |
| `permissions.allow` | **array union, deduped** (`| unique`) | no — top-level replace drops the box's entries |

So the deliverable is not "port 100 lines of jq". It is: **three per-key policies the deploy
manifest cannot yet express**, plus two capabilities (`plugins`, the snapshot guard) that are not
file installs at all. That is a much smaller and much better-defined change than #1339 assumed, and
it generalises — `env`-style nested merge is not Claude-specific.

### The SSOT files

- **`ai/claude/plugins.json`** — new. The plugin ids currently hardcoded in both twins.
- **`mcp-servers.json`** — already the SSOT and already shared; read unchanged.
- **`ai/claude/settings.json`** — already the template. The per-key policy moves from two prose
  comments into the manifest, declared per key.

### Increments

1. **Snapshot guard + plugin sync.** Smallest coherent unit, and where the duplicated *data* is
   worst. The guard's threshold already exists in Go — `cli/internal/mem/session_start_adapter.go:83`,
   `claude_json_min_bytes` defaulting to 10240, the same constant SDD-021's canary uses for the same
   upstream bug (anthropics/claude-code#59870). Reuse it; do not restate it.
2. **MCP registration.** Slice 3 as the manifest already names it. Reads `mcp-servers.json`, skips
   entries `claude mcp get` already knows, surfaces `add` errors rather than swallowing them.
   Carries HIVE-118's remove-then-re-add migration for the stale `uvx hive-vault` entry.
3. **Per-key merge policy.** Extend the manifest's strategy vocabulary with the three policies above
   and declare Claude's `settings.json` against it.

The cutover is separate, and that split is not ceremony: #1450 asked for a deletion whose ticket
turned out to be wrong about what was duplicated, and what worked there was **port, prove parity,
then delete**. The pinning bats tests named in #1339's fix are the only thing currently asserting
the shell behaviour the port must reproduce, so they are the *last* thing to go.

## Out of scope

- **Deleting or repointing anything.** No twin removed, no caller edited, no doc changed to say the
  Go path is canonical.
- **`hooks`.** HARNESS-045 AC1 moved hooks out of `merge_claude_settings` entirely — `dotf harness
  bind` owns them, emitted from `harness/manifest.json`, and a bats guard refuses the literal jq
  assignment because the old one *replaced the whole SessionStart array and deleted a live
  third-party group*, measured 2026-08-27. **The port must not gain a second hooks writer.** This is
  the single most likely way for this work to cause an incident.
- **Fixing #1491 while translating.** Linux is the reference (ADR-020; the CLI-024 and CLI-021
  precedents), so the Go path counts successes and the `.ps1` behaviour is a defect closed
  deliberately with the divergence recorded — never absorbed silently into a translation.
- **The MEM-002 claude-mem retirement block.** A one-time migration living adjacent to this code,
  not part of it; #1431 owns it.
- **Retro-fitting the new per-key policies onto copilot/agy.** They may benefit; changing a
  production config's merge semantics is its own change with its own blast radius.

## Risks / open questions

- **How per-key policy is DECLARED is the one real design decision, and it is open.** Two shapes:
  (a) keep `strategy: merge` and add a sibling `keys: { env: "deep", "permissions.allow": "union" }`
  map; (b) mint new strategy names. (a) keeps one strategy and makes the exceptions explicit, which
  matches how the file already documents `paths`; (b) multiplies the vocabulary. Recommend (a).
  Either way `ai/deploy.json`'s `version` **must** bump to 4 — the file's own rule is *"a field an
  old decoder does not know is invisible to it, the version is not"*, and an old `dotf` silently
  treating a per-key entry as plain merge is exactly the `~/.copilot` wipe that rule was written
  after.
- **The merge policy is an ALLOW-LIST, and that is a trap with a measured instance.** A key present
  in the template but absent from the policy is silently a no-op on every existing machine
  (`outputStyle`). The Go path must make the list explicit *and* fail loudly on a template key it
  does not recognise — otherwise it reproduces the defect with better syntax.
- **`// empty` versus `has()` in the jq original is load-bearing.** In jq a condition evaluating to
  `empty` makes the whole if-expression produce nothing, so a template omitting one optional key
  would deploy **nothing at all**. Go has no equivalent hazard, which means the *test* for it must
  be written deliberately rather than inherited from the shape of the code.
- **Characterization oracle: there is no single capturable script here.** Unlike CLI-021, the
  behaviour is spread across four regions that run at different points of a setup, with side effects
  on `$HOME` and on `claude` itself. Golden capture does not transfer; see `tasks.md`, which settles
  this rather than leaving it open.
- **Plugin install and MCP registration are not file installs.** They shell out to `claude` and can
  fail per item. `dotf deploy`'s plan/compare/install model assumes a file destination, so these two
  need a plan representation that is honest about "this is an action, not a file" — the part of this
  work least constrained by an existing pattern.

## Acceptance criteria

- [ ] Claude's four capabilities are reachable through `dotf deploy`, declared in `ai/deploy.json`.
- [ ] `ai/claude/plugins.json` is the sole plugin list read by the Go path, and a test fails if the
      twins' hardcoded lists drift from it.
- [ ] `env`, `enabledPlugins` (nested merge) and `permissions.allow` (deduped union) are preserved
      per the table above, each with a test that fails under top-level replace.
- [ ] The per-key policy is declared in the manifest, and an unrecognised template key **fails
      loudly** rather than being silently skipped.
- [ ] `ai/deploy.json` `version` is bumped, and an older decoder refuses the file rather than
      misreading it.
- [ ] The Go path emits no `hooks` key under any input — asserted, not merely omitted.
- [ ] Plugin counting reflects install success (Linux behaviour), with #1491's divergence recorded.
- [ ] **No twin deleted, no caller repointed** — nothing under `setup-*.{sh,ps1}` or `scripts/`.

## References

- Bitácora board: mlorentedev/dotfiles#1339, and the measurement comments dated 2026-09-04/05
- ADR-032:122-124 — the `dotf agent` / `dotf harness` noun boundary that rules out the original name
- ADR-020 — strangler-fig; Linux is the reference twin
- `ai/deploy.json` `$comment` — MCP registration named as slice 3 of CLI-039
- AI-039 (#1322) — `strategy: merge` and the `~/.copilot` wipe it was written after
- `specs/archive/SDD-002-settings-portability/proposal.md` — the per-key policy's origin
- HARNESS-045 AC1 — hooks left this function; `dotf harness bind` owns them
- SDD-021 / `cli/internal/mem/session_start_adapter.go:83` — the shared 10240-byte threshold
- #1491 — the Windows plugin counter, filed while measuring this ticket
- CLI-021 (#490) — the twin-port precedent: build beside, prove parity, cut over separately
- #1450 — why "port then delete" and not "delete as asked"
