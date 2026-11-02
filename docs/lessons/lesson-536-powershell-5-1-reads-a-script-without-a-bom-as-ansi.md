---
id: "lesson-536-powershell-5-1-reads-a-script-without-a-bom-as-ansi"
type: lesson
status: active
title: "PowerShell 5.1 reads a script without a BOM as ANSI"
created: "2026-11-02"
---

# PowerShell 5.1 reads a script without a BOM as ANSI

## Context

PLAT-005 added `dotf vm windows exec --script`, which writes a host file into the guest through the agent and runs
it with the PowerShell that ships in Windows, 5.1. The file is UTF-8, as every file on the host is.

## Problem

A script printing `hola ñ` printed mojibake on the real guest and still exited with the right code. Nothing failed:
no parse error, status `ok`. PowerShell 5.1 decodes a `.ps1` with no byte order mark in the system ANSI code page;
PowerShell 7 assumes UTF-8, so a test on a newer shell would not have shown it, and a fake guest cannot.

## Solution

The host prepends a UTF-8 BOM to the script it writes, and only there: the file on the host stays as it was. The
rule is general: text handed to Windows PowerShell 5.1 as a file carries a BOM, and a check of it reads the output
on a real guest with a non-ASCII character in it, because the failure has no exit code.
