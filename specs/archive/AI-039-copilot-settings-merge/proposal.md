---
id: "AI-039-copilot-settings-merge"
type: spec
status: archived # draft | implementing | verifying | archived
created: "2026-08-28"
issue: "mlorentedev/dotfiles#1322"   # repo#NNN — GitHub issue / Project item that tracks this spec
tags: [spec, proposal, copilot, deploy, windows]
template_version: "1.0"
---

# AI-039-copilot-settings-merge

> **Naming**: file lives at `<repo>/specs/AI-039-copilot-settings-merge/proposal.md`. `AI-039-copilot-settings-merge` is `AREA-NNN-slug` (e.g. `TOOL-001-secret-drift`).

## Why

Both setups copy `ai/copilot/*` verbatim into `~/.copilot/`, and the file that carries our
preferences — `config.json` with `model`, `includeCoAuthoredBy`, `telemetry` — is one the CLI
rewrites on every launch: measured on the Windows work box (Copilot CLI 1.0.81), the deployed
`config.json` comes back with the header *"User settings belong in settings.json. This file is
managed automatically."* holding only `trustedFolders`, `firstLaunchAt` and `telemetry`, while the
keys the CLI actually reads live in the user's `settings.json` beside the per-box preferences
(`allowedUrls`, `effortLevel`, `contextTier`, `renderMarkdown`). So the repo targets a file the CLI
overwrites, a verbatim copy of a `settings.json` would wipe the per-box preferences, `telemetry` is
a key `copilot help config` does not document, and `dotf doctor` compares only
`copilot-instructions.md` — it cannot tell any of this.

## What

- `ai/deploy.json` entries gain a `strategy`: `replace` (today's behaviour, the default) or
  `merge` — the source's top-level keys are written into the destination JSON object, every other
  key the destination holds is preserved, and "in sync" means every managed key already carries
  the source value (semantic, so the CLI's `//` header on `config.json` never churns a deploy).
- Copilot's JSON is deployed by `dotf deploy` on both OSes: `copilot-settings`
  (`ai/copilot/settings.json` → `~/.copilot/settings.json`, merge — `model`,
  `includeCoAuthoredBy`, `autoUpdate`), `copilot-config` (`ai/copilot/config.json` →
  `~/.copilot/config.json`, merge — `trustedFolders` only), `copilot-mcp`
  (`ai/copilot/mcp-config.json`, replace). The setup scripts copy only
  `copilot-instructions.md`; the `cp -rf ai/copilot/*` / `Copy-Item ai\copilot\*` glob is gone.
- `telemetry` leaves the repo file (undocumented key). A box that already has it keeps it — merge
  preserves unmanaged keys — and a fresh box never gets it, which is what #1322 asks.
- `dotf doctor` gains "Deployed agent configs (ai/deploy.json)": every non-rendered manifest
  entry is compared side-effect-free; a drifted entry WARNs by name with `dotf deploy <name>` as
  the remedy. Comparing must not create the destination directory: today `Deploy(dryRun)` stages
  before it compares, so the compare is hoisted for non-rendered entries and `--dry-run` stops
  touching disk.
- `tests/copilot-config.bats` asserts every top-level key in both repo files against the set
  `copilot help config` documents (frozen list, CLI version and date recorded in the test),
  `includeCoAuthoredBy=false`, `autoUpdate=false` (ADR-036: the catalog owns the version), no
  `powershellFlags` (the #1324 constraint), no `telemetry`, and that the manifest declares the
  three entries with their strategies.

## Out of scope

- Rendering `trustedFolders` per machine (#1334, AI-042): the literal list is deployed as today,
  now by merge instead of by overwrite — not a regression, not the fix.
- Array union on merge (keeping box-local `trustedFolders` entries): a managed key is owned by the
  repo, whole value, the same contract Claude's settings merge has.
- Moving Claude's settings merge onto this strategy (#1339, CLI-063) — it becomes possible, it is
  not done here.
- Doctor comparing rendered entries (`pi`): rendering needs the secrets daemon; doctor stays
  read-only and says how many entries it did not compare.

## Risks / open questions

- The CLI may rewrite `settings.json` (`/model` writes `model`); doctor then WARNs and
  `dotf deploy copilot-settings` restores the repo value — that is the managed-key contract,
  stated in the manifest comment. Resolved: intended.
- `{COPILOT_HOME}` as the destination token would be the ADR-025 spelling, but `env.ResolvePath`
  needs `machine.json`, which does not exist yet when `dotf deploy` runs on a first setup (deploy
  precedes `dotf env set/generate` in both scripts). Resolved: `{HOME}/.copilot`, like the
  existing entries, with the reason in the manifest comment.
- Windows CRLF on `settings.json`: merge writes plain LF JSON; the CLI accepts it (measured:
  the CLI's own writes are LF). Resolved.

## Acceptance criteria

Observable outcomes. Each must be testable.

- [ ] AC1 — A `merge` entry writes the source's top-level keys into the destination, preserves
  every other destination key, creates the file when absent, tolerates the CLI's `//` header on
  read, and a second run reports no change and does not rewrite the file (mtime unchanged).
- [ ] AC2 — An in-sync or dry-run deploy of a non-rendered entry creates neither the destination
  directory nor a staging file; a manifest with an unknown `strategy` is rejected naming the entry.
- [ ] AC3 — `ai/deploy.json` declares `copilot-settings` (merge), `copilot-config` (merge) and
  `copilot-mcp` (replace); `ai/copilot/settings.json` carries exactly `model`,
  `includeCoAuthoredBy=false`, `autoUpdate=false`; `ai/copilot/config.json` carries exactly
  `trustedFolders`; every top-level key of both is in the documented set; neither setup script
  copies `ai/copilot` by glob any more, both copy `copilot-instructions.md` explicitly.
- [ ] AC4 — `dotf doctor` reports "Deployed agent configs (ai/deploy.json)": PASS when every
  non-rendered entry is in sync, WARN naming each drifted entry and `dotf deploy <name>`, SKIP
  without a repo; it never creates a destination directory.
- [ ] AC5 — On the Windows work box: `dotf deploy` merges the three managed keys into the
  existing `settings.json` while `allowedUrls`, `effortLevel`, `contextTier`, `renderMarkdown`
  survive; a second `dotf deploy` reports `in sync`; `dotf doctor` shows the new section green;
  `copilot -p` still answers.

## References

- Bitácora board: #1322 (AI-039); related #1334 (AI-042), #1339 (CLI-063), #1324 (CLI-058), #1321 (AI-038)
- Related ADR: `docs/adr/adr-020-tooling-cli-go-convergence.md` (C7, strangler fig), `docs/adr/adr-036-install-channels.md` (catalog owns the version → `autoUpdate=false`), `docs/adr/adr-025-cross-machine-paths.md`
- Prior spec: `specs/archive/CLI-039-*` (dotf deploy), `specs/archive/CLI-054-*` (bare deploy installs every entry)

<!-- archived 2026-08-28 — PR: https://github.com/mlorentedev/dotfiles/pull/1365 -->
