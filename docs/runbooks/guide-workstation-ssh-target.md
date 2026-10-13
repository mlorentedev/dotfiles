---
id: "dotfiles-runbook-workstation-ssh-target"
type: runbook
status: active
tags: [runbook, ssh, tailscale, workstations, security, power-management]
created: "2026-10-12"
updated: "2026-10-12"
owner: manu
---

# Workstation SSH Target

This runbook turns a workstation (`macmini`, `msi`, or a Windows host) into a
remote development target that is reachable over SSH **through the Headscale
mesh and nowhere else**. The client side (`ssh/config`) is already converged by
`dotf`. The server side in sections 2 to 4 is applied by hand **for now**.

> **Interim procedure.** Sections 2 to 4 are replaced by one
> `dotf converge --only workstation-target` per machine, tracked in
> [#2356](https://github.com/mlorentedev/dotfiles/issues/2356) (SEC-011, a row of
> PLAT-001b). Until then, every step below pairs its change with a check. Do not
> skip the checks.

## Security model

| Layer | Linux | macOS | Windows |
|---|---|---|---|
| Network | UFW: port 22 only `in on tailscale0` | pf anchor: port 22 only from the mesh range | Firewall rule: `Tailscale` adapter AND `100.64.0.0/10` |
| sshd | drop-in: key only, `AllowUsers manu@100.64.0.0/10` | same drop-in | `sshd_config`: key only, `AllowUsers` mesh range |
| Client | `ForwardAgent no`, keepalives, `IdentitiesOnly yes` (`ssh/config`) | same | same |

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

The drop-in, identical on Linux and macOS:

```text
# Managed by docs/runbooks/guide-workstation-ssh-target.md (SEC-011 automates it)
AuthenticationMethods publickey
PubkeyAuthentication yes
PasswordAuthentication no
KbdInteractiveAuthentication no
PermitRootLogin no
AllowUsers manu@100.64.0.0/10
AllowAgentForwarding no
X11Forwarding no
ClientAliveInterval 60
ClientAliveCountMax 3
```

`AllowUsers manu@100.64.0.0/10` is the sshd half of the mesh-only rule. A
connection from any other source address is refused after the key exchange,
even with the right key.

**Linux:**

```bash
SSHD_UNIT=ssh        # Debian and Ubuntu; Fedora and Arch name the unit sshd
sudo install -m 0644 01-workstation.conf /etc/ssh/sshd_config.d/01-workstation.conf
grep -n '^Include' /etc/ssh/sshd_config   # the Include must come before any other directive
sudo sshd -t && sudo systemctl reload "$SSHD_UNIT"
```

**macOS:** Remote Login is socket-activated. launchd starts a fresh `sshd` for
each connection, so the next connection reads the new file with no reload.
That is also why `ListenAddress` has no effect here: launchd owns the socket.

```bash
sudo install -m 0644 01-workstation.conf /etc/ssh/sshd_config.d/01-workstation.conf
sudo sshd -t
sudo systemsetup -setremotelogin on        # needs Full Disk Access for the terminal
```

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

**Check** (keep the current session open while you run it):

```bash
sudo sshd -T | grep -E '^(authenticationmethods|passwordauthentication|kbdinteractiveauthentication|permitrootlogin|allowusers|allowagentforwarding|clientaliveinterval) '
# from another mesh node, in a NEW terminal:
ssh -o PreferredAuthentications=password -o PubkeyAuthentication=no macmini true   # must fail: Permission denied (publickey)
ssh -o BatchMode=yes macmini true                                                    # must succeed
```

Close the old session only after the new key-only login succeeds.

## 3. Firewall

**Linux** (UFW). There is no LAN rule: the workstation is mesh-only.

```bash
sudo ufw default deny incoming
sudo ufw default allow outgoing
sudo ufw allow in on tailscale0 to any port 22 proto tcp comment 'ssh: mesh only'
sudo ufw --dry-run enable                 # read the rules it would load
sudo ufw enable
```

Check: `sudo ufw status verbose` lists exactly one rule for port 22, `on tailscale0`.

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

**Linux:**

```ini
# /etc/systemd/logind.conf.d/10-workstation.conf
[Login]
HandleLidSwitch=ignore
HandleLidSwitchExternalPower=ignore
HandleLidSwitchDocked=ignore
```

```bash
sudo systemctl kill -s HUP systemd-logind     # reload logind without ending sessions
sudo systemctl mask sleep.target suspend.target hibernate.target hybrid-sleep.target
```

Masking turns off sleep on battery as well. For a laptop that also travels,
skip the mask and rely on the lid settings.
Check: `systemd-analyze cat-config systemd/logind.conf | grep Lid` and
`systemctl is-enabled suspend.target` (prints `masked`).

**macOS:**

```bash
sudo pmset -c sleep 0 disksleep 0 womp 1 autorestart 1
pmset -g custom          # check: AC Power shows sleep 0, womp 1, autorestart 1
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

Converge it and check it:

```bash
dotf deploy --dry-run ssh-config      # what would change, touching nothing
dotf deploy ssh-config                # then the same for ssh-pubkey, or run the full converge
dotf doctor                           # deploy drift, and mode 0600 on ~/.ssh/config
ssh -G macmini | grep -E '^(hostname|identitiesonly|forwardagent|serveraliveinterval) '   # dry run, no connection
ssh -o BatchMode=yes macmini true
```

`dotf doctor` reports a drifted `~/.ssh/config`, or one whose mode is not the
`0600` that `ai/deploy.json` declares, and names the `dotf deploy` command
that fixes it.

## Rollback

Restore the dated backups taken in the preflight, then:

- **Linux:** `sudo sshd -t && sudo systemctl reload "$SSHD_UNIT"` (section 2); `sudo ufw delete allow in on tailscale0 to any port 22 proto tcp` if you remove the target role; `sudo systemctl unmask sleep.target suspend.target hibernate.target hybrid-sleep.target`.
- **macOS:** `sudo rm /etc/ssh/sshd_config.d/01-workstation.conf`; `sudo pfctl -a com.apple/250.KubelabSSH -F all`; `sudo systemsetup -setremotelogin off` if the Mac stops being a target.
- **Windows:** restore `sshd_config`, `sshd -t`, `Restart-Service sshd`. To retire the target, run `Disable-NetFirewallRule -Name OpenSSH-Server-In-TCP` and `Stop-Service sshd`.
