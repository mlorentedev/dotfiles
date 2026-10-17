---
id: "dotfiles-runbook-workstation-ssh-target"
type: runbook
status: active
tags: [runbook, ssh, tailscale, workstations, security, power-management]
created: "2026-10-12"
updated: "2026-10-13"
owner: manu
---

# Workstation SSH Target

This runbook turns a workstation (`macmini`, `msi`, or a Windows host) into a
remote development target that is reachable over SSH **through the Headscale
mesh and nowhere else**. The client side (`ssh/config`) is converged by `dotf`.
On Linux and macOS, `dotf converge --only workstation-target` applies the sshd
half of section 2, and on Linux the UFW rule of section 3 and the lid policy of
section 4.

> **Still by hand:** the firewall (section 3) on macOS and Windows, power on macOS, and
> everything on Windows. Each is a row of
> [#2356](https://github.com/mlorentedev/dotfiles/issues/2356) (SEC-011, PLAT-001b
> PR 7d, PR 7e): it moves into the same converge step and leaves this runbook.
> Until then, every hand step below pairs its change with a check. Do not skip
> the checks.

## Security model

| Layer | Linux | macOS | Windows |
|---|---|---|---|
| Network | UFW: port 22 only `in on tailscale0` (converged) | pf anchor: port 22 only from the mesh range (by hand) | Firewall rule: `Tailscale` adapter AND `100.64.0.0/10` (by hand) |
| sshd | drop-in: key only, `AllowUsers manu@100.64.0.0/10` (converge) | same drop-in (converge) | `sshd_config`: key only, `AllowUsers` mesh range (by hand) |
| Client | `ForwardAgent no`, keepalives, `IdentitiesOnly yes`, host key pinned with `StrictHostKeyChecking yes` (`ssh/config`, `ssh/known_hosts`) | same | same |

The two server layers are independent. If the firewall is down, sshd still
refuses every connection from outside the mesh, whatever key it carries. If
sshd is misconfigured, the firewall still drops the connection.

The workstation aliases have no `*-lan` and no `*-ext` variant, by decision
(2026-10-12). Outside the mesh, SSH to a workstation fails. That is the
intended behaviour, not an outage.

**Known gaps, each tracked:**

