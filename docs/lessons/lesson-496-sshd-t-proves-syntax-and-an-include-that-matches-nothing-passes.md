---
id: "lesson-496-sshd-t-proves-syntax-and-an-include-that-matches-nothing-passes"
type: lesson
status: active
title: "sshd -t proves syntax, so an Include that matches nothing passes it"
created: "2026-10-19"
---

# sshd -t proves syntax, so an Include that matches nothing passes it

## Context

The workstation target installs an sshd drop-in (key only, `AllowUsers
<user>@100.64.0.0/10`) and validated it with `sshd -t` before it listened. On
Windows the `Include` the step accepted was one Win32-OpenSSH resolves to a
path that matches no file (#2763). `sshd -t` passed, the step reported
converged, and the effective config was sshd's defaults: password login on.

## Lesson

A validity check and an effect check are different questions. `sshd -t` asks
whether the config parses; an `Include` that matches nothing parses. Only the
dump, `sshd -T`, shows the values sshd would run with. Check the directives the
change exists to set, against the file the change wrote, not against a second
list kept beside it.

`sshd -T` needs root, so it belongs where root is already ready: the apply.
`Prove` also runs on a converged plan and in the post-condition of a headless
converge, which has no root to ask for; putting the dump there turns a quiet
re-run into `ErrNoTerminal`. The dump is also address-independent when the
drop-in has no `Match` block (measured on OpenSSH 10.3p1: the same values with
and without `-C`), so one dump covers the mesh and the LAN.

## Guard

`checkEffective` (`cli/internal/sshtarget/effective.go`) reads the dump after
the drop-in is installed and fails the apply, undoing it, when a proven
directive is not the drop-in's. The tests pin it against two dumps measured
with `sshd -T -f` on macOS, one that read the drop-in and one that did not.
The 8.9p1 (Ubuntu 22.04) and Windows 9.5p2 dumps are not fixtures yet: they are
unmeasured here, not transcribed.
