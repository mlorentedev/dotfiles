---
tags: [spec, tasks, secrets, cli, go, age, dr]
created: "2026-06-29"
---

# Tasks - CLI-024-secrets-backup

> TDD order. One task = one focused commit. Tick as you go.

## Setup

- [x] Branch created from main: `feat/secrets-dr-escrow` (external worktree)
- [x] `proposal.md` is complete and acceptance criteria are testable
- [x] No open questions left in `proposal.md` "Risks / open questions" (recipient + verify
  resolved by convention; mirror #454 + scheduling confirmed out-of-scope follow-ups)

## Implementation

- [ ] **AC5** — `env.RepoSensitiveDir() (string, error)`: the WRITE seam for `sensitive/`,
  fail-loud when no checkout (mirrors `RepoRegistryPath`). Tests:
  `TestRepoSensitiveDirPrefersRepoCheckout`, `TestRepoSensitiveDirNoCheckoutFailsLoud`.
- [ ] **AC2** — new seams in `cli/internal/secrets/escrow.go`:
  - `BWExporter` interface + `BWExport{Bin}` (`bw sync` + `bw export --format json --raw`,
    `--nointeraction`); the export analog of `BWGet`.
  - `Encryptor func([]byte, recipient string) ([]byte, error)` + `AgeEncrypt` (`age -r
    <recipient>`, plaintext via **stdin**, ciphertext on stdout — inverse of `AgeDecrypt`).
  - `RecipientFn func(keyPath string) (string, error)` + `AgeRecipient` (`age-keygen -y`).
  Their fakes' wiring is asserted with AC1/AC3; the shell-outs are live-smoke only (AC7).
- [ ] **AC1** — `secrets.Backup(cfg BackupConfig) (path string, err error)` core: derive
  recipient -> export -> encrypt -> atomic `0600` write to `DestDir/DestName`. Plaintext stays
  in memory (zeroed after encrypt, defense-in-depth); only ciphertext is written. Reuse the
  existing atomic-write helper (`render`'s) for cross-OS atomicity. Tests:
  `TestBackup_WritesCiphertext_0600`, `TestBackup_NoPlaintextOnDisk`.
- [ ] **AC3** — round-trip verify inside `Backup`: decrypt the written file via `Decrypt`,
  assert `bytes.Equal(plaintext, decrypted)` + `json.Valid` + non-empty; on failure
  `os.Remove(path)` and return a wrapped error. Tests: `TestBackup_RoundTripVerifyPasses`,
  `TestBackup_TamperedCiphertext_RemovesFileAndErrors`.
- [ ] **AC4** — fail-fast, no file written: exporter error (locked/unreachable bw) and
  recipient error (age key absent) each surface a clear, actionable message before any
  write. Tests: `TestBackup_ExporterError_NoFile`, `TestBackup_RecipientError_NoFile`.
- [ ] **AC1/AC4 (cmd)** — `newSecretsBackupCmd()` in `cli/internal/cmd/secrets_backup.go`:
  wires production seams (`BWExport`, `AgeEncrypt`, `AgeRecipient`, `ageDecryptor`), resolves
  dest via `RepoSensitiveDir()`/`dr`, prints the written path + verify result. Optional
  `--out <path>` override. Seams as overridable `var`s for command tests (fakes; no bw, no
  key). Register in `secrets.go` `AddCommand`. Tests: `TestSecretsBackup_HappyPath`,
  `TestSecretsBackup_LockedBw_Errors`.
- [ ] **AC6 (guard)** — `sensitive/dr/.gitignore` deny-all-except-`.age` (`*`, `!.gitignore`,
  `!*.age`) so a plaintext `*.json` export can never be staged; CI/bats assertion that a
  `sensitive/dr/foo.json` is ignored. (incident -> guard, same PR.)
- [ ] **AC6 (runbook, SSOT — no new file)** — update `guide-secrets-governance.md`: its
  BACKUP protocol now invokes `dotf secrets backup` (manual `bw…|age…` pipe demoted to the
  no-`dotf` fallback), RECOVER tightened (decrypt to tmpfs -> `bw import bitwardenjson` ->
  `dotf secrets verify`). A separate `guide-secrets-recover.md` would duplicate the existing
  RECOVER protocol (SSOT violation), so the #257 chain is hardened in place.

## Closing

- [ ] Every acceptance criterion is covered by >=1 test (AC7 shell-outs excepted — live smoke)
- [ ] `features.json` carries non-vacuous verification commands (state left `pending`)
- [ ] `go test ./... && go vet ./... && gofmt -l` clean; `go build ./...` clean
- [ ] `registry.yaml` unchanged (no secret flipped — confirm with `git diff`)
- [ ] Live smoke captured in `verification.md`: real `dotf secrets backup` on the unlocked
  vault produces a verifying escrow (the AC7 evidence)
- [ ] `verification.md` filled in
- [ ] PR opened referencing this spec folder

## Machine-readable features

See `features.json` (sibling). Each acceptance criterion maps to >=1 feature with an
executable `verification`. The agent may not set `"state": "passing"` — only the harness,
after capturing exit 0, may.
