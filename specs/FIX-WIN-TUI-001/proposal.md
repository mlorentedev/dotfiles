---
id: FIX-WIN-TUI-001
type: proposal
status: draft
owner: "@mlorentedev"
created: "2026-10-02"
issue: "mlorentedev/dotfiles#2019"   # the fix merged as #1971 (issue #1970, closed); #2019 tracks review and archive
wip_override: "critical fix for interactive CLIs on Windows"
---

# Fix Windows TTY failure for interactive Node CLIs (pi, opencode)

## Why

As documented in \docs/lessons/lesson-270-a-security-wrapper-that-breaks-the-tool-it-protects-gets-routed-around.md\, Go's \os/exec\ doesn't natively support creating pseudo-terminals (PTY) on Windows. Because of this, \dotf secrets run\ connects to child processes via pipes.

Interactive Node TUIs like \pi\ and \opencode\ (which use \Ink\) refuse to start if they detect their stdout is not a TTY. Since the PowerShell profile was wrapping these commands with \dotf secrets run\ directly, the TUIs were exiting silently on Windows. Additionally, trying to resolve the API keys natively via \dotf secrets show NAN_API_KEY\ was failing because the \show\ command explicitly refuses to return values for registry entries that map to multiple environment variables (which \NAN_API_KEY\ does).

## What

We will bypass \dotf secrets run\'s stdout redaction mechanism via a Base64 extraction payload.

The wrappers in \powershell/profile.ps1\ for \opencode\ and \pi\ will:
1. Call \dotf secrets run --only ...\ but pass it a PowerShell one-liner that encodes the environment variables to a Base64 string.
2. Since the string is encoded, the \edactWriter\ doesn't intercept it.
3. The parent shell captures the Base64 string, decodes it into the API keys, and temporarily places them in the \$env\ variables.
4. Execute the raw binaries (\pi.cmd\, \opencode.cmd\) which now retain full interactive TTY console capabilities.
5. Use a \	ry/finally\ block to guarantee the injected secrets are scrubbed from the session environment variable block upon exit.

## Acceptance Criteria

1. \pi\ and \opencode\ can be launched in PowerShell on Windows without exiting silently.
2. The secrets are properly injected and parsed without triggering "unknown secret" or "exposes 2 vars" errors.
3. The secrets are cleaned up from the parent PowerShell environment block after the program exits.
4. The \pi-config.bats\ tests are updated to expect the new wrapper signature.
