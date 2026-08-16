---
id: "CLI-038-secrets-probe"
type: spec
status: archived # draft | implementing | verifying | archived
created: "2026-08-16"
issue: "mlorentedev/dotfiles#1012"   # repo#NNN — GitHub issue / Project item that tracks this spec
tags: [spec, proposal, secrets, diagnostics, security]
template_version: "1.0"
---

# CLI-038-secrets-probe

## Why

Diagnosing a secrets-backend failure requires seeing what the backend replied, and today the only tool for that is raw `curl` against `bw serve`, whose default behaviour is to print the credential. That is not a hypothetical hazard: a live credential reached a session transcript **three times in one day** (2026-08-15) — once through an ineffective `sed` redaction filter, once through a plain `curl` of `/object/item/<id>` while diagnosing #988, and once through `curl -o /dev/null` with multiple URLs, where `-o` binds to the first URL only and the remaining seven bodies print in full.

The rule against this was already written in `docs/lessons.md`, was correct, and had been read by the person who broke it — twice more, within hours, while writing up the previous instance. So the gap is not knowledge. **The safe path is inconvenient and the unsafe one is the default**, and under that gradient prose loses every time. This repo's standing rule is that a bug class emits a mechanical guard in the same change, not a stronger warning.

## What

A new subcommand, `dotf secrets probe`, that answers the questions a backend diagnosis actually asks — is the endpoint reachable, what status did it return, is the envelope well-formed, which fields exist, did a value change — while being **structurally unable to print a secret value**.

```
dotf secrets probe <secret-id>        # status, content-type, size, envelope validity,
                                      # field NAMES, value LENGTHS, sha256[:12] fingerprints
dotf secrets probe <secret-id> --raw  # additionally echoes non-2xx bodies, capped
dotf secrets probe <secret-id> --count N  # repeat N times, report the status distribution
```

After this change, the sanctioned answer to "what is the daemon actually returning?" is a command that cannot leak, instead of a `curl` invocation that leaks by default.

`--count` exists because the failure this tool is built to investigate (#988) is intermittent and load-dependent: a single probe cannot characterise it, and the alternative is a hand-written shell loop — which is exactly the artisanal `curl` this ticket removes.

## What it prints, precisely

| Fact | Emitted | Why it is safe |
|---|---|---|
| HTTP status, content-type, byte count | yes | transport metadata, no payload |
| envelope validity (`json.Valid`) | yes | a boolean derived from the body |
| `success` / `message` from the envelope | yes | daemon-authored control fields |
| field **names** | yes | schema, not content |
| value **lengths** | yes | integer derived from content |
| value `sha256[:12]` | yes | non-reversible; enough to prove two values differ |
| value bytes | **never** | this is the credential |
| 2xx body | **never**, including under `--raw` | a 200 from `/object/item/<id>` *is* the credential |
| non-2xx body | only under `--raw`, capped | the actual diagnostic artifact; cannot be a credential |

## Out of scope

- **Fixing #988.** The daemon's intermittent 500 is owned by the `cli/internal/secrets/` lane. This ticket builds the instrument, not the fix.
- **Changing `call()`'s safety contract.** #1007 made `call()` report status and byte count and never body bytes, on any status. That stays. Probe gets its own explicitly-named path so the safe path remains the default and the body-reading one cannot be reached by accident.
- **Writes of any kind.** Probe is read-only; it never unlocks, syncs, sets, or rotates.
- **A general HTTP client.** Probe addresses registry-declared secrets through the existing client, not arbitrary URLs — otherwise it becomes `curl` with extra steps and re-opens the hazard it closes.
- **Redacting existing output elsewhere.** Other commands' output is not in scope.

## Risks / open questions

- **It must not become the leak it prevents.** A tool that inspects secret-bearing payloads is one careless flag from being the third instance's successor. Mitigation: the no-value rule is enforced by tests asserting a sentinel value never appears in output, not by reviewer attention.
- **`--raw` is the dangerous surface.** Bounded to non-2xx and capped; a test pins that a 2xx body never prints even with `--raw` set.
- **Seam ownership.** Probe must issue its request through `BWServeClient` — a hand-rolled call would diverge from the path it exists to diagnose and prove nothing about it. `bwserve.go` is another session's active file; the seam signature is agreed with that lane and landed by them, not edited here. Agreed 2026-08-15; new files only on this side.
- **Daemon cache (discovered 2026-08-15, `cli/internal/secrets/` lane).** `bw serve` answers reads from its own cache after a write, so a probe can report a stale value as current. Probe reports what the daemon says, and must not be read as proof of server state. Out of scope to fix; named here so the tool's output is not over-trusted.
- **Known dependency.** This spec's own archive gate runs `dotf spec review`, which shells through `dotf secrets run` with no `--only` — unreliable until #988 lands. The reviewer's fallback credential also lives in `~/.pi/agent/models.json` (#987), which the pending `NAN_API_KEY` rotation touches. Flagged, not solved here.

## Acceptance criteria

- [ ] `dotf secrets probe <id>` reports status, content-type, size, envelope validity, field names, value lengths and `sha256[:12]` fingerprints for a registry-declared secret.
- [ ] A test asserts a known sentinel secret value never appears in probe output, in any mode.
- [ ] A test asserts a 2xx body is never printed, including with `--raw`.
- [ ] `--raw` prints non-2xx bodies only, capped to a bounded length.
- [ ] The probe issues its request through `BWServeClient`, exercising the same transport and connection policy as the code path it diagnoses.
- [ ] `--count N` reports the distribution of outcomes across N attempts without printing any value.
- [ ] Probe performs no write: no unlock, sync, set, or rotate.
- [ ] `docs/lessons.md` names this command as the sanctioned probe, replacing the prose-only rule.

## References

- Bitácora board: `mlorentedev/dotfiles#1012` (see the `issue:` frontmatter field)
- #988 — the bug whose diagnosis motivated all three leak instances; owned by the `cli/internal/secrets/` lane
- #1007 — `call()` reporting status + byte count and never body bytes; the constraint this spec preserves
- #1004 — `verify` distinguishing "absent" from "backend failed to answer"; adjacent, same lane
- `docs/lessons.md` — "Redact at the producer, because the consumer's filter is a guess about a format you have not seen", and its recurrence note (PR #1013)
- `docs/adr/adr-028-secrets-two-tier-bitwarden-age.md` — the two-tier model and the `dotf secrets` facade this subcommand joins