- The Headscale ACL lets every node of the `kubelab`, `manu` and `work` users reach
  every port, including the internet-facing `vps`. Restricting `:22` on the
  workstations to the admin devices is
  [kubelab#586](https://github.com/mlorentedev/kubelab/issues/586) (VPN-ACL-004).
- `100.64.0.0/10` is also carrier-grade NAT space. Linux and Windows match
  the mesh interface as well. macOS and sshd match only the address.
- Node key expiry and the break-glass path:
  [kubelab#2221](https://github.com/mlorentedev/kubelab/issues/2221) (VPN-016).
- No reachability probe, and no negative probe from the LAN:
  [kubelab#2222](https://github.com/mlorentedev/kubelab/issues/2222) (OBS-036).
- One SSH key for every host:
  [#2357](https://github.com/mlorentedev/dotfiles/issues/2357) (SEC-012).
- Host keys are pinned one line per host, by hand. The SSH CA replaces the
  list with one signature:
  [#2373](https://github.com/mlorentedev/dotfiles/issues/2373) (SEC-013).
- SSH open to the internet on the VPS and gcp1:
  [kubelab#2224](https://github.com/mlorentedev/kubelab/issues/2224) (SEC-029).
  Ansible accepts host keys on first use:
  [kubelab#2223](https://github.com/mlorentedev/kubelab/issues/2223) (ANSIBLE-066).

## Preflight

Do these before you touch a machine:

1. **Keep a second way in.** Use a physical console, KVM or RDP. A wrong sshd
   drop-in or firewall rule over SSH locks you out, and the mesh is the only
   remote path by design.
2. **Back up what you will change**, with a dated copy next to the original,
   for example `sudo cp -p /etc/ssh/sshd_config /etc/ssh/sshd_config.bak-$(date +%F)`.
   Rollback restores these copies. Never rely on a package's defaults.
3. **Read the current state** with the commands in each section's *Check* block.
   Change only what differs.

## 1. Tailscale and mesh join

The node must run Tailscale as an **unattended system service**. If the
service only starts at user login, the machine leaves the mesh on every reboot
and stays off it, because there is no other remote path.

**Linux:**

```bash
curl -fsSL https://tailscale.com/install.sh | sh     # installs and enables tailscaled.service
sudo tailscale up --login-server https://vpn.kubelab.live
```

**macOS:** only the open-source `tailscaled` variant runs before login. The App
Store and Standalone apps (`io.tailscale.ipn.macsys`, the one installed on
`macmini` today) cannot. With FileVault on, a reboot also stops at the unlock
screen until someone types the password. Pick one for each Mac and record it:

- **Unattended (target):** switch variants in the order Tailscale requires.
  Quit the app, delete it, empty the Trash and **reboot**, so its system
  extension is unloaded. Only then install `tailscaled` as a LaunchDaemon:
  `brew install tailscale && sudo tailscaled install-system-daemon`, then
  `sudo tailscale up --login-server https://vpn.kubelab.live`. Check with
  `systemextensionsctl list | grep -i tailscale`, which must list no extension
  `[activated enabled]`. FileVault
  still blocks the boot until it is unlocked. Use `sudo fdesetup authrestart`
  for planned restarts. An unplanned restart, such as a power cut, still needs
  a person at the unlock screen.
- **Attended:** keep the app. The Mac is reachable only while a user is logged in.

The `workstation-target` step does not switch variants; it reports which one
runs. Its plan, and `dotf doctor` as a WARN, say so while the mesh would not
come back after an unattended reboot: on macOS while `launchctl print
system/com.tailscale.tailscaled` finds no daemon, on Linux while `systemctl
is-enabled tailscaled` is not `enabled`. On `macmini` the WARN stays by
decision (attended, #2356 2026-10-13).

**Windows** (elevated PowerShell, after installing Tailscale):

```powershell
tailscale up --login-server https://vpn.kubelab.live --unattended
```

`--unattended` keeps the node connected with no user logged in.

**Registration:** `tailscale up` prints a login URL. Open it. It shows the
`headscale nodes register` command to run on the Headscale host (kubelab
`docs/runbooks/headscale-setup.md`). Prefer this interactive path to a
pre-auth key: it leaves no reusable secret in shell history or the process list.

**Check:**

```bash
tailscale status --self --peers=false     # shows the node's 100.64.x.y address and name
tailscale ip -4
```

The address in `tailscale ip -4` must match the workstation's `Hostname` in
`ssh/config` through MagicDNS: `dscacheutil -q host -a name macmini.kubelab.internal`
on macOS, `getent hosts msi.kubelab.internal` on Linux.

## 2. SSH server hardening

sshd takes the **first** value it reads for each keyword. So the hardening
drop-in must sort before every other drop-in: Ubuntu's `50-cloud-init.conf` can
set `PasswordAuthentication yes`, and macOS ships `100-macos.conf`. Name it
`01-workstation.conf`.

The drop-in is `ssh/sshd-workstation.conf` in the checkout, identical on Linux
and macOS. It sets key-only authentication, no root login, no agent or X11
forwarding, client keepalives, and `AllowUsers {user}@{mesh}`. converge renders
`{user}` as the account it runs as and `{mesh}` as `100.64.0.0/10`, and
installs the rendered file.

`AllowUsers manu@100.64.0.0/10` is the sshd half of the mesh-only rule. A
connection from any other source address is refused after the key exchange,
even with the right key.

**Linux and macOS:** one converge step does this section, idempotently: the
authorized key, the host keys, the drop-in (validated with `sshd -t` before
sshd listens) and the sshd service. It then proves the result from the node
itself: sshd must accept the owner's key from the mesh address and refuse it
from the LAN address, or the run's changes are undone and the run fails
(ADR-045 amendment 2026-10-12).

```bash
dotf env role add ssh-target                       # once per machine: the opt-in
dotf converge --plan --only workstation-target     # what would change; never asks for root
dotf converge --only workstation-target            # asks for the sudo password once, on the terminal
dotf converge --only workstation-target            # second run: 0 changes, and the proof again
dotf doctor                                        # section "Workstation SSH target": PASS
```

The step refuses a node with no `100.64.0.0/10` address (join the mesh first)
and an `sshd_config` that sets one of the drop-in's keywords before its
`Include`. Without a terminal (a scheduled `dotf update`), the step is skipped
and its reason is the plan.

- **Linux:** it finds the `ssh` unit (Debian, Ubuntu) or `sshd` (Fedora, Arch),
  enables the socket unit when the distribution ships one and the service
  otherwise, and reloads sshd only when it was already running. A machine
  without `openssh-server` fails the step: install the package first.
- **macOS:** Remote Login is socket-activated: launchd starts a fresh `sshd`
  for each connection, so no reload is needed, and `ListenAddress` has no
  effect because launchd owns the socket.

**Windows:** `C:\ProgramData\ssh\sshd_config` has no `Include` by default, so
the same keywords go into the file itself, above its `Match Group administrators`
block. `scripts/windows-openssh-authorize.ps1` installs the key and the
firewall rule (section 3). It does not edit `sshd_config`.

```powershell
sshd -t -f C:\ProgramData\ssh\sshd_config
Restart-Service sshd
```

**`authorized_keys`:** `~/.ssh/authorized_keys` holds the ED25519 public key
from `ssh/id_ed25519.pub`, mode `0600`, in a `0700` `~/.ssh`. On Windows,
administrator accounts read `C:\ProgramData\ssh\administrators_authorized_keys`,
which the script above reconciles together with its ACL.

**Check** (keep the current session open while you run it). converge's own
proof logs in from the node itself; these lines check the same from outside:

```bash
sudo sshd -T | grep -E '^(authenticationmethods|passwordauthentication|kbdinteractiveauthentication|permitrootlogin|allowusers|allowagentforwarding|clientaliveinterval) '
# from another mesh node, in a NEW terminal:
ssh -o PreferredAuthentications=password -o PubkeyAuthentication=no macmini true   # must fail: Permission denied (publickey)
ssh -o BatchMode=yes macmini true                                                    # must succeed
```

Close the old session only after the new key-only login succeeds.

## 3. Firewall

**Linux** (UFW): converged by the same step. There is no LAN rule: the
workstation is mesh-only.

```bash
dotf converge --only workstation-target --plan   # lists `firewall` when it differs
dotf converge --only workstation-target          # asks for root once
```

The step adds `allow in on tailscale0 to any port 22 proto tcp` (comment
`ssh: mesh only`), then sets `default deny incoming`, and only then enables ufw,
so a run over the mesh never cuts its own session. A failure undoes the run in
reverse: the policy and the switch return to what they were, and the rule is
deleted only if the run added it.

It **refuses** when a rule already opens port 22 anywhere but `tailscale0`
(`ufw allow 22`, `allow OpenSSH`, a LAN subnet, a port range or list that
covers 22, or the mesh range without the interface), names each rule, and
changes nothing. Those rules are yours: remove or narrow them
(`sudo ufw status numbered`, `sudo ufw delete N`) and run it again.

The rules are readable by root only, so a plan without root reads ufw's
switch, the input policy and a witness of the rule files the last apply read
(`/var/lib/dotf/workstation-firewall`). A rule written later makes the plan
pending, and the next converge reads the rules as root before accepting them.
A Linux without ufw plans with a note saying so: there, sshd's `AllowUsers` is
the only fence.

Check: `sudo ufw status verbose` reads `Status: active`, `Default: deny (incoming)`,
and lists port 22 only `on tailscale0`.

**macOS** (pf, as a sub-anchor of the `com.apple/*` anchor that `/etc/pf.conf`
already declares, so the system file stays untouched):

```text
# /etc/pf.anchors/kubelab.ssh
pass in quick inet proto tcp from 100.64.0.0/10 to any port 22 flags S/SA keep state
block drop in quick proto tcp from any to any port 22
```

```bash
sudo pfctl -n -f /etc/pf.anchors/kubelab.ssh                   # parse only: the dry run
sudo pfctl -a com.apple/250.KubelabSSH -f /etc/pf.anchors/kubelab.ssh
sudo pfctl -E
sudo pfctl -a com.apple/250.KubelabSSH -s rules                # check: both rules loaded
```

A runtime-loaded anchor does not survive a reboot. Until SEC-011 installs the
LaunchDaemon that reloads it, re-run the two load commands after each restart.
The sshd `AllowUsers` rule still holds while the anchor is absent.

**Windows:** `scripts/windows-openssh-authorize.ps1` creates or narrows the
`OpenSSH-Server-In-TCP` rule. It matches the `Tailscale` adapter and
`100.64.0.0/10`. If the adapter does not exist, the script stops before it
installs or starts anything.

```powershell
Get-NetFirewallRule -Name OpenSSH-Server-In-TCP | Get-NetFirewallAddressFilter     # RemoteAddress 100.64.0.0/10
Get-NetFirewallRule -Name OpenSSH-Server-In-TCP | Get-NetFirewallInterfaceFilter   # InterfaceAlias Tailscale
Get-NetFirewallRule -Direction Inbound -Enabled True | Get-NetFirewallPortFilter | Where-Object LocalPort -eq 22
```

The last command must list only the one rule. Another enabled rule for port 22
reopens it.

**Negative check (every OS):** from a machine on the LAN that is **not** on the
mesh, the port must be closed:

```bash
nc -vz -w 3 <workstation LAN address> 22    # must fail: timed out or refused
```

## 4. Power management

A remote target must not sleep while it is on AC power.

**Linux:** the same converge step writes
`/etc/systemd/logind.conf.d/50-workstation.conf`:

```ini
[Login]
HandleLidSwitchExternalPower=ignore
HandleLidSwitchDocked=ignore
```

It reloads logind with `SIGHUP` only where `systemctl show -p CanReload
systemd-logind` says `yes`, and never restarts it, which would end the
sessions. Where logind cannot reload, the policy applies from the next boot.
Battery keeps the distribution's lid behaviour, and no sleep target is masked,
so a laptop that travels still suspends. Check, after the next boot where
logind could not reload (until then it still reports the old value):
`busctl get-property org.freedesktop.login1 /org/freedesktop/login1 org.freedesktop.login1.Manager HandleLidSwitchExternalPower`
prints `s "ignore"`.

A GNOME desktop has its own idle suspend, which logind does not govern:
`gsettings get org.gnome.settings-daemon.plugins.power sleep-inactive-ac-type`
must print `'nothing'` (it does on msi). Converging it is
[#2374](https://github.com/mlorentedev/dotfiles/issues/2374).

**macOS:** the same converge step sets the AC profile `sleep 0`, `disksleep 0`,
`womp 1`, `autorestart 1` and `powernap 1` with one `pmset -c`, and only the keys
that differ. `displaysleep` and every battery setting are left alone. A key the
hardware does not report on AC (`womp` on a Mac without Ethernet) is skipped.
The plan reads `pmset -g custom` without root and lists `power policy` while any
key differs; the proof reads it back after the apply. Check:

```bash
pmset -g custom          # AC Power: sleep 0, disksleep 0, womp 1, autorestart 1, powernap 1
fdesetup status          # FileVault on: see section 1
```

`autorestart 1` powers the Mac back on after a power cut. With FileVault on, it
then waits at the unlock screen.

**Windows** (elevated):

```powershell
powercfg /change standby-timeout-ac 0
powercfg /change hibernate-timeout-ac 0
powercfg /setacvalueindex SCHEME_CURRENT SUB_BUTTONS LIDACTION 0      # lid: do nothing
powercfg /setactive SCHEME_CURRENT
powercfg /q SCHEME_CURRENT SUB_BUTTONS LIDACTION                      # check: AC index 0x0
```

## 5. Mesh parity through dotf

The client side is code. Each workstation profile in `ssh/config` (inside the
`KUBELAB-SSH` block) sets the following:

- `Hostname` is the MagicDNS name, not the mesh IP.
- `User manu`, `IdentityFile ~/.ssh/id_ed25519` and `IdentitiesOnly yes`.
- `ServerAliveInterval 30` and `ServerAliveCountMax 3`.
- `ForwardAgent no`. A workstation runs its own `dotf identity restore`; it
  never borrows the client's agent.
- A last `Host *` block sets `AddKeysToAgent yes` and, for Apple's OpenSSH,
  `UseKeychain yes` behind `IgnoreUnknown UseKeychain`, so the file still
  parses on Linux and Windows. The first connection asks for the key's
  passphrase once; on macOS the keychain keeps it, and after a reboot the key
  loads on first use. `ssh-add` is never a step.
- `UserKnownHostsFile ~/.ssh/known_hosts_pinned` and `StrictHostKeyChecking
  yes`. The client accepts only the key pinned in `ssh/known_hosts`, deployed
  by the `ssh-known-hosts` entry, and refuses any other, including on the
  first connection.

### Pinning a new host

A key is never accepted on first use. To add a host:

1. Read the fingerprint the host presents over the mesh, without trusting it:
   `ssh-keyscan -t ed25519 <name>.kubelab.internal | ssh-keygen -lf -`.
2. Compare it against a record that does not come over that network: on the
   host's own console, `ssh-keygen -lf /etc/ssh/ssh_host_ed25519_key.pub`.
3. Only if they match, add the line to `ssh/known_hosts` with every name and
   address of the host, and the fingerprint on a `# SHA256:` comment above it.
   Add `UserKnownHostsFile` and `StrictHostKeyChecking yes` to its `Host`
   block in `ssh/config`.
4. `bats tests/ssh-known-hosts.bats` fails on a line that does not parse, on
   a key that differs from the fingerprint recorded above it, and on a host
   that pins without a line.

Converge it and check it:

```bash
dotf deploy --dry-run ssh-config      # what would change, touching nothing
dotf deploy ssh-config                # then ssh-known-hosts and ssh-pubkey, or run the full converge
dotf doctor                           # deploy drift, and mode 0600 on ~/.ssh/config
ssh -G macmini | grep -E '^(hostname|identitiesonly|forwardagent|serveraliveinterval) '   # dry run, no connection
ssh -o BatchMode=yes macmini true
```

`dotf doctor` reports a drifted `~/.ssh/config`, or one whose mode is not the
`0600` that `ai/deploy.json` declares, and names the `dotf deploy` command
that fixes it.

## Rollback

A converge run whose apply or proof fails undoes its own changes, in reverse
order, and says so. The host keys it generated stay. To retire a machine as a
target, `dotf env role rm ssh-target`, then remove what the step wrote (there
is no converge undo for a role removal yet). For the hand steps, restore the
dated backups taken in the preflight, then:

- **Linux:** if the machine served no ssh before it became a target, stop the listener first, so sshd never runs on the stock config (which allows passwords) once the drop-in is gone. This mirrors the step's own rollback (`sshtarget.Systemd.Disable`): `sudo systemctl disable --now ssh.socket && sudo systemctl stop ssh.service` where sshd is socket-activated (Ubuntu 22.10 and later), else `sudo systemctl disable --now ssh` (`sshd` on Fedora and Arch). Then `sudo rm /etc/ssh/sshd_config.d/01-workstation.conf /etc/systemd/logind.conf.d/50-workstation.conf`; if sshd stays enabled, `sudo sshd -t && sudo systemctl reload ssh`; `sudo ufw delete allow in on tailscale0 to any port 22 proto tcp` (and `sudo ufw disable` if ufw was off before the target); and `sudo rm /var/lib/dotf/workstation-firewall`.
- **Linux, a node set up by hand before the converge step:** the earlier version of this runbook masked the sleep targets, which the step neither writes nor removes. `systemctl is-enabled suspend.target` prints `masked` on such a node, and it then never suspends, on battery included. `sudo systemctl unmask sleep.target suspend.target hibernate.target hybrid-sleep.target` restores the distribution's behaviour; the logind drop-in keeps the lid policy.
- **macOS:** if the Mac stops being a target, stop Remote Login first, for the same reason: `sudo launchctl bootout system/com.openssh.sshd && sudo launchctl disable system/com.openssh.sshd`. `bootout` alone only stops the running job, so Remote Login returns at the next boot; `disable` persists the off state, as the step's own rollback does (`sshtarget.Launchd.Disable`). Then `sudo rm /etc/ssh/sshd_config.d/01-workstation.conf` and `sudo pfctl -a com.apple/250.KubelabSSH -F all`. The power profile is not security-relevant and may stay; to return to the defaults, `sudo pmset -c disksleep 10 autorestart 0` (the values the macmini had before the step).
- **Windows:** restore `sshd_config`, `sshd -t`, `Restart-Service sshd`. To retire the target, run `Stop-Service sshd; Set-Service sshd -StartupType Disabled` (a stop alone returns at the next boot) and `Disable-NetFirewallRule -Name OpenSSH-Server-In-TCP`.
