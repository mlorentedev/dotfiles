---
id: "OPS-026-age-root-file-authority"
type: spec
status: draft # draft | implementing | verifying | archived
created: "2026-08-19"
issue: "mlorentedev/dotfiles#937"   # repo#NNN — GitHub issue / Project item that tracks this spec
tags: [spec, proposal]
template_version: "1.0"
---

# OPS-026-age-root-file-authority

> **Naming**: file lives at `<repo>/specs/OPS-026-age-root-file-authority/proposal.md`. `OPS-026-age-root-file-authority` is `AREA-NNN-slug` (e.g. `TOOL-001-secret-drift`).

## Why

<!-- from issue #937: OPS-026: the age key that decrypts every secret is itself absent from the registry -->

`secrets/registry.yaml` is ADR-028's mapping SSOT, and the one secret it does not
manage is the one that decrypts all the others. The age identity at
`~/.config/age/key.txt` has no plane, no expose contract, no declared off-machine
copy and no rotation cadence — it is the root of the hierarchy and it is invisible
to every command that reports on secrets. That matters beyond tidiness: 28 of 33
secrets currently resolve through Bitwarden, and the DR escrow that would survive
losing that account is encrypted with this key (#1000). An inventory that omits its
own root cannot tell anyone the root is at risk.

## What

The registry gains a backend, `file-authority`, for a secret whose authority **is**
the local plaintext file. Nothing resolves it — there is no ciphertext to decrypt
and no remote to fetch from — so `verify` asks the only questions that mean anything
for a root: does the file exist, is its mode `0600`, and does its public fingerprint
match the declared off-machine copy.

Observable afterwards:

- `dotf secrets ls` lists `age-key-personal` alongside the 33 it already shows.
- `dotf secrets verify` reports it OK/MISSING/FAILED **without attempting a decrypt**,
  and a drifted or wrong-key copy reports FAILED rather than OK.
- ADR-028's worked example is amended to something the validator accepts.

## Out of scope

- **Making the off-machine copy.** That is a physical act — printed, on a USB, in a
  second manager — and belongs to the operator (#1000). This spec makes its absence
  and its drift *visible*; it cannot create it.
- **Rotating the age identity.** Re-encrypting five `backend: age` secrets and the DR
  escrow under a new key is its own operation with its own failure modes (#996, #938).
- **`dotf secrets run` exposing the root.** Handing the key that decrypts everything
  to a child process through the same facade as the secrets it protects is a widening
  of blast radius, not a convenience. The `expose.file` contract records where the
  file belongs; materialising it through `run` is deliberately not added.
- **The drift comparison.** Checking that the local key still matches the copy held
  off this machine is #1000's own AC3, not this spec's. It was drafted here and moved
  out on the way: it needs a live Bitwarden session and `age-keygen -y`, and — now
  that the off-machine copy is an offline USB rather than a vault item — the check
  that actually pays is comparing the local key against a **declared public
  recipient**, which needs no session at all. That is a design decision belonging to
  #1000. `fileAuthorityResolver`'s comment states that its check answers a narrower
  question until then, so the gap cannot be read as coverage.
- **Multi-root support.** One root, one machine. A second machine with its own
  identity is a real future case and is explicitly not designed for here.

## Risks / open questions

- **A new backend must not become a hole in the resolver.** `ValidBackends()` is bound
  to the Loader by a resolver-coverage test; a backend that resolves nothing has to be
  excluded from that binding *deliberately and visibly*, or the test stops meaning what
  it says. This is the same class as GUARD-002's own defects — a check that quietly
  answers a cheaper question.
- **The fingerprint check needs a live Bitwarden session.** `verify` must degrade
  honestly when there is none: the mode and existence checks still run, and the
  fingerprint comparison reports SKIPPED with a reason — never OK. An unchecked copy
  reported as matching is the exact failure this exists to prevent.
- **`age-keygen -y` is the only fingerprint source, and it reads the private key.**
  The public recipient string is safe to print and compare; the private key must never
  reach a log, an error message or a test fixture. Comparison happens on the derived
  public value only.
- **Resolved, and worth recording:** the ADR's ratified example does not validate today
  — `age-offline` with a file expose requires an `age:` source (`registry.go:342`),
  `expose.file` requires `var`, and `offline:` is not a field. That is why this issue
  aged rather than being "simply never added".

## Acceptance criteria

- [ ] AC1 — `file-authority` is in `ValidBackends()`, and `validateSecret` accepts a
      secret declaring it with a file expose and **no** `age:` source, while still
      rejecting one that omits `var` or `path`.
- [ ] AC2 — `age-key-personal` exists in `secrets/registry.yaml` as `plane: floor`,
      `backend: file-authority`, with the `bw:` block naming the convenience copy, and
      `dotf secrets ls` shows it.
- [ ] AC3 — `dotf secrets verify` reports the root without attempting a decrypt, and
      the pre-existing 33 entries still report `33 ok, 0 missing, 0 failed`.
- [ ] AC4 — a wrong-mode file (e.g. `0644`) reports FAILED, observed by mutating the
      mode and seeing the check fail, with the mutation confirmed present first.
- [ ] AC6 — the resolver-coverage test states in its own text why `file-authority` has
      no Loader entry, so a future reader cannot mistake the exclusion for an oversight.
- [ ] AC7 — ADR-028's worked example (lines 134-139) is amended to the shape the
      validator accepts, and says why the root cannot be `age-offline`.

## References

- Bitácora board: `mlorentedev/dotfiles#937` (see the `issue:` frontmatter field)
- `#1000` — the circular dependency this makes visible: the escrow's key has no copy
  outside the vault the escrow exists to replace
- Related ADR: `docs/adr/adr-028-secrets-two-tier-bitwarden-age.md` §2 and lines 134-139
- Related patterns: `00_meta/patterns/pattern-verification-fails-toward-unproven.md`
