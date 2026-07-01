---
id: "CLI-024-secrets-age-discovery"
type: spec
status: draft # draft | implementing | verifying | archived
created: "2026-07-01"
issue: "mlorentedev/dotfiles#518"   # repo#NNN — GitHub issue / Project item that tracks this spec
tags: [spec, proposal, secrets, cli, go, age, doctor, dr]
template_version: "1.0"
---

# CLI-024-secrets-age-discovery

> **Naming**: file lives at `<repo>/specs/CLI-024-secrets-age-discovery/proposal.md`. A #586
> ADR-028 **Phase 4** follow-up slice: harden the **age root-of-trust** that the escrow
> (`dotf secrets backup`, #661) and every future recover leans on.

## Why

ADR-028 made the **age key the DR root-of-trust**: the escrow shipped in #661 is an
age-encrypted Bitwarden export, so on the worst day the *only* thing standing between a lost
account and a full recover is `age --decrypt` succeeding with the deployed key. Two gaps make
that succeed-or-fail *silently* today:

1. **Discovery is unwired.** #518 reports the failure mode directly: on a fresh Windows box the
   key deployed to `~/.config/age/key.txt`, but nothing pointed the age-consuming tools at it —
   `SOPS_AGE_KEY_FILE` was unset (SOPS defaults to `~/.config/sops/age/keys.txt`), so every
   decrypt failed until the env var was exported *by hand*, mid-task. The `env-contract` — the
   SSOT that `dotf env generate` renders into `paths.{sh,ps1}` — declares **no** age key path,
   so discovery depends on per-shell luck.
2. **The doctor check tests presence, not function.** `checkSecretsTooling` today only asserts
   the key *file exists* (`pathExists`) — a corrupt key, a wrong key, or a broken `age` binary
   all pass green and then fail at recover time, exactly when you cannot afford a surprise. A
   root-of-trust with no *functional* health check is a lock whose key you never test until the
   day you're locked out.

This slice closes both: it wires zero-config cross-platform discovery into the env-contract, and
upgrades the doctor check from "the file is there" to "the key actually round-trips". It is the
`incident → guard in the same change` rule applied to the DR floor the escrow already stands on.

## What

Concrete, observable changes after this PR:

1. **Age key path declared in the env-contract (discovery).** The dotfiles' own
   `env-contract.json` gains two `env_vars` entries — `AGE_KEY_PATH` (the var the dotfiles age
   flow's `ageKeyPath()` cascade already reads) and `SOPS_AGE_KEY_FILE` (the var the `sops`
   binary reads) — **both** defaulting to the deployed key across OSes
   (`$HOME/.config/age/key.txt` / `$env:USERPROFILE\.config\age\key.txt`). `dotf env generate`
   renders them into `paths.{sh,ps1}`, so every shell discovers the key with **zero per-shell
   config**. Two vars, one target: the dotfiles flow and any ad-hoc `sops` op (the literal #518
   bug) both resolve the same identity. `required: false`, **no `path_exists` validation** — a
   freshly provisioned box legitimately has no key until it is restored offline; declaring the
   *path* must not FAIL a healthy setup (the presence check owns that, as a WARN).

2. **Doctor secrets check upgraded to an end-to-end round-trip (guard).** When the key is
   present, `checkSecretsTooling` now derives the recipient (`age-keygen -y`), encrypts a fixed
   sentinel plaintext to it, decrypts the ciphertext back with the identity, and asserts it
   round-trips **byte-for-byte**. Verdicts:
   - key present **and round-trips** → **PASS** "age root-of-trust verified (round-trip)".
   - key present **but round-trip fails** (corrupt/wrong key, broken `age`) → **FAIL**, a red
     check with an actionable message — the silent failure #518 asks to surface.
   - key **absent** → **WARN** (unchanged) and the round-trip is **skipped**; a fresh box is not
     failed for a key it hasn't restored yet.
   - `age`/`age-keygen` **absent** → the existing `age` FAIL already covers it; the round-trip
     is skipped (no second, redundant failure).

3. **Round-trip reuses the existing age seams (no new production I/O path).** The verifier is a
   small helper wired from the `secrets` package's already-shipped seams — `AgeRecipient`
   (`age-keygen -y`), `AgeEncrypt` (`age -r`, stdin), `AgeDecrypt` (`age -d -i`, stdin) — the
   same three the escrow uses. The doctor check calls it through an injected function seam
   (mirroring the `System.HTTPGet` idiom the PAT check uses), so its tests inject fakes:
   **no real `age`, no key, no network in CI**. The sentinel plaintext is a fixed in-code
   constant — no committed fixture, no per-operator recipient baked anywhere.

No secret value is read, written, or flipped; `registry.yaml` and the escrow are untouched. This
PR adds *discovery wiring* + a *health check* over the existing key, nothing more.

## Out of scope

- **Physically rooting the key offline (#586 AC4 / ADR-028 §4).** Moving the authoritative
  `AGE-SECRET-KEY-*` to encrypted USB + paper + keychain is a manual operator action, not a
  codeable one. This slice makes the key *discoverable and verifiable*; where the master copy
  physically lives stays a runbook step.
- **Removing SOPS from the tool catalog.** SOPS is a deliberately-installed general tool
  (CLI-029) for per-project encrypted envs that consumes the *same* age key; it is retained, so
  its discovery var belongs here. Auditing whether it still earns its place in `packages.json`
  is a separate concern (ticket if pursued), not this PR.
- **Scheduling `dotf secrets backup`** (systemd timer / Task Scheduler) and the **off-box mirror
  (#454)** — the other #586 follow-up slices.
- **De-duplicating the escrow's internal round-trip verify.** `Backup` already round-trips its
  artifact; extracting a shared `secrets.VerifyRoundTrip` used by both is a tempting refactor but
  would edit shipped, tested code for no behavior gain — noted for later, kept out to hold this
  PR to one logical change.

## Risks / open questions

- **`path_exists` validation would FAIL a fresh box.** If the new env-contract entries carried
  `validation: path_exists`, a machine whose key isn't restored yet would fail the env-contract
  sweep. *Resolved:* declare the path with **no** validation (like `VAULT_PATH`); the WARN-level
  presence check remains the single owner of "key missing" semantics.
- **Round-trip needs the `age`/`age-keygen` binaries and a readable key.** *Mitigation:* the
  check is strictly additive — it runs the round-trip **only** when both binaries and the key are
  present; any absence falls through to the existing FAIL/WARN, never a new failure on an
  otherwise-healthy or legitimately-bare box.
- **A present-but-broken key must be loud, not silent.** That is the whole point: a decrypt
  mismatch is a **FAIL** (red check), carrying the key path and the age error, so the operator
  learns at setup time — not at recover time — that the root-of-trust is dead.
- **CI must stay hermetic.** The production `age` I/O is thin and stdin-piped; *Mitigation:* the
  doctor test injects a fake round-trip seam (PASS / FAIL / mismatch), and the real
  `AgeEncrypt`/`AgeDecrypt`/`AgeRecipient` are already covered by the escrow's live smoke — the
  same "seams in CI, live I/O in a smoke" split the repo already uses.
- **Resolved by the codebase (not open):** SOPS-vs-age — SOPS is a retained general tool on the
  same key (CLI-029, `docs/lessons.md`), so declaring `SOPS_AGE_KEY_FILE` is correct, not cruft;
  sentinel source — a fixed in-code constant (round-trip proves the *key*, not the *plaintext*).

## Acceptance criteria

Observable outcomes. Each must be testable without a real `age`/key in CI.

- [ ] **AC1** — the dotfiles' own `env-contract.json` declares `AGE_KEY_PATH` and
  `SOPS_AGE_KEY_FILE`, `required: false`, no `path_exists` validation, with cross-OS defaults
  resolving to `~/.config/age/key.txt`; `dotf env generate` renders both into `paths.{sh,ps1}`.
  The generic `dotf init` starter template (`initrepo/templates/env-contract.json`) is
  **intentionally not** touched — the age key is a dotfiles machine fact, not a universal
  per-repo expectation. *Verify:* `env-contract.bats` asserts both entries + their OS defaults;
  a `jq` check asserts neither carries `path_exists`; the starter template stays `env_vars: []`.
- [ ] **AC2** — with the key present and a healthy identity, the secrets-tooling check emits a
  **PASS** naming the verified round-trip. *Verify:* doctor test injecting a fake round-trip seam
  that succeeds → report contains the PASS line, exit-relevant status ok.
- [ ] **AC3** — with the key present but the round-trip failing (mismatch / age error), the check
  emits a **FAIL** with the key path + cause. *Verify:* doctor test injecting a failing seam →
  report contains the FAIL and the run is non-green.
- [ ] **AC4** — with the key **absent**, the check emits the existing **WARN** and does **not**
  attempt the round-trip (no FAIL). *Verify:* doctor test with no key file → WARN present, the
  round-trip seam is never invoked.
- [ ] **AC5** — the round-trip verifier is wired from the existing `secrets` age seams and is
  unit-tested with fakes only. *Verify:* the new/changed Go tests reference the seam, and
  `grep` confirms no `exec.Command("age"...)` is introduced in the doctor package.
- [ ] **AC6** — `go vet ./...` + `gofmt -l` clean module-wide, the touched packages
  (`internal/doctor`, `internal/env`) pass `go test`, and `env-contract.bats` is green. (The
  whole-module `go test ./...` additionally passes **except** the pre-existing, unrelated
  `internal/spec` `TestEmbeddedTemplatesMatchVault` template-drift failure — out of scope here,
  tracked for its own re-vendor.) The production age round-trip (thin `age`/`age-keygen` I/O) is
  covered by a **live smoke** with a real key, not in CI (parity with `AgeDecrypt`/`BWGet`).

## References

- Issue: `mlorentedev/dotfiles#518` (SOPS age-key auto-discovery + doctor check, cross-platform).
- Parent: `mlorentedev/dotfiles#586` (ADR-028 Phase 4 curation — this hardens the age
  root-of-trust the offline-key move / escrow depend on).
- ADR: `docs/adr/adr-028-secrets-two-tier-bitwarden-age.md` (§4 offline age key; §5 escrow;
  §Mitigations names the doctor secrets check as the governance hook).
- Reuse: `cli/internal/secrets/{escrow,resolve}.go` (`AgeRecipient`/`AgeEncrypt`/`AgeDecrypt`
  seams), `cli/internal/cmd/secrets.go` (`ageKeyPath()` cascade),
  `cli/internal/doctor/{checks_secrets_tooling,system,checks_pat}.go` (the check being upgraded +
  the `System` seam idiom + the `HTTPGet` injection pattern to mirror), `env-contract.json` +
  `cli/internal/initrepo/templates/env-contract.json` (the two contract copies).
- Prior merged slice: `CLI-024-secrets-backup` (#661 — the escrow this guards; same seam/test idiom).
- Related patterns: `00_meta/patterns/secrets-security.md`, `pattern-spec-driven-development.md`.
