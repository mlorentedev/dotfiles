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

## Risks / open questions

- *Risk:* A valid tool named `env` or wrapped with `env` could be blocked.
- *Mitigation:* Under `dotf secrets run`, `dotf` itself is the environment injector; calling `env` as the child command is an anti-pattern.

## Acceptance criteria

- [x] AC1: `dotf secrets run -- env` exits non-zero and refuses to run with a clear ADR-028 error message.
- [x] AC2: `dotf secrets run -- printenv` and `dotf secrets run -- /usr/bin/env` are similarly refused.
- [x] AC3: Shell wrappers like `sh -c "env | grep..."` are detected and refused.
- [x] AC4: Legitimate tools (e.g. `python`, `goreleaser`, `dotf review`) run unhindered.
- [x] AC5: Comprehensive table-driven unit tests in `cli/internal/cmd/secrets_test.go` verify safe and unsafe commands.
- [x] AC6: Claude permissions deny list and Pi models catalog are updated.

## References

- Issue: mlorentedev/dotfiles#1458
- ADR-028: On-demand secrets delivery and process injection
- Global doctrine: Non-negotiable rules on secret store dumps
