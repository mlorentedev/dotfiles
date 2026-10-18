---
id: "lesson-477-a-vendor-gate-that-rejects-automation-is-an-input-boundary"
type: lesson
status: active
title: "A vendor gate that rejects automation is an input boundary"
created: "2026-10-17"
---

# A vendor gate that rejects automation is an input boundary


**Context:** PLAT-003 Tier 3b needs Microsoft's Windows 11 Arm64 ISO. The owner asked whether `dotf` could download it instead of a browser. Microsoft's download page is backed by a public API, the one Fido (Rufus) and Mido (quickemu) script.

**Problem:** Measured on 2026-10-17: the page, the edition id (`3816`), the per-language SKU list (`27170` for Spanish) and the published SHA256 table all answer a plain `curl`. The last step, `GetProductDownloadLinksBySku`, answers `ErrorSettings.SentinelReject`: Microsoft's bot protection. The tools that still get past it emulate the browser fingerprint script Microsoft serves, which circumvents a deliberate anti-automation control, breaks whenever the script changes, and spends this IP's reputation on every rejected try.

**Solution:** Treat the media as an input the owner obtains once, as Packer users supply an `iso_url`, and automate the half that works: `dotf` pins the ISO's SHA256, accepts either the file or the browser's 24-hour link, downloads and verifies, and on a mismatch names both hashes, so a Microsoft refresh is a one-line pin bump. One rejection was the evidence; the request was not retried.

**Also measured, 2026-10-18:** the published table is not a reliable source for the pin. The ISO the page served for "English" (volume label `CCCOMA_A64FRE_EN-US_DV9`) hashes to the value the table lists under "Arabic 64-bit"; the "English 64-bit" row holds another file's hash. The table proves a file is one Microsoft published, not which language it is. Read the language from the file itself (its ISO 9660 volume label, 32 bytes at offset 32808) before pinning it.

**Why:** Zero manual operations is about machine state, which must be reproducible from the repo. Licensed vendor media behind a gate that refuses automation is not machine state, and the value `dotf` adds is in verifying it, not in defeating the gate.
