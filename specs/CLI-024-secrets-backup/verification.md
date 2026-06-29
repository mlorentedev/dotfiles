---
tags: [spec, verification, secrets, cli, go, age, dr]
created: "2026-06-29"
---

# Verification - CLI-024-secrets-backup

## Evidence

Run: `cd cli && go test ./internal/secrets/ ./internal/cmd/ ./internal/env/` → all `ok`.

| AC | Behavior | Tests (all PASS) |
|----|----------|------------------|
| AC1 | escrow written 0600; plaintext never on disk | `TestBackup_WritesCiphertext_0600`, `TestBackup_NoPlaintextOnDisk` |
| AC2 | recipient derived via seam, threaded to encryptor | `TestBackup_RecipientThreadedToEncryptor` |
| AC3 | round-trip verify; tamper/non-JSON → remove + error | `TestBackup_RoundTripVerifyPasses`, `TestBackup_TamperedCiphertext_RemovesFileAndErrors`, `TestBackup_NonJSONExport_FailsVerify` |
| AC4 | locked bw / missing key → fail fast, no file | `TestBackup_ExporterError_NoFile`, `TestBackup_RecipientError_NoFile`, `TestBackup_EmptyExport_Refused`, `TestBackup_NilExporter_Errors` |
| AC5 | `RepoSensitiveDir` prefers checkout, fails loud | `TestRepoSensitiveDirPrefersRepoCheckout`, `TestRepoSensitiveDirNoCheckoutFailsLoud` |
| AC1/AC4 (cmd) | happy path / locked bw / no checkout / `--out` | `TestSecretsBackup_HappyPath`, `TestSecretsBackup_LockedBw_Errors`, `TestSecretsBackup_NoCheckout_FailsLoud`, `TestSecretsBackup_OutFlagOverridesDest` |
| AC6 (guard) | plaintext `.json` unstageable; `.age` trackable | `git check-ignore` (below) |
| AC6 (runbook) | governance runbook invokes the real command | `grep 'dotf secrets backup'` (below) |

`go test ./internal/secrets/ -run Backup -v` → 10/10 PASS (captured 2026-06-29).

**Guard (AC6):** `git check-ignore -v sensitive/dr/bitwarden-export.json` → matched by
`sensitive/dr/.gitignore:5:*`; `git check-ignore sensitive/dr/bitwarden-export.age` → exit 1
(the `.age` escrow stays committable).

**Runbook (AC6):** `grep 'dotf secrets backup' docs/runbooks/guide-secrets-governance.md` →
matches; RECOVER tightened in place. No `guide-secrets-recover.md` (would duplicate).

## Test status

- `go build ./...` clean; `go vet ./internal/secrets/ ./internal/cmd/ ./internal/env/` clean.
- Manual smoke: `dotf secrets backup --help` renders the description + `--out` flag (wiring
  confirmed, no `bw`/`age` invoked).
- No regressions: the affected packages are green. (Pre-existing, unrelated: `internal/spec`
  `TestEmbeddedTemplatesMatchVault` fails on `main` too — vendored `templates/tasks.md` drift
  vs vault `spec-tasks.md`; not touched here, needs its own re-vendor change.)
- `git diff --stat -- secrets/registry.yaml` → empty (no secret flipped).
- `gofmt`: the four new files are LF + formatted (absent from `gofmt -l`). The Windows working
  tree shows CRLF on pre-existing files; `.gitattributes` `* text=auto` normalizes commits to
  LF, so the Linux CI `lint` job is clean.

### Pending — live smoke (AC7, operator-run; needs an unlocked vault)

```
bw unlock           # export BW_SESSION
dotf secrets backup # → escrow written and verified: <checkout>/sensitive/dr/bitwarden-export.age
age -d -i ~/.config/age/key.txt sensitive/dr/bitwarden-export.age | head -c 80   # sanity: JSON
```

Paste the (redacted) output here once run, then tick the AC7 box in `tasks.md`.

## Decisions made during implementation

- **No separate `guide-secrets-recover.md` (SSOT).** `guide-secrets-governance.md` already
  owns the RECOVER protocol; a second file would duplicate it (Standing Order #2). Hardened
  the existing BACKUP/RECOVER sections in place instead — a course correction from the initial
  proposal sketch after reading the runbook in full.
- **Recipient by convention, not a new file.** `age-keygen -y` of the identity (what every
  other encrypt site in the repo does) — the private key stays the single source; no
  `age-recipients.txt` introduced.
- **Order = no-partial-file guarantee.** recipient → export → encrypt all run before any write;
  verify-then-remove on failure. A locked vault / absent key leaves the disk untouched.

## Promotion candidates

- [ ] Lesson for `docs/lessons.md`? Maybe — "self-encrypt escrow needs the key present, so
  verify is free; a keyless-backup design would need a committed recipients file" (decide at PR).
- [ ] ADR-worthy? No — ADR-028 already ratified the escrow design; this implements §5.
- [ ] New `00_meta/patterns/` pattern? No — repo-specific.

## Archive checklist

- [ ] `proposal.md` frontmatter set to `status: archived`
- [ ] Folder moved: `specs/CLI-024-secrets-backup/` → `specs/archive/CLI-024-secrets-backup/`
- [ ] Bitácora board ticket (#586 slice) updated with the PR link (ADR-018)
- [ ] Promotions above executed (if any)
