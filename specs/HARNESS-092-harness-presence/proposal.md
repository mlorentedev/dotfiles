---
id: "HARNESS-092-harness-presence"
type: spec
status: implementing # draft | implementing | verifying | archived
created: "2026-08-28"
issue: "mlorentedev/dotfiles#1326"   # repo#NNN — GitHub issue / Project item that tracks this spec
tags: [spec, proposal, harness, windows, cli]
template_version: "1.0"
---

# HARNESS-092-harness-presence

> **Naming**: file lives at `<repo>/specs/HARNESS-092-harness-presence/proposal.md`. `HARNESS-092-harness-presence` is `AREA-NNN-slug` (e.g. `TOOL-001-secret-drift`).

## Why

Agent presence — the roster of invocable personas and the skills each MUST consume, injected
between `AGENT-PRESENCE` markers into every harness's always-loaded instructions file — is how a
persona's enforcement reaches a harness that has no provider hook. It lived only in
`compile-harness.sh` (`build_agent_presence` / `inject_agent_presence`), which `setup-linux.sh`
runs and `setup-windows.ps1` never ported: measured 2026-08-27 on the Windows work box, zero
regions in `~/.claude/CLAUDE.md`, `~/.pi/agent/AGENTS.md`, `~/.config/opencode/AGENTS.md` and
`~/.copilot/copilot-instructions.md`, and every doctor check green — `checkInstructionDrift`
strips the region before comparing, so nothing looked (#1326).

## What

- `dotf harness presence` renders the roster for every `agents.presence[]` target of the manifest
  and injects it into `$HOME/<file>`: replacing an existing region in place, appending a fresh
  one otherwise, leaving everything outside the markers byte-identical, honouring the file's line
  ending, and not rewriting a file whose region already holds this roster. The begin marker
  carries the block's sha (LF form, first 16 hex chars — `sha256sum | cut -c1-16`, as the shell
  did). `--render <agent>` prints the block and writes nothing. A target file that does not
  exist is skipped and said so; a harness no persona targets gets no region; a broken record
  fails the command.
- `compile-harness.sh --deploy` delegates: `deploy_agent_presence` calls the verb,
  `build_agent_presence` (still composing the compact doctrine payload for agy/codex) calls
  `--render`, `inject_agent_presence` (and the now-dead `agent_skills_line` /
  `record_has_block_skills`) are deleted. A dotf that is absent or predates the verb is a loud
  WARN naming what was NOT deployed and the deploy continues — the engine's bootstrap contract,
  the same degrade the tier and capability resolvers have; no awk fallback, because a roster
  rendered by a second parser is the drift the port removes. A dotf that runs the verb and fails
  does fail `--deploy`.
- `setup-windows.ps1` calls `dotf harness presence --repo-root $DotfilesDir` after
  `Deploy-SkillRecord`, once every base file is in place.
- `dotf doctor` gains "Agent presence (forced skills)": per target (gated on `requires_command`
  like the instruction-drift check), the region's sha is compared with the roster the records
  render today — PASS with counts, WARN "no presence region" or "stale presence" naming the file
  and `dotf harness presence`, SKIP without a repo. It renders, never writes.
- The four presence scenario tests in `tests/compile-harness.bats` move to Go
  (`cli/internal/harness/presence_test.go`); the bats layer keeps its contract: `--deploy` calls
  the verb with the checkout as `--repo-root`, honours its exit status, and refuses to deploy
  without it. The stub `dotf` simulates `--render` from the records as it already simulates
  `resolve-skills`.

## Out of scope

- Porting the rest of `compile-harness.sh --deploy` (records, doctrine, skill catalog) — that is
  CLI-026 (#495); this is its first writing slice.
- Array-union or per-harness ordering of the roster: name order, as the shell's glob gave.
- The shell's `targets:` substring match (`[pi]` matched `copilot`): Go matches whole names,
  which is the documented intent; recorded as a behaviour difference, not preserved.

## Risks / open questions

- CRLF instructions files on Windows (the checkout copies them CRLF): the injector keeps the
  file's ending and the sha is computed over LF, so Linux and Windows agree. Resolved by test.
- `dotf` availability inside `compile-harness.bats`: the suite stubs `dotf` on purpose (ADR-020,
  "do not build dotf here"); the stub gains the verb. Resolved.
- A first setup on a box whose base files are not yet deployed when the verb runs: targets are
  skipped and reported; the next setup run converges. Resolved: same as the shell's behaviour.

## Acceptance criteria

Observable outcomes. Each must be testable.

- [ ] AC1 — `BuildPresence` renders byte-identically to the shell (header once, name order,
  `[ids]` or `none`, autonomous personas skipped, `targets:` honoured by whole name); the sha is
  line-ending independent.
- [ ] AC2 — `DeployPresence` injects every manifest target once, keeps user content and the
  GENERATED region intact, appends when no markers exist, replaces a stale region in place, is
  idempotent (second run "unchanged", file untouched), keeps CRLF, skips an absent file, injects
  nothing for an empty roster.
- [ ] AC3 — `compile-harness.sh --deploy` calls `dotf harness presence --repo-root <checkout>`,
  fails when it fails, warns "presence NOT deployed" and continues when dotf predates the verb,
  and carries no injector; `setup-windows.ps1` calls the verb after `Deploy-SkillRecord`.
- [ ] AC4 — `dotf doctor` reports presence by status (PASS with counts / WARN missing / WARN stale
  / SKIP without repo), gates copilot's file on the binary, and writes nothing.
- [ ] AC5 — On the Windows work box: `dotf harness presence` injects the roster into the four
  files (measured 0 regions before), a second run reports every file current, `dotf doctor`'s
  section is green, `compile-harness.sh --check` still passes on Linux CI.

## References

- Bitácora board: #1326 (HARNESS-092); related #495 (CLI-026), #563 (HARNESS-047), #1319 (AC7 of HARNESS-045: resolve-skills delegation)
- Related ADR: `docs/adr/adr-020-tooling-cli-go-convergence.md` (C7, strangler fig), `docs/adr/adr-027-*` (marker namespaces)
