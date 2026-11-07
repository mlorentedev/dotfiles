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
dotf converge --only ts-bridge-client
```

The role owns:

- the pinned ts-bridge binary;
- `~/.config/git/ts-bridge.gitconfig`;
- only the `https://gitea.kubelab.live` Git proxy entry.

It does not own profiles, credentials, auth keys, SSH identities, tsnet state,
system DNS, browser proxy settings or an unattended service.

## Start the bridge

Until #3244 provides lifecycle commands, start the declared profile explicitly:

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

- listener down: the operator-controlled bridge is stopped or starting;
- listener ready, destination failed: the bridge is alive but cannot carry
  private-forge traffic;
- destination pass: TLS was verified through `socks5h`.

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

## Stop a detached bridge

On Windows, resolve the exact listener owner and stop only that PID:

```powershell
$bridgePid = (Get-NetTCPConnection -LocalPort 1081 -State Listen).OwningProcess
Stop-Process -Id $bridgePid
```

Then re-run doctor. Listener-down must be explicit; it must never look healthy.
