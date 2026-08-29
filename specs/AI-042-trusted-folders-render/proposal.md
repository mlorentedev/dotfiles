---
id: "AI-042-trusted-folders-render"
type: spec
status: implementing # draft | implementing | verifying | archived
created: "2026-08-29"
issue: "mlorentedev/dotfiles#1334"   # repo#NNN — GitHub issue / Project item that tracks this spec
tags: [spec, proposal, copilot, agy, deploy, windows]
template_version: "1.0"
---

# AI-042-trusted-folders-render

> **Naming**: file lives at `<repo>/specs/AI-042-trusted-folders-render/proposal.md`. `AI-042-trusted-folders-render` is `AREA-NNN-slug` (e.g. `TOOL-001-secret-drift`).

## Why

`ai/copilot/config.json` carries six `trustedFolders` literals and `ai/agy/settings.json` three
`trustedWorkspaces`, spelling two usernames and two home roots (`/home/manu`, `C:\Users\mlorente`,
`C:/Users/mlorente`). Both are deployed to `$HOME` by both setups, so on any other machine — a new
box, a different user, a renamed home — the trust lists silently match nothing and every session
starts with a trust prompt. `tests/antigravity.bats` guards this class for one file
(`mcp_servers.json`) only (#1334, scripts-parity audit F2).

## What

- `ai/deploy.json` entries gain `paths`: `native` or `slash`. When set, `dotf deploy` expands the
  `{HOME}` / `{VAR}` tokens the manifest already uses for `dst` **inside the source's JSON string
  values**, rendering each token's expansion in the declared separator form wherever it sits
  (`filepath.FromSlash` for `native`, `filepath.ToSlash` for `slash`) and, for a string that
  **begins** with a token — a path — the whole string as well; a token elsewhere (inside a URL)
  leaves the rest of the string untouched (review rounds 2–3), before the strategy
  (merge or replace) runs. Expansion is JSON-aware so a native Windows path is JSON-escaped by the
  encoder, never by hand. `ManifestVersion` becomes 3 (a field that changes what an entry's content
  means; the frozen-schema test forces the bump).
- Which form, per tool, is the form that tool has been **observed** to accept — not a guess:
  - Copilot → `native`. Its trust check is `repoPathsEqual` in a native module (unreadable from the
    npm package), and the entries the CLI wrote itself when the user trusted folders on Windows are
    backslash paths. `copilot -p` does not evaluate trust at all (measured: identical behaviour with
    a forward-slash list and with an empty list), so the interactive form is the only evidence.
  - agy → `slash`. `ai/agy/settings.json` has carried `C:/Users/…/*` since it was written and the
    user works in agy on the Windows box daily.
- `ai/copilot/config.json` → four templated entries (`{HOME}/Projects`, `{HOME}/Projects/*`,
  `{HOME}/Projects/Workspace`, `{HOME}/Projects/Workspace/*`), one list for both OSes.
  `ai/agy/settings.json` → two templated `trustedWorkspaces`, and the file becomes a manifest entry
  (`agy-settings`, replace, `paths: slash`, `requires: agy`) instead of a `deploy_file` /
  `Copy-Item` in each setup — the third twin ADR-020 says to collapse on touch.
- `tests/copilot-config.bats` (renamed scope: Copilot + agy trust lists) refutes any `/home/<user>`,
  `C:\Users\<user>` or `C:/Users/<user>` literal across `ai/**/*.json`, and asserts the templates.
- `dotf doctor`'s manifest check needs nothing new: `PlanConfig` expands before comparing.

## Out of scope

- Reading Copilot's native trust store (`folderTrustIsTrusted`) or writing to it: `trustedFolders`
  in `config.json` is the CLI's secondary, file-based list and the only one a deploy can carry.
- A per-machine list of extra trusted roots (`machine.json`): the two roots above are the
  contract's; a box-local addition is the tool's own prompt, as today.
- Rendering through `dotf secrets render` (`{env:VAR}` from the secrets registry): the registry
  has no `HOME`; the manifest's own token resolver does, and it is the same one `dst` uses.

## Risks / open questions

- Copilot's `repoPathsEqual` might also accept forward slashes; `native` is the form with evidence,
  and switching later is one manifest word. Resolved: declare, do not guess.
- A merge entry with `paths` expands the *source* only; the destination's unmanaged keys are never
  touched. Resolved by test.
- Stacked on #1369 (`ManifestVersion` 2 + `DisallowUnknownFields`); rebased `--onto main` after its
  squash. Resolved: known pattern.

## Acceptance criteria

Observable outcomes. Each must be testable.

- [ ] AC1 — `paths: native` expands `{HOME}` inside JSON string values and renders each expansion
  with the OS separator — and the whole string when it begins with a token; `paths: slash` renders
  them with `/`; a token inside a longer string (a URL) leaves the rest of it as written; strings
  without a token are untouched;
  the output is valid JSON (a Windows path is escaped by the encoder); an unknown `paths` value and
  a non-JSON source with `paths` are rejected naming the entry.
- [ ] AC2 — `paths` composes with `merge` (expansion before the merge, unmanaged destination keys
  preserved) and with `replace`; `PlanConfig` reports "in sync" against an already-rendered
  destination.
- [ ] AC3 — `ai/copilot/config.json` and `ai/agy/settings.json` carry no user or home literal; the
  manifest declares `copilot-config` with `paths: native` and `agy-settings` (`slash`, `requires:
  agy`); neither setup copies `ai/agy/settings.json` any more; `ManifestVersion` is 3 with the
  frozen field set extended.
- [ ] AC4 — On the Windows work box: `dotf deploy` renders `C:\Users\mlorente\Projects\*` into
  `~/.copilot/config.json` (the CLI's own form, `firstLaunchAt` preserved) and
  `C:/Users/mlorente/Projects/*` into agy's `settings.json`; a second run is `in sync`; `dotf doctor`
  green; `copilot -p` and `agy` still answer.

## References

- Bitácora board: #1334 (AI-042); related #1322 (AI-039), #1369 (manifest v2), #495 (CLI-026)
- Related ADR: `docs/adr/adr-020-tooling-cli-go-convergence.md` (C7), `docs/adr/adr-025-cross-machine-paths.md`
