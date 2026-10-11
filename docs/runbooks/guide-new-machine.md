---
id: "dotfiles-runbook-new-machine"
type: runbook
status: active
tags: [runbook, dotfiles, bootstrap, converge, doctor, secrets, identity]
created: "2026-10-11"
updated: "2026-10-11"
owner: manu
---

# New Machine

How a machine goes from factory-fresh to fully working, and which part of that is yours to do. `dotf converge` installs and configures everything the repository declares. Your **identity** is the one part it cannot produce: the age key, the Bitwarden session, the GitHub login and the knowledge vault. Those are restored once per machine, in the order below, because each step needs the one before it (#2013 D10).

## 1. Converge the machine

On Linux or macOS (Windows: `install.ps1`, see the README):

```bash
curl -fsSL https://raw.githubusercontent.com/mlorentedev/dotfiles/main/install.sh | bash
```

`install.sh` installs `dotf` and hands the machine to `dotf converge`; every argument after `bash -s --` goes to converge. To see what it would do first, without changing anything:

```bash
curl -fsSL https://raw.githubusercontent.com/mlorentedev/dotfiles/main/install.sh | bash -s -- --plan
```

**Reading the report.** Converge prints one line per step, in order, tagged `[ OK ]` (already converged), `[CHANGE]` (it changed something), `[SKIP]` (the step does not apply here, with the reason), `[OPAQUE]` (a setup script that cannot say what it would change: it runs, and counts as neither) or `[FAIL]` (the run stops there), then a one-line summary. An applied run is also saved to `~/.local/state/dotfiles/converge/last.json` (under `$XDG_STATE_HOME/dotfiles` when that is set), and converge prints the path; a plan is never saved. On macOS `legacy-setup` is always `[SKIP] not supported on darwin`: macOS has no setup script, and every step there is native (ADR-045). A second `dotf converge` on a converged machine reports 0 changed.

Then check what converge produced, leaving identity aside:

```bash
dotf doctor --scope machine
```

Every identity check prints a SKIP naming the step below that restores it. A FAIL here is a defect of the bootstrap, not of your restore, and belongs in a ticket. The from-zero CI job runs this same command on fresh macOS and Ubuntu runners (`.github/workflows/from-zero.yml`).

## 2. Restore the identity

The step numbers are the ones `dotf doctor --scope machine` cites.

**Step 1: the age key.** It decrypts the offline floor, so nothing below starts without it. Restore it from the offline backup to `~/.config/age/key.txt` (or `AGE_KEY_PATH`), following [Secrets Governance § RECOVER](guide-secrets-governance.md#protocol--recover-disaster), step 1. Generate a new key only for a brand-new identity ([§ FIRST MACHINE](guide-secrets-governance.md#protocol--first-machine-age-key--ssh)).

**Step 2: Bitwarden.** It is the live store every `dotf secrets` read resolves through (ADR-028).

```bash
bw login                 # once per machine: email, master password, second factor
dotf secrets unlock      # once per boot: starts the bw serve daemon every terminal shares
dotf secrets verify      # every registry secret resolves (no values printed)
```

Use `bw login`, not `bw unlock`, on a new machine or after a session expired: an expired refresh token cannot be unlocked.

**Step 3: GitHub.**

```bash
gh auth login
dotf doctor --fix        # points git's credential helper at gh by absolute path
```

**Step 4: the knowledge vault.**

```bash
git clone https://github.com/mlorentedev/knowledge.git "$(dotf env path VAULT_PATH)"
```

## 3. Converge again, then the full doctor

```bash
dotf converge
dotf doctor
```

The second converge deploys what waited on the secrets. For example, `opencode.jsonc` is rendered from them, and converge reports it `kept (secrets locked)` until step 2. The full `dotf doctor` must now pass, identity checks included. From here on, a reboot needs only `dotf secrets unlock`.

## Related

- [Secrets Governance](guide-secrets-governance.md): the age key, Bitwarden, backup and recover protocols
- [Tool Installation](tool-installation.md): how converge installs each tool
- ADR-045: one entrypoint converges every OS
