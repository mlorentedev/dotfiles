---
id: "lesson-412-converge-a-root-only-file-without-root-through-a-witness"
type: lesson
status: active
title: "A plan without root can still see a root-only file change, through a witness of its stat"
created: "2026-10-13"
---

# A plan without root can still see a root-only file change, through a witness of its stat

## Context
The workstation step converges ufw so port 22 opens only on the mesh (#2356, PLAT-001b
PR 7e). Its plan runs without root, from doctor and from `converge --plan`, and must
say whether the machine is converged. ufw keeps its rules in `/etc/ufw/user.rules` and
`user6.rules`, mode 0640 root, so a plan cannot read them.

## The Trap
Two answers look reasonable and both are wrong. Reading only what is world-readable
(`ENABLED=yes`, the input policy) reports converged on a machine where someone later ran
`ufw allow 22`, which is the rule the step exists to keep out. Reporting the firewall as
always pending keeps every converge asking for root and breaks `changed=0` on a re-run.

The spec carried a third trap: its row said msi "has UFW active". Read the same day,
`ufw.conf` said `ENABLED=no`. A claim about a machine's state that nobody measured is
a guess.

## The Solution
- The apply reads the rules as root, refuses anything that opens port 22 off the mesh,
  and as its last step records a witness: each rule file's size and mtime, which `stat`
  returns without root because only the files, not `/etc/ufw`, are closed.
- The plan compares the witness with a fresh `stat`. Any write to the rules moves the
  mtime, so the plan reads pending and the next apply reads the rules as root again.
  Nothing written since: converged, without root.
- The apply also asks `ufw status verbose` what ufw loaded before it records the witness,
  because the files say what ufw will load and the status says what it did.
- Before writing a machine's state into a spec, read it, with root if that is what it
  takes, and write down the date of the reading.
