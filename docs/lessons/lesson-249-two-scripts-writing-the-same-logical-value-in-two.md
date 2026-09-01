# Lesson 242 — Two scripts writing the same logical value in two syntaxes make an equality rule OS-dependent

**Date:** 2026-08-31
**Context:** HARNESS-045 AC1 — cutting `setup-linux.sh` and `setup-windows.ps1` over to
`dotf harness bind`

## What happened

`dotf harness bind` adopts a hook entry the old setup scripts wrote by **exact command
equality** (`sameCommand`), because an unmarked entry running our exact command is ours no
matter who wrote it. That rule was designed and mutation-tested against the deployed
`~/.claude/settings.json` on Linux, where it is correct.

The two setup scripts had written the same logical command in two different syntaxes:

| script | deployed command |
|---|---|
| `setup-linux.sh` | `/home/manu/.local/bin/dotf mem session-start` |
| `setup-windows.ps1` | `"C:\Users\manu\.local\bin\dotf.exe" mem session-start` |

Quoted, and with a `.exe` suffix. The Go resolver produced neither. So on Windows the
adoption rule would have matched nothing and **appended a second session-start hook** —
every session running the memory hook twice — while on Linux the identical code adopted
cleanly. One rule, one binary, opposite outcomes per OS.

Nothing in the Linux test suite could see it. The fixture hardcoded the bare form, so the
Windows leg of CI would not have caught it either: it would have asserted that the
duplicate was correct.

## The fix

Two halves, and the second is the transferable one.

1. Render the token the way each OS's script had: quoted on Windows, bare elsewhere unless
   the path contains a space.
2. **Take the OS as a parameter, not from `runtime.GOOS`.**

```go
func hookBinaryToken(path, goos string) string {
	if goos == "windows" || strings.ContainsAny(path, " \t") {
		return `"` + path + `"`
	}
	return path
}
```

`runtime.GOOS` is read exactly once, at the command's edge. Everything below it is a pure
function of an explicit `goos`, so the Windows branch is table-tested from the Linux box
that develops it. Reading the global inside would have made the Windows behaviour
observable only by travelling to the Windows box — which in this repo means deferring the
finding into a batched Windows sitting, i.e. finding it after the duplicate had shipped.

The test fixture had to move with it: it now writes the shape *this OS's* setup script
deployed, so the assertion means the same thing on both legs of CI.

## The lesson

**When a value has been written by two per-OS scripts, "the same command" is a claim about
syntax, not about intent — and any equality rule over it is OS-dependent until proven
otherwise.** Consolidating twin scripts into one binary does not merge their output
formats; it merely moves the divergence inside, where it is easier to miss because there is
now only one code path to read.

Two corollaries:

- **An OS-conditional output format is only testable where the OS is an argument.** A
  branch on `runtime.GOOS` is untestable from the other OS by construction, which converts
  a static, seconds-long check into a trip to another machine. Pass the discriminator in.
- **A fixture that hardcodes one platform's shape does not fail on the other — it asserts
  the bug.** The Windows CI leg would have gone green on a duplicated hook, because the
  fixture and the expectation were both wrong in the same direction. Derive the fixture
  from the same function the production path uses.

This is the mirror image of [lesson 219](lesson-219-a-stale-cli-refuses-with-the-same-exit-status-as-a.md):
there, two *different* questions produced the same answer (exit 1) and had to be told
apart; here, the *same* question produced two answers and had to be reconciled.

## Where it bit

`cli/internal/cmd/harness_bind.go` (`hookBinaryToken`, `resolveDotfPath`),
`cli/internal/cmd/harness_bind_test.go` (`TestHookBinaryTokenMatchesWhatEachSetupScriptDeployed`,
`bindFixture` deriving the deployed shape rather than hardcoding it).
