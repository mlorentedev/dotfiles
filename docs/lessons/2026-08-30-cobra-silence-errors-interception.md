---
id: lesson-cobra-silence-errors
type: lesson
title: Intercepting Cobra Errors without Breaking SilenceErrors
created: 2026-08-30
tags: [go, cobra, cli, errors]
---

# Intercepting Cobra Errors without Breaking `SilenceErrors`

**Context:** We needed to intercept specific errors (`TerminalFailureError`) at the top level of the `dotf` CLI to print a clean JSON latch for AI agents, bypassing Cobra's default `"Error: ..."` prefix.

**The Problem:**
Initially, we used a custom `errInterceptor` (an `io.Writer` wrapping `stderr`) injected via `rootCmd.SetErr(&interceptor)`. However, if a subcommand (like `vault health`) sets `SilenceErrors = true` to guarantee silent JSON output on failure, `errInterceptor` still fired if the code manually printed anything, or it failed to strip the `"Error: "` prefix from wrapped errors because Cobra formats the error *before* writing it to `SetErr`.
If we forced `SilenceErrors = true` on the `rootCmd`, it cascaded and silenced *every* subcommand, forcing us to re-implement normal error printing manually everywhere.

**The Solution:**
Instead of intercepting the string stream, intercept the error object natively:

1. Send Cobra's automatic error printing to `/dev/null`: `rootCmd.SetErr(io.Discard)`.
2. Execute the command and capture both the executed command and the error natively: `executedCmd, err := rootCmd.ExecuteC()`.
3. Handle the error purely using Go types (`errors.As`), completely bypassing Cobra's string formatting.
4. For normal errors, respect the subcommand's original intent: `if !executedCmd.SilenceErrors { fmt.Fprintf(...) }`.

This guarantees that we can unwrap our custom errors cleanly while preserving the exact `SilenceErrors` behavior declared by each individual subcommand.
