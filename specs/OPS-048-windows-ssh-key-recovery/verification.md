---
tags: [spec, verification, templates]
created: "2026-09-24"
---

# Verification - OPS-048-windows-ssh-key-recovery

## Evidence

Map every acceptance criterion from `proposal.md` to concrete proof (commit hash, test name, or observed behavior).

- [x] AC1 -> Pester `accepts a matching private/public key pair and rejects a mismatch`, `rejects a private key that requires an interactive passphrase`, `removes every unrelated ACL entry from the private key`, and `repairs the private key ACL before loading the key`; live `dotf secrets run` reported fingerprint `SHA256:LqJwbtpkPE85CRuaqMuZkcHhA13NSguZvZmMUg5G618`.
- [x] AC2 -> Pester `adds the dedicated key once and preserves unrelated keys`, `replaces the previous managed identity during rotation`, and ACL tests; live host rotation followed by key-only SSH returned `WIN-9KP9GOBAT8N`.
- [x] AC3 -> Pester `defines stable mesh and LAN aliases with the dedicated identity`; live `acemagic-office-lan` SSH returned `WIN-9KP9GOBAT8N`.
- [x] AC4 -> Pester `documents the complete lifecycle and the trust boundary` and `requires Windows Pester evidence for every acceptance criterion`; the runbook covers bootstrap, reconciliation, recovery, rotation/revocation, DR, server host-key verification, and public-key-only checks.

## Test status

- Test suite: `Invoke-Pester -Path tests/windows-ssh-key-recovery.Tests.ps1 -CI` -> 15 passed, 0 failed.
- Relevant Go tests: `go test -count=1 ./internal/secrets ./cmd/dotf` -> passed.
- Lint: `Invoke-ScriptAnalyzer` with `.PSScriptAnalyzerSettings.psd1` over the two scripts and Pester suite -> no issues.
- Manual smoke test: provisioned the private key into Bitwarden through stdin, rematerialized it with `dotf secrets run`, validated its fingerprint/ACL, and connected through both direct LAN IP and `acemagic-office-lan`; both returned `WIN-9KP9GOBAT8N`.
- No regressions in relevant suites: yes. The complete Go suite has one environment-only Windows failure in `internal/doctor` because the current process lacks symlink privilege; all other packages pass and this change does not touch that package.

## Decisions made during implementation

Brief log of non-obvious trade-offs or course corrections taken during the work. Routine choices belong in commit messages, not here.

- `Set-Acl` requested `SeSecurityPrivilege` on the real private key despite temporary-file tests passing. DACL-only reconciliation now uses deterministic `icacls` operations and verifies the final SID set.
- Fingerprint equality did not prove non-interactive usability. The client guard now rejects passphrase-protected keys, and host rotation replaces the previous managed key by its stable comment while preserving unrelated keys.
- Client-key ACL reconciliation must precede every operation that loads private key material; otherwise the recovery path cannot repair a Bitwarden-materialized file with inherited access.
- Server host identity is verified through an authenticated console/RDP fingerprint before either alias is accepted, and recovery checks require public-key authentication in batch mode.

## Promotion candidates

Before archiving, flag what (if anything) should be promoted to the vault. If all three are "no", archive in repo is the only persistence.

- [x] Lesson for the repo's `docs/lessons/`? No - the Windows ACL and passphrase constraints are fully captured in tests and the operator runbook.
- [x] ADR-worthy decision for the repo's `docs/adr/adr-XXX.md`? No - this implements the existing ADR-028 secret-governance model.
- [x] New pattern candidate for `00_meta/patterns/`? No - no cross-project recurrence has been established.

## Archive checklist

- [ ] `proposal.md` frontmatter set to `status: archived`
- [ ] Folder moved: `specs/OPS-048-windows-ssh-key-recovery/` -> `specs/archive/OPS-048-windows-ssh-key-recovery/`
- [ ] Bitácora board ticket for this spec moved to Done / closed with PR link (ADR-018)
- [ ] Promotions above executed (if any)
