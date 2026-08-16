---
id: "CLI-039-dotf-deploy"
type: spec
status: verifying
created: "2026-08-16"
issue: "mlorentedev/dotfiles#1023"
tags: [spec, verification, cli, deploy]
template_version: "1.0"
---

# Verification — CLI-039-dotf-deploy (slice 1)

## The point of the exercise: the twins got shorter

```
$ git diff --stat origin/main -- setup-linux.sh setup-windows.ps1
 setup-linux.sh    | 26 +++++++-------------------
 setup-windows.ps1 | 31 +++++++++----------------------
 2 files changed, 16 insertions(+), 41 deletions(-)
```

**Net -25 lines**, and the two remaining call sites are a `dotf deploy pi`
invocation each. The pi block was 21 lines of shell and 25 of PowerShell
implementing the same four steps differently; it is now one Go implementation.

## Live

```
$ dotf deploy --dry-run
in sync   pi         /home/manu/.pi/agent/models.json

$ dotf deploy nope
Error: no such config in the manifest: "nope" (declared: [pi])
```

`in sync` is the comparison working: the deployed file already matches what
render produces from the source, so nothing is rewritten.

## Acceptance criteria

| AC | Evidence |
|---|---|
| full deploy on Linux and Windows, identical behaviour | one Go implementation; both setups call it. Windows path not executed here — no Windows box in this session, stated rather than implied |
| setups net shorter | -25 lines, above |
| destinations via `dotf env` | `ExpandDst` takes the env seam; `TestExpandDst_*` |
| render is called, not reimplemented | `deployRenderer` wraps `secrets.Render`; `TestDeploy_RunsTheRendererOnTheStagedCopyOnly` |
| 0600 on a secret-bearing destination | `TestDeploy_InstallsWithDeclaredMode`, gated off Windows where POSIX bits are not meaningful |
| idempotent, no rewrite | `TestDeploy_IsIdempotentAndDoesNotRewrite`, asserted on mtime |
| `--dry-run` writes nothing | `TestDeploy_DryRunWritesNothing` |
| errors name the config | `TestDeploy_ErrorsNameTheConfig` |

## Toolchain

Go: `build`, `vet`, `test ./internal/...`, `golangci-lint run` (v2.12.2 pin) —
clean. Shell: `shellcheck setup-linux.sh` (only the pre-existing SC1091 info on
sourcing `utils.sh`), `bash -n` and `zsh -n` both clean. `setup-windows.ps1`
added lines are ASCII-only.

## Deliberately not closing #1023

This is slice 1 of 3. The PR says `Refs #1023`, not `Closes`, so the spec stays
active for opencode and MCP. Slice 2 is gated on an empirical answer to whether
opencode resolves `{env:VAR}` itself — the setup comment claims both tools do,
and pi's identical claim was disproven only by experiment (#987). Assuming it
twice is the mistake that made that issue expensive.
