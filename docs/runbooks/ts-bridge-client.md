---
title: "Operate the role-gated ts-bridge client"
status: active
---

# Operate the role-gated ts-bridge client

Use this runbook on a machine that cannot run native Tailscale and must reach a
private Git forge through the userspace ts-bridge SOCKS listener.

## Converge the managed surfaces

```powershell
dotf env role add ts-bridge-client
dotf tools install ts-bridge
dotf deploy gitconfig
dotf converge --only git-config,ts-bridge-client
```

The role owns:

- the pinned ts-bridge binary;
- `~/.config/git/ts-bridge.gitconfig`;
- only the `https://gitea.kubelab.live` Git proxy entry.

It does not own profiles, credentials, auth keys, SSH identities, tsnet state,
system DNS, browser proxy settings or an unattended service.

## Start the bridge

Until #3244 provides lifecycle commands, start the declared profile explicitly:

The effective health address must be `127.0.0.1:9081`, matching doctor. Check
`TS_HEALTH_ADDR` and the current directory's `.env` for overrides before
starting.

```powershell
ts-bridge connect `
  --profile <profile> `
  --config <profile-config.yml> `
  --socks5 127.0.0.1:1081 `
  --quiet
```

Keep the process attached when practical. If it must run detached, retain its
PID so it can be stopped by identity rather than by process name.

## Verify the three layers

After a lifecycle start has recorded the profile state, use:

```powershell
dotf bridge status --profile <profile>
dotf bridge status --profile <profile> --json
```

Healthy exits 0. Every other state exits non-zero; human output uses
`ERROR` / `WHY` / `FIX`, while JSON carries the same fields. Until the start
command lands in TOOL-035 PR5, use doctor for a manually-started bridge.

```powershell
dotf doctor --scope machine --verbose
```

The `ts-bridge client` section must report:

```text
[ OK ] ts-bridge version matches packages.json
[ OK ] bridge listener ready
[ OK ] Gitea destination reachable with verified TLS
```

The states are intentionally separate:

- `stopped`: no process record exists, or the recorded PID is no longer
  running, so status cannot prove a bridge is running under this profile. A
  bridge started by hand from this runbook leaves no record, so `stopped` does
  not mean nothing is listening: confirm the listener before starting a second
  bridge, which cannot bind the port the first one holds;
- `unowned`: the recorded PID is alive but fails the exact identity check, so
  status refuses to signal it;
- `launch_mismatch`: the profile was re-saved after the process started, so the
  running bridge no longer matches the declared descriptor — restart it from
  the recorded profile;
- `listener_down`: the owned process is alive but the readiness probe does not
  answer; read the attached `ts-bridge` terminal output. Read-only status
  cannot tell a slow startup from a listener that will never come up, so
  bounded logs arrive with the lifecycle `start` command;
- `destination_failed`: the listener answers but cannot carry private-forge
  traffic;
- `healthy`: listener ready and TLS verified through `socks5h`.

Verify Git routing independently:

```powershell
git config --global --includes --get-urlmatch http.proxy https://gitea.kubelab.live
git config --global --includes --get-urlmatch http.proxy https://github.com
git -C <gitea-repository> ls-remote --symref origin HEAD
```

The first command prints `socks5h://127.0.0.1:1081`. The GitHub lookup prints
nothing because GitHub remains direct.

## Recover Copilot App origin verification

Copilot App can persist a project as a GitHub repository when the private
origin is unreachable during registration. Restoring the bridge does not
invalidate that cached identity: Git `ls-remote` can pass while session creation
still says `origin could not be verified`.

1. Prove the doctor section and Git `ls-remote` pass.
2. In Copilot App, remove the affected repository registration.
3. Add the same local repository path again.
4. Create one session and confirm workspace initialization succeeds.

Do not edit Copilot App's internal database. Re-adding the project is the
supported way to rebuild its origin metadata.

## Browser access is separate

The role configures Git only. A browser does not read Git configuration, and
system DNS remains unchanged. Direct browser access to the private forge is
therefore not evidence that the Git bridge is unhealthy. Browser/PAC routing is
a separate architecture decision and must not be hidden inside this role.

## Stop the bridge

Until TOOL-035 PR5 lands an owned `dotf bridge stop`, keep the bridge attached
and stop it with Ctrl+C. Do not derive a PID from port 1081 or kill by process
name: another application may own that listener, and PID reuse makes a stale
number unsafe.

After the owned stop command lands, use it and re-run doctor. Listener-down must
be explicit; it must never look healthy.
