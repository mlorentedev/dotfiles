# ADR-034: Agent-config secrets resolve at runtime, not at deploy time

- **Status:** accepted
- **Date:** 2026-08-16
- **Supersedes (partially):** the deploy-time substitution posture of SDD-009, per config as each converts
- **Related:** [ADR-028](adr-028-secrets-two-tier-bitwarden-age.md), [ADR-020](adr-020-tooling-cli-go-convergence.md), [ADR-030](adr-030-secrets-registry-source-model.md)
- **Issue:** #987

## Context

Agent configs (pi's `models.json`, opencode's `opencode.jsonc`) need a provider
API key. The mechanism to date is **deploy-time substitution**: the repo source
carries `{env:VAR}` placeholders, and `dotf secrets render` rewrites them with
real values as setup installs the file.

Two things went wrong with that, and they point the same way.

**The failure path lies.** pi's resolver interpolates `$VAR` and `${VAR}` only;
`{env:VAR}` contains no `$`, so pi sends the placeholder itself as the bearer
token. The setup script logs that placeholders are *"resolved at runtime"*, which
nothing does. pi's own `isConfigValueConfigured("{env:NAN_API_KEY}")` returns
`true`, because a string with no `$` has no variables that could be missing. And
the server can only answer 401 — indistinguishable from a bad credential. Three
layers each reported a health none had established, and a session spent most of
its length inspecting Bitwarden, the registry and the injection path, all
healthy.

**The success path is worse.** When substitution *works*, the credential is
written into the deployed config and lives on disk in plaintext. On 2026-08-15
that is exactly how a live `NAN_API_KEY` reached a session transcript: someone
opened `models.json` to debug the failure path above. It also contradicts
ADR-028, whose posture is that secrets are injected into one child process and
never persisted — a rule the deploy step was quietly exempt from.

The injection those configs would need already exists on every platform: the
review launcher shells through `dotf secrets run`, and both shell profiles and
the PowerShell profile define wrapper functions that scope `--only` to the
agent's keys.

## Decision

**An agent config references its secret in the syntax its own runtime resolves,
and the value is supplied by the environment at invocation.**

For pi that is `"apiKey": "${NAN_API_KEY}"`. `dotf secrets render` substitutes
only `{env:VAR}`, so such a config passes through the deploy step byte-identical
— no setup change is needed to adopt this, which is also why adoption is per
config rather than a flag day.

Deploy-time materialisation is not removed from `render`; it retires by having
no callers, as each config converts.

A `dotf doctor` check enforces both halves on the deployed copy: a remaining
`{env:` is a broken deploy, and an `apiKey` that is neither `${VAR}`, `$VAR` nor
`!command` is a materialised secret. It names the provider and never the value.

## Consequences

**Good.** No credential in a pi config on any machine. `models.json` stops being
a second copy, so rotation touches Bitwarden alone. An unset variable now fails
with `Failed to resolve API key from environment variable: NAN_API_KEY` instead
of a 401 — converting the most expensive diagnostic failure this repo has
recorded into one that names its cause. And the fix needs no line in either
setup script, which is the direction ADR-020 wants those scripts moving.

**Bad.** A `pi` invoked from a non-interactive script, outside the launcher, gets
no injection: shell functions do not reach non-interactive shells. That now fails
with a named error rather than a 401, and the doctor guard covers the config-side
half, but it is a real narrowing of where a bare `pi` works.

**Ugly.** Until each machine redeploys, its config still holds a materialised
literal — so the guard goes red on exactly the machines that have not yet
converted. That is intended as a mechanical reminder rather than a regression,
but it means merging this puts a red section on a working box.

## Alternatives considered

**Make the fallback honest and keep materialising.** Fix the "resolved at
runtime" warning, add the guard, leave the secret on disk. Cheaper, and it
addresses the misleading message — but it leaves the credential at rest, which is
the half that actually caused a leak. Rejected: it fixes the symptom that was
noticed and not the one that did damage.

**Use pi's `!command` form** — `"apiKey": "!dotf secrets show NAN_API_KEY"`,
executed by pi and its stdout used. Genuinely attractive: it needs no wrapper at
all, so it closes the non-interactive gap this decision accepts. Rejected for now
because it depends on `dotf secrets show`, whose contract is itself contested
(#952: the no-rendering rule and the advertised `show` command contradict each
other). Worth revisiting once that settles.

**Render at deploy time into a 0600 file outside the config.** Keeps one
mechanism for every agent regardless of what its runtime supports, at the cost of
a second artifact to keep in sync and a credential still at rest. Rejected: it
preserves the property this decision exists to remove.

## Scope

pi only, today. opencode carries the same `{env:}` syntax and one materialised
literal, but whether it self-resolves has never been tested — and pi's identical
assumption was disproven only by experiment, which is precisely the reason not to
assume it twice. Converting opencode is gated on that evidence.
