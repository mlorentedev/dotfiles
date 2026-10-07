---
id: "lesson-333-createprocess-detached-ignores-no-window"
type: lesson
status: active
title: "CreateProcess DETACHED_PROCESS ignores CREATE_NO_WINDOW"
created: "2026-10-02"
---

# CreateProcess DETACHED_PROCESS ignores CREATE_NO_WINDOW

## Context
When spawning a background daemon on Windows (like `bw serve`) via Go's `exec.Command` in a console application (like a CLI tool), it is common to use `syscall.SysProcAttr` with `CreationFlags` to detach it. 

## The Trap
We attempted to suppress the flashing console window (especially visible when the executable is a scoop shim) by bitwise-ORing `CREATE_NO_WINDOW` (`0x08000000`) with `DETACHED_PROCESS` (`0x00000008`). However, the Windows `CreateProcess` API documentation explicitly states that `CREATE_NO_WINDOW` is ignored if `DETACHED_PROCESS` is specified. Combining them is a no-op for window suppression.

## The Solution
To successfully launch a detached process while suppressing the GUI console window, do not use `CREATE_NO_WINDOW`. Instead, use the `HideWindow: true` property in Go's `SysProcAttr`, which maps to `STARTF_USESHOWWINDOW` with `SW_HIDE`. This achieves the background detachment without rendering a console shim window.
