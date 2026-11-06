# Run report: plat001c-20261106T0654Z

Armed 2026-11-06T06:54Z, expires 14:54Z. Envelope: `.agent/unattended.yml` on this branch.

## Items

| Item | State | Record |
|---|---|---|
| #3222 user Path from `env-persist` | ready, CI settling | the PR |
| Blocker 1 reader, `winget_arch` | #3226 draft, CI queued | the PR, spec row "Blocker 1" |
| AC4 re-measure on a clean Arm64 guest | measured, AC4 **not met** | below |
| Owner item 8: MEMORY.md | six finished threads archived to `archive/superseded-handoffs-2026-11-05.md`; still 209 lines | #3227 |
| Owner item 8: tickets | #3227 (handoff-write line cap + auto-archive), #3228 (PR polling over REST) | the issues |

## AC4 re-measure (2026-11-06 07:11–08:20Z)

The guest came from clean-base with no VC++ runtime (`vcredist before: False`). It used a dev dotf (windows/arm64) built from #3174, #3222 and #3226 merged together, plus a `vcredist` entry with `winget_arch` placed ahead of `mise` in `packages.json`. That entry exists only locally and is not pushed. The run set `MISE_HTTP_DOWNLOAD_TIMEOUT=3h`; with it, claude and argocd finished on the 55 kB/s link where 30 minutes had not been enough.

```
=== tools, run 1
  [FAIL] tools 1h07m catalog installed: sops, vcredist, mise, tea, powershell, git, eza, docker, aws, hive, poetry, pre-commit; ... installed: actionlint, age, agy, argocd, bat, bats, claude, ...
copilot: npm exited 0 but copilot does not run on PATH (is npm's global bin dir on PATH?)
bw: npm exited 0 but bw does not run on PATH ...
yarn: npm exited 0 but yarn does not run on PATH ...
after mise install, not running at their pin: bats
=== env-persist, run 1   [CHANGE] set 13 variables and the tool directories   exit 0
=== tools, run 2         [FAIL] the same four npm CLIs (opencode too) and bats   exit 1
=== env-persist, run 2   [ OK ] 13 variable(s) and the tool directories in the user environment   exit 0
=== a new process with no profile, PATH from the registry
dotf C:\Users\dotf\.local\bin\dotf.exe       mise C:\Users\dotf\.local\bin\mise.exe
node/uv/gh/jq/python/claude/pi  C:\Users\dotf\AppData\Local\mise\shims\<name>.cmd
```

- **Blocker 1 closed:** the catalog itself placed the VC++ runtime, and mise then ran. No hand install was needed.
- **Blocker 2 closed:** a process with no profile resolves every tool through the user Path that #3222 writes. A second `env-persist` run reports OK.
- **AC4 not met.** The tools step still exits 1, for two reasons:
  - #2438: npm's global prefix under a mise Node is `…\mise\installs\node\24.22.1`. The `.cmd` launchers land in that directory, which is on no PATH, and `%APPDATA%\npm` does not exist. The fix the issue prefers (option 2, mise's `npm:` backend) needs `versions.conf` pins, which this envelope freezes. I commented the evidence on the issue and left it for an attended session.
  - bats not running at its pin: #2346/#3214, held by another session.
- The doctor output from that run says nothing about the persisted variables. The script imported only `Path` from the registry into the fresh process, so the "DOTFILES_DIR unset" lines come from the script itself.
- #3142 did not reproduce. `aws` installed (MSI 2.39.5), and `winget list --id Amazon.AWSCLI -e` exited 0 right after. I commented this on the issue and left it open.

## Held for the owner

- The `vcredist` catalog entry. Every released dotf rejects the new key, so it waits for a release that carries #3226. I did not open a draft for it, because the from-zero job would go red. The entry is in the spec row.
- Deleting `setup-windows.ps1`'s `dotf env persist` block, and the AC4 follow-up. Both wait for that same release.
- MEMORY.md is still over the 200-line load cap (#3227). Read this report before the handoff block.
