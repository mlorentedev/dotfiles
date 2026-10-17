---
id: "ADR-053-the-windows-test-guest-runs-on-qemu-driven-by-dotf"
type: adr
status: accepted
owner: manu
date: "2026-10-17"
supersedes: []
extends: [adr-020-tooling-cli-go-convergence, adr-045-one-entrypoint-converges-every-os]
issue: mlorentedev/dotfiles#2344
tags: [architecture, decision, virtualization, windows, testing]
created: "2026-10-17"
---

# ADR-053: The Windows test guest runs on QEMU, driven by `dotf`

## Context

PLAT-003 Tier 3b (#2344) needs a Windows 11 guest on the owner's Mac that reverts to a clean snapshot before each run, so Windows behaviour is verified there instead of on a physical box. The issue assumed VMware Fusion and `vmrun revertToSnapshot`, and the spec said "VMware Fusion or UTM". Measured on 2026-10-17, with nothing installed:

- **VMware Fusion** has no Homebrew cask any more; its download sits behind a Broadcom login. Installing it is a manual step, which the zero-manual-operations rule forbids.
- **UTM 5.0.8** (the cask, stable since 2026-10-17) does have `utmctl snapshot restore`, `exec` and `file push` (`utmctl/UTMCtl.swift` at the `v5.0.8` tag), but runs only on macOS, and `utmctl` refuses to work from an SSH session (`UTMCtl.swift:158`), so no unattended or remote run could use it.
- **Parallels** exposes `prlctl` only in its paid Pro and Business editions.
- **VirtualBox 7.2** is free and has one CLI on every host, but its macOS package needs root, its Linux host conflicts with KVM unless the kernel is 6.16 or newer, and Oracle does not list Linux on Arm64 as a host.

The owner set the criterion: the professional choice that works the same on Linux, Windows and macOS, open source and free.

## Decision

1. **The hypervisor is QEMU** (GPL-2.0), reached as the `qemu` package of the OS manager and accelerated by the host's own hypervisor (HVF on macOS, KVM on Linux). Windows 11 needs TPM 2.0, which `swtpm` (BSD-3-Clause) emulates. Both are `system` entries in `packages.json`; no step needs root or a kernel module.
2. **`dotf` owns the guest definition**, as it owns the Lima template of Tier 3a: the QEMU arguments, the firmware, the TPM socket and the guest-agent channel are code in `cli/internal/vm`, tested against a fake, not a GUI's saved configuration.
3. **Commands reach the guest through the QEMU guest agent, never SSH.** `clean-base` is a Windows with no `sshd`, because the first use of the guest is verifying that `dotf` installs `sshd` from nothing (#2672).
4. **The guest runs the Windows from-zero path of ADR-045**: `install.ps1`, which installs `dotf` and runs `dotf converge`, then a second `dotf converge` that must report `changed` 0. The legacy `setup-windows.ps1` is not what it certifies.
5. **No credential enters the image or a snapshot.** The guest has a local test account with no password; a Windows licence key, if the owner provides one, is applied after a revert and never snapshotted.

## Consequences

- One tool on every host, the same substrate Lima wraps on Linux and UTM wraps on macOS. "The same on every host" means the same tooling, not the same image: the guest architecture follows the host CPU (Windows 11 Arm64 on the Mac, x64 on an x86 host).
- `dotf` carries more code than a wrapper over a hypervisor CLI: an argument builder and a guest-agent client.
- The guest agent runs as LocalSystem, so a command it starts runs with the system profile, not a user's. How the from-zero run reaches the test user is measured on the real guest and recorded with the implementation.
- Only the macOS host is built now: `qemu` and `swtpm` are declared for Homebrew only, so no Linux machine pulls QEMU with no consumer. The Linux host is a follow-up. A Windows host is the weakest: `swtpm` has no native Windows build.
- The Windows image is the owner's download (Microsoft's multi-edition Arm64 ISO, behind a session link); `dotf` verifies it against the SHA256 Microsoft publishes for the chosen language. Measured 2026-10-17: the page, the edition id, the SKU list and the hash table answer a plain request, but the download-link API answers `SentinelReject`, Microsoft's bot protection. `dotf` does not emulate a browser to get past it (lesson 475). The owner's one manual act is getting the 24-hour link in a browser; `dotf` takes the link or the file, downloads and verifies.

## Alternatives rejected

- **VMware Fusion**: no unattended install path.
- **UTM**: macOS only, and no run from an SSH session.
- **Parallels**: paid, and its licence key would become a secret the catalog routes.
- **VirtualBox**: root on macOS, a KVM conflict with Tier 3a's Lima on Linux hosts below kernel 6.16, and no official Linux Arm64 host.
- **Packer or Vagrant as the driver**: no longer open source (BSL since 2023), and `dotf` already plays the driver role for Tier 3a.
