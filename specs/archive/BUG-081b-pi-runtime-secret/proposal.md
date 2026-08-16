---
id: "BUG-081b-pi-runtime-secret"
type: spec
status: archived # draft | implementing | verifying | archived
created: "2026-08-16"
issue: "mlorentedev/dotfiles#987"
tags: [spec, proposal, secrets, pi, security]
template_version: "1.0"
---

# BUG-081b-pi-runtime-secret

> Filed as `BUG-081b` rather than `BUG-081`: that id is already carried by two
> open issues (#987 and #989), a collision #893 tracks. The sub-id letter exists
> for exactly this, and propagating the duplicate would deepen it.

## Why

`ai/pi/models.json` declares `"apiKey": "{env:NAN_API_KEY}"` — **dotf's**
placeholder syntax, which pi's resolver does not understand. Proven against pi's
shipped `dist/core/resolve-config-value.js` rather than inferred:

```
"${NAN_API_KEY}"    -> resolves to the env value      (fp 3387dac8dd02, sentinel)
"$NAN_API_KEY"      -> resolves to the env value      (fp 3387dac8dd02)
"{env:NAN_API_KEY}" -> the LITERAL 17-char string     (fp f3407edcacbd)
```

That fingerprint is the same one #987 captured off the wire with a capture
server: pi sends the placeholder itself as the bearer token, so the server can
only answer 401 — indistinguishable from a bad credential. A previous session
spent most of its length looking at Bitwarden, the registry and the injection
path, all of which were healthy.

Nothing warns, at any layer. `setup-linux.sh` says the placeholders are
*"resolved at runtime"*, which is false. pi's own preflight
`isConfigValueConfigured("{env:NAN_API_KEY}")` returns **true**, because the
string contains no `$` and therefore has no env vars that could be missing.

Meanwhile the working path is worse than the broken one: on a **successful**
`dotf secrets render`, the credential is materialised into
`~/.pi/agent/models.json` and sits on disk in plaintext. That is what a
concurrent session read while debugging on 2026-08-15, landing a live
`NAN_API_KEY` in a transcript. It also contradicts ADR-028's stated posture —
secrets injected into one child process, never persisted.

## What

`ai/pi/models.json` declares `"apiKey": "${NAN_API_KEY}"` — pi's own syntax —
and pi resolves it at runtime from the environment that is already injected.

After this change:

- **No pi config on any machine contains a credential.** `dotf secrets render`
  substitutes only `{env:VAR}`, so `${NAN_API_KEY}` passes through untouched and
  the deployed file is the source file.
- **`models.json` stops being a second copy of the secret**, so rotation touches
  Bitwarden and nothing else.
- **An unset variable fails with a named error** —
  `Failed to resolve API key from environment variable: NAN_API_KEY` — instead of
  a 401 that looks like a bad credential.
- **`dotf doctor` fails** on a deployed pi config that still carries either
  defect: an unresolvable `{env:` placeholder, or a materialised secret.

## Injection already exists on every platform

The change depends on `NAN_API_KEY` being present in pi's environment. All three
paths predate this spec:

| path | mechanism |
|---|---|
| review launcher | `dotf secrets run -- pi …` (`review_launch.go:80`, Go) |
| interactive Linux/macOS | `pi()` in `.zshrc:70`, `.bashrc:92` |
| interactive Windows | `function pi` in `powershell/profile.ps1:254` |

## Out of scope

- **The setup scripts.** They need no edit: `render` is a passthrough once the
  source uses `${}`, so the existing deploy blocks install the file verbatim.
  Their false *"resolved at runtime"* warning is left in place deliberately — it
  dies with the block when config deployment ports to `dotf` (#1023), rather than
  being patched twice in two languages now.
- **opencode.** Its config carries the same `{env:}` syntax and one materialised
  literal, but whether opencode self-resolves has never been tested; pi's
  identical assumption was disproven only by experiment. Gated on that evidence,
  not bundled here.
- **A macOS bootstrap.** `dotf` ships darwin binaries and this change is
  platform-neutral, but there is no `setup-macos.sh` (README: planned). Out of
  scope, and not silently claimed.

## Risks / open questions

- **A script invoking `pi` directly**, outside an interactive shell and outside
  the launcher, gets no injection: shell functions do not reach non-interactive
  shells. It now fails with pi's named error rather than a 401, and the doctor
  guard covers the config-side half. Accepted, and stated.
- **The guard goes red on this machine the moment it merges**, because the
  deployed copy still holds a materialised literal until a redeploy. Intended:
  it is the mechanical reminder that the maintenance window is owed. Called out
  so it is not mistaken for a regression.
- **`!command` values** are also supported by pi (executed, stdout used), which
  would make the config self-sufficient with no wrapper at all. Not taken: it
  depends on `dotf secrets show`, whose contract is itself contested (#952).
  Recorded in the ADR as the considered alternative.

## Acceptance criteria

- [ ] `ai/pi/models.json` uses pi's `${VAR}` syntax; no `{env:` remains in it.
- [ ] `dotf secrets render` leaves the file unchanged (a passthrough), asserted
      by a test, so the deployed copy equals the source.
- [ ] `dotf doctor` FAILs when a deployed pi config contains `{env:` — pi
      provably cannot resolve it.
- [ ] `dotf doctor` FAILs when a deployed pi config's `apiKey` is neither
      `${VAR}` nor `$VAR` — i.e. a materialised secret on disk.
- [ ] Both doctor branches are observed failing against real state, not only a
      fixture.
- [ ] No `setup-*.sh` / `setup-*.ps1` line changes.
- [ ] An ADR records the posture change: agent-config secrets resolve at runtime;
      deploy-time materialisation retires per config as configs convert.

## References

- Bitácora board: `mlorentedev/dotfiles#987`
- `docs/adr/adr-028-secrets-two-tier-bitwarden-age.md` — the posture this restores
- #1012 / `dotf secrets probe` — the instrument that measured the two stores
- #1023 (CLI-039) — the config-deployment port that retires the setup blocks
- #952 — why `!dotf secrets show` was not chosen
- #893 — the `BUG-081` id collision this spec's sub-id avoids
