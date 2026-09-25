---
id: "SEC-001-secrets-run-guard"
type: spec
status: implementing
created: "2026-09-02"
issue: "mlorentedev/dotfiles#1458"
tags: [spec, proposal, secrets, security]
template_version: "1.0"
---

# SEC-001-secrets-run-guard: Guard secrets run against introspection commands

## Why

Commands like `dotf secrets run -- env` or `printenv` execute an introspection binary within the child environment, dumping all decrypted secrets directly to stdout and contaminating session transcripts. This violates the core ADR-028 doctrine ("Never dump a secrets store to standard output"). A deterministic guard is required to refuse introspection commands at invocation time.

## What

1. `dotf secrets run` rejects commands whose base binary is `env`, `printenv`, or `export`, returning a clear error and exiting non-zero without decrypting or launching.
2. `dotf secrets run` inspects shell wrapper arguments (`sh -c`, `bash -c`, `zsh -c`) and rejects shell snippets invoking dangerous introspection commands.
3. `ai/claude/settings.json` explicitly adds `Bash(env:*)`, `Bash(printenv:*)`, `Bash(export -p:*)` to `permissions.deny`.
4. `ai/pi/models.json` registers the `openrouter` provider with `${OPENROUTER_API_KEY}` for automated deployment.

## Out of scope

- Arbitrary binary payload content inspection (we block by binary and shell arguments, not inspecting arbitrary external scripts).
- **A caller who means to print the environment (owner decision, 2026-09-24, after review round 3).** The guard is a tripwire for the accidental shapes of an environment dump, the ones an agent types without thinking. It is not a boundary against deliberate printing: any interpreter AC4 lets run can print the environment (`python3 -c`, `awk`'s `ENVIRON`, `cat /proc/self/environ`), and so can `eval` or a script file. What keeps the values out of the output is the redactor (AC7), which scrubs every injected value of 6 bytes or more whatever printed it.
- Indirection a lexical guard cannot see (review round 2, F6): a command run by another command (`nice env`, `timeout 5 env`, `xargs env`), a snippet read from stdin (`sh -s`) or from a script file, and a command name assembled at run time (`en$'v'`, `e=en; ${e}v`). The redactor still scrubs every injected value of 6 bytes or more from what such a command prints.
- Shells outside the POSIX family (`pwsh`, `cmd`, `fish`): #1650.
- **Deploying the Claude deny list to an existing installation.** `merge_claude_settings` merges `permissions.allow` and not `permissions.deny`, so AC6 and AC10 hold for the template only. The Go port of the Claude settings deploy owns the deploy (#1339). Decided by the owner on 2026-09-24, after review round 2 (F2).

## Risks / open questions

- *Risk:* A valid tool named `env` or wrapped with `env` could be blocked.
- *Mitigation:* Under `dotf secrets run`, `dotf` itself is the environment injector; calling `env` as the child command is an anti-pattern.

## Acceptance criteria

- [x] AC1: `dotf secrets run -- env` exits non-zero and refuses to run with a clear ADR-028 error message.
- [x] AC2: `dotf secrets run -- printenv` and `dotf secrets run -- /usr/bin/env` are similarly refused.
- [x] AC3: Shell wrappers like `sh -c "env | grep..."` are detected and refused. Once any argument sets the c flag (`-c`, `+c`, or a c inside a cluster), every argument after it is inspected, because which one a shell runs depends on its own option grammar (fail closed).
- [x] AC4: Legitimate tools (e.g. `python`, `goreleaser`, `dotf review`) run unhindered.
- [x] AC5: Comprehensive table-driven unit tests in `cli/internal/cmd/secrets_test.go` verify safe and unsafe commands.
- [x] AC6: The Claude settings template (`ai/claude/settings.json`) denies `Bash(env:*)`, `Bash(printenv:*)` and `Bash(export -p:*)`, and the Pi models catalog registers `openrouter`. Template-scoped: the deploy is #1339 (see Out of scope).
- [x] AC7: Real-time byte-level stream redactor (`redactWriter`) scrubs any injected secret value (len >= 6) from stdout/stderr, replacing with `[REDACTED:<KEY>]`.
- [x] AC8: Post-mortem and multi-agent testing isolation lesson recorded in `docs/lessons/lesson-261-...`.
- [x] AC9: `dotf secrets show` implements solutions 1-2-3 (`--reveal` explicit flag, `-c`/`--clip` clipboard copy, interactive TTY masking, and agent session refusal). The refusal recognises a marker exported by every harness `harness/model-map.json` declares.
- [x] AC10: The Claude settings template (`ai/claude/settings.json`) deny list is hardened with `sudo`, `git clean`, `dotf secrets show`, and pipe-to-shell bans. Template-scoped: the deploy is #1339 (see Out of scope).

## References

- Issue: mlorentedev/dotfiles#1458
- Retroactive review and its follow-ups: #1626 (sweep), #1646 (agent-session markers), #1650 (non-POSIX shells), #1339 (deny-list deploy)
- ADR-028: On-demand secrets delivery and process injection
- Global doctrine: Non-negotiable rules on secret store dumps
