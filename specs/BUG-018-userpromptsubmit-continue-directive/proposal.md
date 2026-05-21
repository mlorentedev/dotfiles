---
id: "BUG-018-userpromptsubmit-continue-directive"
type: spec
status: implementing
created: "2026-05-21"
tags: [spec, proposal, claude-mem, heal, cross-os-parity, hook-protocol]
template_version: "1.0"
---

# BUG-018-userpromptsubmit-continue-directive

## Why

After BUG-017 (PR #84) closed the EPIPE race for claude-mem hooks, the user immediately hit a SECOND blocker. Error message changed (no more `printf: write error`), proving BUG-017 landed, but `UserPromptSubmit operation blocked by hook ... No stderr output`. Minutes later, after a narrow BUG-018 manual patch (only session-init), the Stop hook also failed 9 consecutive times forcing Claude Code's `CLAUDE_CODE_STOP_HOOK_BLOCK_CAP` to override.

Root cause: claude-mem ships 6 hooks. 5 terminate with `node ... hook claude-code <event>` and lack Claude Code's `{"continue":true,"suppressOutput":true}` directive. bun-runner.js writes diagnostic stdout when stdin is empty (upstream claude-mem#2188); Claude Code reads non-JSON-directive stdout as "do not continue" -> blocks. SessionStart-start is the only hook already emitting the directive.

## What

Extend `heal_hooks_json` / `Repair-HooksJson` with a regex-based substitution that finds any `hook claude-code <event>"` terminator and appends ` 2>/dev/null; echo '{"continue":true,"suppressOutput":true}'` before the closing JSON quote. Same heal pass as BUG-017 (one sed/Replace per pattern). All 5 affected hooks get the directive in one pass.

Idempotent: subsequent runs see no broken terminator left -> silent skip.

## Out of scope

- **Setup hook (`node version-check.js`).** Different terminator pattern; fires only on plugin install/update, not user hot path. Future BUG-018b if it surfaces.
- **AST-level JSON manipulation.** Pure string substitution is sufficient.

## Risks / open questions

- **Risk: regex over-matches in future hooks.json variants.** Mitigation: regex constrained to `[a-z][a-z-]*` for event name -- excludes Setup's `version-check.js` and capitalized identifiers.
- **Risk: `2>/dev/null` silences legitimate stderr.** Mitigation: bun-runner writes to stdout (Claude Code's complaint is "No stderr output"); silencing stderr is harmless.

## Acceptance criteria

- [ ] Both heal scripts contain regex substitution covering all 5 hook terminators.
- [ ] Both heal scripts log `BUG-018` and reference `continue` in the replacement.
- [ ] `tests/setup-linux.bats`: 1 new parity assert.
- [ ] `bash -n` + PowerShell AST + PSScriptAnalyzer clean; ASCII-only.
- [ ] Empirical: user's hooks.json patched on all 5 hooks; prompts complete without loop.

## References

- Vault: `10_projects/dotfiles/11-tasks.md` BUG-018 entry.
- Predecessor: BUG-017 (PR #84) -- EPIPE race fix.
- Upstream: [thedotmack/claude-mem#2607](https://github.com/thedotmack/claude-mem/issues/2607) (cascade race) + claude-mem#2188 (hook protocol mismatch).
