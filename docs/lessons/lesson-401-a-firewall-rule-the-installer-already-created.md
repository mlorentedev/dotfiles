---
id: "lesson-401-a-firewall-rule-the-installer-already-created"
type: lesson
status: active
title: "A firewall rule the installer already created is never the one you wrote"
created: "2026-10-12"
---

# A firewall rule the installer already created is never the one you wrote

## Context
`scripts/windows-openssh-authorize.ps1` enables OpenSSH Server on a Windows host and owns its
firewall rule, `OpenSSH-Server-In-TCP`. A review against the owner's rule "no SSH from outside
the mesh" asked which addresses that rule admits.

## The Trap
The script created the rule with no `-RemoteAddress`, so it was open to any address on every
profile. Adding the restriction to `New-NetFirewallRule` would have looked like the fix and
changed nothing on a real host. The script runs `Add-WindowsCapability` first, and the
capability creates a rule with the same name, open to `Any`. The `$null -eq $rule` branch, the
only one that carried the parameters, never ran. The other branch only re-enabled whatever was
already there. The Pester test mocked `Get-NetFirewallRule` to return `$null`, so it exercised
the branch that real hosts never take.

The order hid a second problem: sshd started before the rule was settled. That left a window
where it listened behind the installer's open rule.

## The Solution
Every branch now converges the rule to the same state, `-InterfaceAlias Tailscale` and
`-RemoteAddress 100.64.0.0/10`. A new rule is created that way, and an existing one is narrowed
with `Set-NetFirewallRule`. The rule is settled before `Start-Service sshd`. A host without the
`Tailscale` adapter stops before anything is installed. Pester covers the pre-existing rule,
the missing adapter, and the source order of firewall and service.

## Takeaways
- When an installer creates a resource with the name you manage, the "create if absent" branch is
  the one that never runs. Converge the existing object to the declared state. Do not only create
  a correct one.
- A mock that returns "absent" tests the first-install path. Add a test whose mock returns what the
  installer leaves behind.
- Settle the guard before the guarded service starts. Otherwise the service is exposed for a
  window, and repeating the run does not close it.
