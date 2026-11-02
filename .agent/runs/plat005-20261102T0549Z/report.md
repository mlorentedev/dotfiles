# Run plat005-20261102T0549Z

- Envelope: `chore/autopilot-plat005-20261102` at `150976ed`, armed 05:49Z, expires 17:49Z.
- Ended 11:40Z. `reason: goal-blocked`: the single queue item, #2842, has two owner decisions open and no other work was in scope.
- Window covered: `caffeinate` held the host for the whole run. About 1h20 was lost to a usage limit.

## Landed

| PR | Change |
|---|---|
| #3064 | Windows `up`, `down`, `status` |
| #3073 | Windows `exec` |
| #3078 | Windows `cp`, streamed; lesson-545 (PowerShell 5.1 BOM) |
| #3079 | `exec --elevated`, Windows and Linux |

## Stopped, waiting on the owner

1. **Slice 1d, `run` recomposed** (#2842 comment 6572738715). AC6 cannot be met as written. Recommendation A: compose `run` below the verbs.
2. **PR 3, Windows fixture and acceptance** (#2842 comment 6575787768), on **#3080**: an `AllowUsers` CIDR denies every login on Windows, because Win32-OpenSSH rewrites `/` to `\`. Recommendation A: `AllowUsers {user}` plus `Match Address *,!{mesh}` with `DenyUsers *`. Do not loosen the comparison. `ssh/sshd-workstation.conf` is outside this envelope.

The measured Windows `sshd -T` dumps (OpenSSH 9.5p2) are at the end of this report, with line endings normalized to LF, so PR 3 need not re-measure them unless the drop-in changes. The fixture needs them as measured: CRLF.

## Evidence

- Windows guest: `exec`, `cp` and timeout re-proved. `--elevated`: High Mandatory Level with Administrators enabled, plain: Medium with deny-only; `net session` exit 0 against denied. OpenSSH capability installed inside a session in 10.6 min.
- Lima guest, latest code: `--elevated` `id -u` 0, plain 501; `--elevated --timeout 2s` on a TERM-ignoring program ended `timeout` after 7 s, no `sleep` left.
- #2763 was already closed: evidence recorded, no closing keyword used.

## Disclosures

- Review: CodeRabbit hit its free limit on part of the PRs. Every triage was self-triage, stated as such on the PR.
- #3079's CodeRabbit finding (unprivileged `timeout` cannot kill a root command) was real and fixed before landing. **The fix `5bf96dd2` itself landed with no reviewer output**: CodeRabbit paused on the new commit (last reviewed `43d41b47`) and pr-agent's only comment predates it. Its evidence is the Lima proof and the unit test. Disclosed on the PR.
- pr-agent reposted identical comments on new heads; disposition notes this.
- #3073 reached 304 gross lines (265 net executable) against a cap of 300; disclosed on the PR.
- Baseline failure: `TestEmbeddedTemplatesMatchVault` (#3050), pre-existing and not touched.
- Parks were left off the board while PRs landed against #2842, so it read `Backlog` during the run. It is `Blocked` now.
- Both guests are down.

## Owner items

- #3080 A or B; #2842 slice 1d A or B.
- GitGuardian incident 39344577.
- #2859 permission rules.
- Permission mode for this run was never confirmed.
- Retire the `feat-vm-linux-exec` worktree and this branch after reading this report.

## `sshd -T` effective

```text
port 22
addressfamily any
listenaddress 0.0.0.0:22
listenaddress [::]:22
logingracetime 120
x11displayoffset 10
maxauthtries 6
maxsessions 10
clientaliveinterval 60
clientalivecountmax 3
requiredrsasize 1024
streamlocalbindmask 0177
unusedconnectiontimeout none
permitrootlogin no
ignorerhosts yes
ignoreuserknownhosts no
hostbasedauthentication no
hostbasedusesnamefrompacketonly no
pubkeyauthentication yes
gssapiauthentication no
gssapicleanupcredentials yes
passwordauthentication no
kbdinteractiveauthentication no
printmotd yes
printlastlog yes
x11forwarding no
x11uselocalhost yes
permittty yes
permituserrc yes
strictmodes yes
tcpkeepalive yes
permitemptypasswords no
compression yes
gatewayports no
usedns no
allowtcpforwarding yes
allowagentforwarding no
disableforwarding no
allowstreamlocalforwarding yes
streamlocalbindunlink no
fingerprinthash SHA256
exposeauthinfo no
pidfile __PROGRAMDATA__\ssh/sshd.pid
modulifile __PROGRAMDATA__\ssh/moduli
xauthlocation /usr/X11R6/bin/xauth
ciphers chacha20-poly1305@openssh.com,aes128-ctr,aes192-ctr,aes256-ctr,aes128-gcm@openssh.com,aes256-gcm@openssh.com
macs umac-64-etm@openssh.com,umac-128-etm@openssh.com,hmac-sha2-256-etm@openssh.com,hmac-sha2-512-etm@openssh.com,umac-64@openssh.com,umac-128@openssh.com,hmac-sha2-256,hmac-sha2-512
banner none
forcecommand none
chrootdirectory none
trustedusercakeys none
revokedkeys none
securitykeyprovider internal
authorizedprincipalsfile none
versionaddendum none
authorizedkeyscommand none
authorizedkeyscommanduser none
authorizedprincipalscommand none
authorizedprincipalscommanduser none
hostkeyagent none
kexalgorithms curve25519-sha256,curve25519-sha256@libssh.org,ecdh-sha2-nistp256,ecdh-sha2-nistp384,ecdh-sha2-nistp521,diffie-hellman-group-exchange-sha256,diffie-hellman-group16-sha512,diffie-hellman-group18-sha512,diffie-hellman-group14-sha256
casignaturealgorithms ssh-ed25519,ecdsa-sha2-nistp256,ecdsa-sha2-nistp384,ecdsa-sha2-nistp521,sk-ssh-ed25519@openssh.com,sk-ecdsa-sha2-nistp256@openssh.com,rsa-sha2-512,rsa-sha2-256
hostbasedacceptedalgorithms ssh-ed25519-cert-v01@openssh.com,ecdsa-sha2-nistp256-cert-v01@openssh.com,ecdsa-sha2-nistp384-cert-v01@openssh.com,ecdsa-sha2-nistp521-cert-v01@openssh.com,sk-ssh-ed25519-cert-v01@openssh.com,sk-ecdsa-sha2-nistp256-cert-v01@openssh.com,rsa-sha2-512-cert-v01@openssh.com,rsa-sha2-256-cert-v01@openssh.com,ssh-ed25519,ecdsa-sha2-nistp256,ecdsa-sha2-nistp384,ecdsa-sha2-nistp521,sk-ssh-ed25519@openssh.com,sk-ecdsa-sha2-nistp256@openssh.com,rsa-sha2-512,rsa-sha2-256
hostkeyalgorithms ssh-ed25519-cert-v01@openssh.com,ecdsa-sha2-nistp256-cert-v01@openssh.com,ecdsa-sha2-nistp384-cert-v01@openssh.com,ecdsa-sha2-nistp521-cert-v01@openssh.com,sk-ssh-ed25519-cert-v01@openssh.com,sk-ecdsa-sha2-nistp256-cert-v01@openssh.com,rsa-sha2-512-cert-v01@openssh.com,rsa-sha2-256-cert-v01@openssh.com,ssh-ed25519,ecdsa-sha2-nistp256,ecdsa-sha2-nistp384,ecdsa-sha2-nistp521,sk-ssh-ed25519@openssh.com,sk-ecdsa-sha2-nistp256@openssh.com,rsa-sha2-512,rsa-sha2-256
pubkeyacceptedalgorithms ssh-ed25519-cert-v01@openssh.com,ecdsa-sha2-nistp256-cert-v01@openssh.com,ecdsa-sha2-nistp384-cert-v01@openssh.com,ecdsa-sha2-nistp521-cert-v01@openssh.com,sk-ssh-ed25519-cert-v01@openssh.com,sk-ecdsa-sha2-nistp256-cert-v01@openssh.com,rsa-sha2-512-cert-v01@openssh.com,rsa-sha2-256-cert-v01@openssh.com,ssh-ed25519,ecdsa-sha2-nistp256,ecdsa-sha2-nistp384,ecdsa-sha2-nistp521,sk-ssh-ed25519@openssh.com,sk-ecdsa-sha2-nistp256@openssh.com,rsa-sha2-512,rsa-sha2-256
loglevel INFO
syslogfacility AUTH
authorizedkeysfile .ssh/authorized_keys .ssh/authorized_keys2
hostkey C:/Users/Public/dotf/sshdump/host_ed25519
allowusers dotf@100.64.0.0\10
authenticationmethods publickey
channeltimeout none
maxstartups 10:30:100
persourcemaxstartups none
persourcenetblocksize 32:128
permittunnel no
ipqos af21 cs1
rekeylimit 0 0
permitopen any
permitlisten any
permituserenvironment no
pubkeyauthoptions none
```

## `sshd -T` ignored

```text
port 22
addressfamily any
listenaddress 0.0.0.0:22
listenaddress [::]:22
logingracetime 120
x11displayoffset 10
maxauthtries 6
maxsessions 10
clientaliveinterval 0
clientalivecountmax 3
requiredrsasize 1024
streamlocalbindmask 0177
unusedconnectiontimeout none
permitrootlogin without-password
ignorerhosts yes
ignoreuserknownhosts no
hostbasedauthentication no
hostbasedusesnamefrompacketonly no
pubkeyauthentication yes
gssapiauthentication no
gssapicleanupcredentials yes
passwordauthentication yes
kbdinteractiveauthentication yes
printmotd yes
printlastlog yes
x11forwarding no
x11uselocalhost yes
permittty yes
permituserrc yes
strictmodes yes
tcpkeepalive yes
permitemptypasswords no
compression yes
gatewayports no
usedns no
allowtcpforwarding yes
allowagentforwarding yes
disableforwarding no
allowstreamlocalforwarding yes
streamlocalbindunlink no
fingerprinthash SHA256
exposeauthinfo no
pidfile __PROGRAMDATA__\ssh/sshd.pid
modulifile __PROGRAMDATA__\ssh/moduli
xauthlocation /usr/X11R6/bin/xauth
ciphers chacha20-poly1305@openssh.com,aes128-ctr,aes192-ctr,aes256-ctr,aes128-gcm@openssh.com,aes256-gcm@openssh.com
macs umac-64-etm@openssh.com,umac-128-etm@openssh.com,hmac-sha2-256-etm@openssh.com,hmac-sha2-512-etm@openssh.com,umac-64@openssh.com,umac-128@openssh.com,hmac-sha2-256,hmac-sha2-512
banner none
forcecommand none
chrootdirectory none
trustedusercakeys none
revokedkeys none
securitykeyprovider internal
authorizedprincipalsfile none
versionaddendum none
authorizedkeyscommand none
authorizedkeyscommanduser none
authorizedprincipalscommand none
authorizedprincipalscommanduser none
hostkeyagent none
kexalgorithms curve25519-sha256,curve25519-sha256@libssh.org,ecdh-sha2-nistp256,ecdh-sha2-nistp384,ecdh-sha2-nistp521,diffie-hellman-group-exchange-sha256,diffie-hellman-group16-sha512,diffie-hellman-group18-sha512,diffie-hellman-group14-sha256
casignaturealgorithms ssh-ed25519,ecdsa-sha2-nistp256,ecdsa-sha2-nistp384,ecdsa-sha2-nistp521,sk-ssh-ed25519@openssh.com,sk-ecdsa-sha2-nistp256@openssh.com,rsa-sha2-512,rsa-sha2-256
hostbasedacceptedalgorithms ssh-ed25519-cert-v01@openssh.com,ecdsa-sha2-nistp256-cert-v01@openssh.com,ecdsa-sha2-nistp384-cert-v01@openssh.com,ecdsa-sha2-nistp521-cert-v01@openssh.com,sk-ssh-ed25519-cert-v01@openssh.com,sk-ecdsa-sha2-nistp256-cert-v01@openssh.com,rsa-sha2-512-cert-v01@openssh.com,rsa-sha2-256-cert-v01@openssh.com,ssh-ed25519,ecdsa-sha2-nistp256,ecdsa-sha2-nistp384,ecdsa-sha2-nistp521,sk-ssh-ed25519@openssh.com,sk-ecdsa-sha2-nistp256@openssh.com,rsa-sha2-512,rsa-sha2-256
hostkeyalgorithms ssh-ed25519-cert-v01@openssh.com,ecdsa-sha2-nistp256-cert-v01@openssh.com,ecdsa-sha2-nistp384-cert-v01@openssh.com,ecdsa-sha2-nistp521-cert-v01@openssh.com,sk-ssh-ed25519-cert-v01@openssh.com,sk-ecdsa-sha2-nistp256-cert-v01@openssh.com,rsa-sha2-512-cert-v01@openssh.com,rsa-sha2-256-cert-v01@openssh.com,ssh-ed25519,ecdsa-sha2-nistp256,ecdsa-sha2-nistp384,ecdsa-sha2-nistp521,sk-ssh-ed25519@openssh.com,sk-ecdsa-sha2-nistp256@openssh.com,rsa-sha2-512,rsa-sha2-256
pubkeyacceptedalgorithms ssh-ed25519-cert-v01@openssh.com,ecdsa-sha2-nistp256-cert-v01@openssh.com,ecdsa-sha2-nistp384-cert-v01@openssh.com,ecdsa-sha2-nistp521-cert-v01@openssh.com,sk-ssh-ed25519-cert-v01@openssh.com,sk-ecdsa-sha2-nistp256-cert-v01@openssh.com,rsa-sha2-512-cert-v01@openssh.com,rsa-sha2-256-cert-v01@openssh.com,ssh-ed25519,ecdsa-sha2-nistp256,ecdsa-sha2-nistp384,ecdsa-sha2-nistp521,sk-ssh-ed25519@openssh.com,sk-ecdsa-sha2-nistp256@openssh.com,rsa-sha2-512,rsa-sha2-256
loglevel INFO
syslogfacility AUTH
authorizedkeysfile .ssh/authorized_keys .ssh/authorized_keys2
hostkey C:/Users/Public/dotf/sshdump/host_ed25519
authenticationmethods any
channeltimeout none
maxstartups 10:30:100
persourcemaxstartups none
persourcenetblocksize 32:128
permittunnel no
ipqos af21 cs1
rekeylimit 0 0
permitopen any
permitlisten any
permituserenvironment no
pubkeyauthoptions none
```
