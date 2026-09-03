# 261 - Never test secret guards against live credentials, and redact at the stream boundary

**Date:** 2026-09-02
**Area:** Security, CLI, secrets, multi-agent testing

## What happened

During the implementation of an introspection guard in `dotf secrets run` (SEC-001, #1458),
an adversarial testing subagent was launched to probe bypass vectors against the newly written
`assertSafeChildCommand`. The subagent was asked to test shell wrappers against the live binary.

Because the host environment had an unlocked Bitwarden session and valid local keyrings,
the child process evaluated the production registry. When the subagent probed an evasion
vector not yet covered by the initial boundary regex (`sh -c "'env'"`), the guard let the command
through. The resulting output dumped the complete environment—including active API tokens—directly
into the subagent's stdout stream, which was written to disk in its local transcript.

Two root failures combined to cause this breach:

1. **Testing leak-prevention logic against production credential stores:** Testing an anti-leak
   guard using live keys is an antipattern. When the guard fails its test, it fails by leaking
   the very keys it was meant to guard. All adversarial security testing must use synthetic,
   in-memory mock registries (`FOO=mock-dummy-token`), never the operator's real vault.
2. **Relying solely on an input-command blacklist:** Filtering only `argv` at launch time
   leaves open the entire grammar of shell escapes, aliases, script wrappers, and unforeseen
   child binaries. Input validation is the first line of defense, but it cannot be the only one.

## The resolution

Two mechanical changes were introduced to close this failure mode permanently:

1. **Defense-in-depth output redaction in Go (`redactWriter`):** `dotf secrets run` knows the
   exact set of secret values it resolves and injects into `childEnv`. In addition to refusing
   introspection binaries at launch, `dotf` wraps the child's `stdout` and `stderr` streams with a
   real-time byte-level redactor. Any byte sequence matching an injected secret value (len >= 6)
   is replaced with `[REDACTED:<KEY>]` before reaching the terminal, pipe, or transcript. Even
   if an unrecognised binary or shell construct dumps memory or environment variables, the plaintext
   secret never leaves the `dotf` process boundary.
2. **Strict subagent isolation doctrine:** Subagents and test harnesses must never be directed to
   invoke live credential decryption commands. All evasion test cases are formalized into Go unit
   tests (`TestAssertSafeChildCommand`, `TestRedactWriter`) operating on synthetic fixtures.

## Doctrine update

- **Never test secret guards against live credentials.** Mock the registry.
- **Redact at the stream boundary.** If the parent process knows the secret it gave the child,
  it must scrub that secret from whatever the child gives back.
