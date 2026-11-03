---
id: "lesson-546-sshd-t-prints-no-match-block"
type: lesson
status: active
title: "Plain sshd -T prints no Match block, so it cannot prove a fence that lives in one"
created: "2026-11-02"
---

# Plain sshd -T prints no Match block, so it cannot prove a fence that lives in one

## Context

The workstation drop-in limited sshd to the mesh with `AllowUsers user@100.64.0.0/10`. Win32-OpenSSH rewrites the `/`
of an AllowUsers value to `\`, the CIDR stops parsing, and every login is denied (#3080). The fence moved to
`Match Address *,!100.64.0.0/10` with `DenyUsers *`.

## Problem

The apply proves the drop-in by comparing it with `sshd -T`. With the range in a Match block that dump no longer
carries the fence: it reads the same whether the block denies the LAN, the mesh, or nothing. A proof that kept the
old shape would pass on a machine open to the LAN.

## Solution

Ask sshd about a connection: `sshd -T -C user=u,host=h,addr=a`, once from outside the mesh, which must print
`denyusers *`, and once from inside, which must not. OpenSSH 8.9p1 refuses a spec missing any of the three keywords;
10.3p1 accepts `addr=` alone, so a Mac alone would not show that. A Match in an Included file ends with that file,
measured on the three builds, but it holds every line after it within the file, so it goes last.

Any directive that moves into a Match block leaves `sshd -T`. A test that greps the dump for it then proves nothing.
