---
id: "CLI-024-secrets-backup"
type: spec
status: draft # draft | implementing | verifying | archived
created: "2026-06-29"
issue: "mlorentedev/dotfiles#586"   # repo#NNN — GitHub issue / Project item that tracks this spec
tags: [spec, proposal, secrets, cli, go, age, bitwarden, dr]
template_version: "1.0"
---

# CLI-024-secrets-backup

> **Naming**: file lives at `<repo>/specs/CLI-024-secrets-backup/proposal.md`. Slice 1 of
> ADR-028 **Phase 4** (#586): the repo-side DR foundation the vault-side curation depends on.

## Why

ADR-028 §5 makes the **age-encrypted Bitwarden escrow** the thing that turns "lose the
Bitwarden account → lose every secret (API keys, tokens, *and* the TOTP/2FA seeds)" into
"recover everything from an offline age key + a repo clone". The ratified Consequences
(§1, §3) sequence it first: the escrow (`dotf secrets backup`) and the recover runbook
**must exist before** the least-reversible Phase-4 mutations — moving the `AGE-SECRET-KEY-*`
offline (#518), restructuring live items — because they are the safety net those operations
lean on. Today the escrow is a **manual command a runbook tells you to type**
(`docs/runbooks/guide-secrets-governance.md:49`), which violates Standing Order #1
(automate, don't instruct) and is the kind of step that silently never runs. There is no
`dotf secrets backup`. This spec builds it, plus the `recover` runbook (#257) it pairs with.

## What

Concrete, observable changes after this PR:

1. **`dotf secrets backup` subcommand.** One idempotent invocation runs the escrow end to
   end: `bw sync` → `bw export --format json --raw` → encrypt to the operator's own age
   recipient → **atomic, `0600`** write to `sensitive/dr/bitwarden-export.age` in the
   dotfiles checkout. The plaintext vault export is **piped in memory** (exporter stdout →
   encryptor stdin) and **never touches disk**; only the ciphertext is written. The
   command is the unit a scheduler later calls — no manual steps.

2. **Recipient derived from the identity, not hardcoded or duplicated.** The age recipient
   is `age-keygen -y <keyPath>` — the repo's established self-encrypt convention (every
   existing encrypt site does exactly this: `ci.yml`, `ai/nan/README.md`,
   `docs/runbooks/secrets-management.md`, `scripts/age-standalone.sh`). The private key
   stays the **single source**; no new committed recipients file, no `age1…` literal in
   code or config. Key path resolves through the existing `ageKeyPath()` cascade
   (`$AGE_KEY_PATH` → `~/.config/age/key.txt`).

3. **Two new testable seams, mirroring the existing ones (Open/Closed).**
   - `BWExporter` (`Export() ([]byte, error)`) — production `BWExport{Bin}` shells
     `bw sync` + `bw export --format json --raw` with `--nointeraction`; the age analog of
     `BWGet`. A locked / unreachable vault fails fast with bw's stderr surfaced.
   - `Encryptor` (`Encrypt(plaintext []byte, recipient string) ([]byte, error)`) +
     `AgeRecipient(keyPath)` — production `AgeEncrypt` shells `age -r <recipient>` reading
     plaintext from **stdin** (never a temp file); the inverse of `AgeDecrypt`.
   - The round-trip verify **reuses the existing `Decryptor`/`AgeDecrypt`** — no new seam.
   Command-layer tests inject fakes → **no `bw`, no age key, no network in CI** (the exact
   coverage shape `fakeDecryptor`/fake `BWReader` already give).

4. **Built-in verified round-trip.** After writing, the command decrypts the artifact with
   the identity and asserts it round-trips to the **exact** exported bytes and parses as a
   **non-empty** Bitwarden export JSON. A verify failure **removes the just-written file**
   and exits non-zero — a corrupt or empty escrow is never left on disk or committed. This
   is the "escrow (and its verified round-trip)" that ADR-028 §3 gates age-file retirement
   (C6) on.

5. **`RepoSensitiveDir()` write-seam** (fail-loud when no checkout), mirroring
   `RepoRegistryPath` (#635 / ADR-030). The escrow is version-controlled state, so it must
   land in the checkout to be committed — never the throwaway deployed copy under
   `~/.dotfiles`. No checkout → a clear refusal, not a silent write to the deployed copy.

6. **Recover runbook made real — no new file (SSOT).** `guide-secrets-governance.md`
   already owns the ADD/ROTATE/BACKUP/RECOVER protocols, so this PR **updates** its BACKUP
   section to invoke `dotf secrets backup` (the manual `bw…|age…` pipe demoted to the
   no-`dotf` fallback) and tightens RECOVER (decrypt to tmpfs → `bw import bitwardenjson` →
   `dotf secrets verify`). A separate `guide-secrets-recover.md` would duplicate that
   protocol — a SSOT violation (Standing Order #2), so the recover chain (#257) is hardened
   in place rather than forked.

`registry.yaml` is **not** edited and no secret is flipped: this PR adds the escrow
*capability* + its recover doc, with zero change to live secret resolution.

## Out of scope

- **Off-box mirror of the escrow (#454).** This PR writes + commits the artifact in-repo;
  pushing it to a second off-box destination is OPS-015/#454's job — the **next** slice.
- **Scheduling automation** (systemd timer / Windows Task Scheduler). The command is the
  idempotent unit a scheduler invokes; wiring the timer is a thin, separable follow-up
  slice (tracked), kept out to hold this PR to one logical change (atomic-PR rule). "It is
  automatable" is satisfied here; "it is scheduled" is the follow-up.
- **The offline age-key move (#518), folder taxonomy, per-purpose token split (#321),
  naming-drift fixes** — the other #586 workstreams. This is the foundation they depend on,
  not them.
- **`bw serve` daemon lifecycle** — a perf upgrade behind the `BWExporter` seam, not
  correctness. Deferred (same posture as the bw-backend spec).
- **Pruning / rotation of old escrow snapshots** — the artifact is a single
  rolling file (git history is the version trail); a retention policy is a later concern.

## Risks / open questions

- **Whole-vault plaintext in process memory.** Between export and encrypt the full vault
  plaintext lives in a Go `[]byte`. *Mitigation:* it is piped, **never written to disk**;
  the process is short-lived; this is the same exposure shape as `AgeDecrypt` returning
  plaintext in memory. Defense-in-depth (zero the buffer after encrypt) is a cheap add,
  noted in `tasks.md`.
- **Plaintext escrow must never be committed.** The artifact is ciphertext by construction
  (atomic write of the encryptor output only). *Mitigation:* a guard so a plaintext
  `sensitive/dr/*.json` can never be staged — fits the repo's "incident → guard in the
  same PR" rule; covered in `tasks.md` (a `.gitignore` entry + a CI/pre-commit assertion).
- **Key absent on the box.** `age-keygen -y` needs the private identity; a box without it
  cannot encrypt the escrow. *Mitigation:* fail fast with an actionable message
  ("age identity required at <path> to encrypt the escrow — restore the key first"), never
  a hang. A backup box legitimately holds the convenience key copy (ADR-028 §4); recover
  needs the key anyway.
- **Locked / unreachable Bitwarden.** *Mitigation:* `bw export` runs `--nointeraction` and
  surfaces bw's stderr fail-fast (parity with `BWGet`); the atomic write + verify mean no
  partial or empty escrow is ever produced.
- **`bw export` schema/version drift.** The JSON shape could change across bw releases.
  *Mitigation:* verify asserts it parses + is non-empty (not a deep schema check — that
  would couple us to bw's internal format); the recover runbook pins
  `bw import --format bitwardenjson`.
- **Resolved by convention (not open):** recipient source = `age-keygen -y` (repo
  convention, single key); round-trip verify = always (the key is present by construction).
  Neither needs a decision.

## Acceptance criteria

Observable outcomes. Each must be testable with the seams above (no real `bw`/age in CI).

- [ ] **AC1** — `dotf secrets backup` orchestrates `BWExporter` → `Encryptor` → atomic
  write, producing `sensitive/dr/bitwarden-export.age` at mode `0600`; the exported
  plaintext is **never** written to disk. *Verify:* command test with fake exporter + fake
  encryptor asserts the file holds the ciphertext, mode is `0600`, and no plaintext file
  appears in the dir.
- [ ] **AC2** — the recipient is obtained via the `AgeRecipient` (`age-keygen -y`) seam and
  passed to the encryptor; no recipient literal exists in code/config. *Verify:* test with
  a fake recipient asserts it is threaded into the encrypt call.
- [ ] **AC3** — after writing, the artifact is decrypted (via `Decryptor`) and must
  round-trip to the exact exported bytes and parse as non-empty JSON; on a verify failure
  the file is **removed** and the command exits non-zero. *Verify:* table test (clean
  round-trip passes; tampered ciphertext → file deleted + error).
- [ ] **AC4** — a locked/unreachable Bitwarden (exporter error) and a missing age key
  (recipient error) each fail fast with a clear, actionable message and **no file written**.
  *Verify:* two command tests with erroring fakes.
- [ ] **AC5** — the write lands in the checkout via `RepoSensitiveDir`; no checkout → a
  fail-loud refusal (never the deployed copy). *Verify:* env test mirroring
  `RepoRegistryPath`'s repo-vs-deployed coverage.
- [ ] **AC6** — a plaintext `sensitive/dr/*.json` cannot be committed (guard), and
  `guide-secrets-governance.md`'s BACKUP protocol invokes `dotf secrets backup` (the manual
  pipe demoted to the no-`dotf` fallback), with RECOVER tightened (#257). *Verify:*
  `git check-ignore` on a `.json` under `sensitive/dr/` + `grep 'dotf secrets backup'` of
  the runbook.
- [ ] **AC7** — `go test ./... && go vet ./... && gofmt -l` clean; the production
  `BWExport`/`AgeEncrypt` (thin I/O to `bw`/`age`) are covered by a **live smoke** with the
  operator's unlocked session, not in CI (parity with `BWGet`/`AgeDecrypt`).

## References

- ADR: `docs/adr/adr-028-secrets-two-tier-bitwarden-age.md` (§5 escrow; Consequences §1
  repo-side-first, §3 retirement gated on the verified round-trip; §4 offline age key).
- Issue: `mlorentedev/dotfiles#586` (Phase 4 curation — this delivers the DR-escrow AC +
  the recover runbook; folders / #321 split / #518 offline key are follow-up slices).
- Related issues: #257 (recover runbook), #454 (off-box mirror — next slice), #518
  (offline age key — gated on this).
- Reuse: `cli/internal/secrets/{resolve,bw}.go` (`Decryptor`/`AgeDecrypt`, `BWGet`,
  `BWReader` seam pattern), `cli/internal/cmd/secrets.go` (`ageKeyPath`, seam `var`s),
  `cli/internal/env/env.go` (`ResolveSensitiveDir`, `RepoRegistryPath` write-seam pattern).
- Prior merged specs: `CLI-024-secrets-bw-backend` (the seam/test idiom this mirrors),
  `CLI-024-secrets-registry`, `CLI-025-secrets-render`.
- Runbook this automates: `docs/runbooks/guide-secrets-governance.md` (§ the manual escrow
  line this replaces).
