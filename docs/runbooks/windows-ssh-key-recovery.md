---
id: "dotfiles-runbook-windows-ssh-key-recovery"
type: runbook
status: active
tags: [runbook, windows, openssh, bitwarden, recovery]
created: "2026-09-24"
owner: manu
---

# Windows SSH Key Recovery

This runbook manages the dedicated `acemagic-office` SSH identity. Bitwarden is
the private-key SSOT; git contains only the public key and idempotent
reconciliation scripts.

## Security boundary

The first authorization requires an already authenticated administrative channel
such as local console or RDP. Automation cannot create its own initial trust.
Windows OpenSSH uses `administrators_authorized_keys` for administrator accounts,
so the dedicated key can authenticate administrator-group accounts, not only
`manu`. Do not reuse it elsewhere.

Never commit, print, or place the private key in a command argument. Feed it to
`dotf secrets set` through stdin or its hidden prompt.

## First trust bootstrap

From the dotfiles checkout on the Windows host, open an elevated PowerShell and
run:

```powershell
pwsh -NoProfile -File .\scripts\windows-openssh-authorize.ps1
```

The script installs and enables OpenSSH Server, creates or enables the port 22
firewall rule, adds the committed public key exactly once, preserves unrelated
keys, protects the administrator key store ACL, and restarts `sshd` only when the
key store changed.

## Normal reconciliation

Re-run the same host command after an OpenSSH repair or policy change. A repeated
run is safe and must report `authorized key changed: False`.

On the client, materialize and validate the private key:

```powershell
dotf secrets run --only ACEMAGIC_OFFICE_SSH_KEY -- `
  pwsh -NoProfile -File .\scripts\windows-ssh-client-key.ps1
ssh acemagic-office hostname
```

This dedicated automation key must not require an interactive passphrase. Its
protection comes from Bitwarden at rest plus the protected current-owner and
SYSTEM Windows ACL after materialization. The client guard rejects an encrypted
key rather than allowing SSH to fail later during signing.

Use `ssh acemagic-office-lan` when directly connected to the dock LAN.

## Clean-machine recovery

1. Clone dotfiles and authenticate/unlock the Bitwarden CLI.
2. Run the `dotf secrets run` command above. The file secret materializes at
   `~/.ssh/id_ed25519_ts_bridge_acemagic`.
3. The client guard compares private and committed-public fingerprints and
   verifies non-interactive signing, then replaces inherited ACLs with
   current-owner and SYSTEM access.
4. Verify both aliases with `hostname`; never verify by printing key material.

## Rotation and revocation

1. Generate a new dedicated keypair without a passphrase at a temporary local
   path: `ssh-keygen -t ed25519 -N '' -f <temporary-path>`.
2. Replace the committed `.pub` file and review the fingerprint change.
3. Reconcile the host through the existing authenticated channel before replacing
   the Bitwarden private key.
4. Write the new private key with `dotf secrets set
   ACEMAGIC_OFFICE_SSH_KEY --yes`, using stdin or the hidden prompt.
5. Materialize, run the client guard, and verify SSH.
6. Remove obsolete key material from every workstation and refresh the DR escrow.

If compromise is suspected, remove the old public key from the host first and
restore access through RDP/local console with a newly generated identity.

## Disaster recovery

If Bitwarden is unavailable, restore the age key from its offline authority and
recover the Bitwarden escrow by following
[Secrets Governance](guide-secrets-governance.md#protocol--recover-disaster).
After importing the escrow, follow **Clean-machine recovery**. The private key is
not independently duplicated in git or another ad-hoc file.

## Verification

```powershell
ssh-keygen -lf .\ssh\id_ed25519_ts_bridge_acemagic.pub
ssh acemagic-office hostname
ssh acemagic-office-lan hostname
```

Expected public fingerprint:
`SHA256:LqJwbtpkPE85CRuaqMuZkcHhA13NSguZvZmMUg5G618`.
The host reconciler's second run reports no key-store change, and the client guard
reports the same fingerprint without displaying private material.
