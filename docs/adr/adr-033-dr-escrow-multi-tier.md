---
id: "ADR-033-dr-escrow-multi-tier"
type: adr
status: proposed
owner: manu
date: "2026-08-16"
supersedes: []
extends: [adr-028-secrets-two-tier-bitwarden-age]
issue: mlorentedev/dotfiles#1000
tags: [architecture, decision, secrets, disaster-recovery, backup, homelab, minio, age]
created: "2026-08-16"
---

# ADR-033: Disaster-recovery escrow — one artifact, many homes

> ADR-028 established *what* the escrow is (a full Bitwarden export, age-encrypted, round-trip
> verified). It did not say **how often it is refreshed, where else it lives, or how anyone knows
> it still restores**. Those three gaps are the entire content of this ADR, and each of them is
> currently answered by "nobody has decided".

## Status of the thing being decided

Measured 2026-08-16, not assumed:

| Fact | Value |
|---|---|
| Escrow artifact | `sensitive/dr/bitwarden-export.age`, 141 KB, tracked in git |
| Times refreshed, ever | **2** — created 2026-06-30 (`4683064`), refreshed 2026-08-15 02:54 (`5e79333`) |
| Schedule | **none** — no cron, no timer, no CI job |
| Recovery drill | **never run** — `dotf doctor` reports `[WARN] no recovery drill recorded` |
| Off-machine copies of the ciphertext | **1** (the GitHub repo) |
| Off-machine copies of the age key | 1 USB (VeraCrypt) + 1 Bitwarden item |

So "periodic, multi-level backup" is today a description of an intent, not of a system.

## The principle: one artifact, many homes

**Only the age blob travels.** No tier re-encrypts, no tier ever sees plaintext, and no tier needs
to be trusted — an age ciphertext is safe on storage we do not control. Tiers therefore differ in
exactly three ways: **transport, retention, and which loss they survive**.

This keeps the SSOT intact: `dotf secrets backup` remains the single producer, and a destination is
configuration rather than a second implementation. A tier that re-encrypted, or that exported
separately, would be a second producer and would drift.

## The failure matrix — and the row MinIO does not cover

Each tier must declare which column it survives. This is the part that makes "several levels"
meaningful rather than merely numerous.

| Tier | Location | Laptop lost | Bitwarden account lost | GitHub account lost | **House lost** (fire/theft) |
|---|---|---|---|---|---|
| 0 — Bitwarden cloud | vendor | ✅ | ❌ | ✅ | ✅ |
| 1 — working copy | laptop | ❌ | ✅ | ✅ | ❌ |
| 2 — git / GitHub | vendor | ✅ | ✅ | ❌ | ✅ |
| 3 — USB (VeraCrypt) | house | ✅ | ✅ | ✅ | ❌ |
| 4 — MinIO (Beelink) | **house** | ✅ | ✅ | ✅ | ❌ |
| 5 — geographic | off-site | ✅ | ✅ | ✅ | ✅ |

**The Beelink is in the same building as the laptop and the USB.** MinIO buys
*account-independence* — surviving the simultaneous loss of Bitwarden and GitHub — but it buys **no
geographic redundancy**. A fire or a burglary takes tiers 1, 3 and 4 together. Adding MinIO and
calling the problem solved would be the most likely mistake here, so it is stated in the table
rather than left to be discovered.

Note also that tier 0 and tier 2 are the only always-on tiers: per kubelab ADR-028 the Beelink is
an **on-demand** platform node — "services here tolerate being offline when homelab is off" — so
tier 4 is best-effort by design and must never be the only recent copy.

## Decision

### D1 — The producer stays single, destinations are configuration

`dotf secrets backup` produces the artifact. Distribution is a separate, dumb step that copies an
opaque blob. Nothing downstream may decrypt, re-encrypt, or re-export.

### D2 — Phase 1 (now, unblocked): prove it, not multiply it

Multiplying the tiers of a backup that has never been restored multiplies zero. In order:

1. **Run the recovery drill** against the real USB, end to end, and `touch ~/.dotfiles/.dr-drill`.
   What is already proven is narrower than it looks: `backup` verifies its own round-trip with the
   *local* key, and #1000 verified by fingerprint (`b4d18dc3867b`) that the Bitwarden copy matches
   it. **Never proven:** that the USB copy is that same key, and that the export re-imports
   (`bw import bitwardenjson`). Those are steps 1 and 4 of the runbook chain, and step 1 had no
   instructions at all until #848.
2. **Make staleness loud** (#997, and see D5 for the definition).
3. **Put the escrow on the USB.** `scripts/backup-secrets-to-usb.sh` copies a flat glob —
   `"$SECRETS_DIR"/*.secret` and `*.secret.age` — so it does not descend into `sensitive/dr/`. The
   full-vault escrow has never been on the USB. This is a three-line fix with a bats assertion, and
   it is the cheapest row-4 coverage available.

### D3 — Phase 2 (blocked on #1008): scheduled, unattended refresh

Phase 2 does not exist until #1008 resolves, and that dependency is mechanical, not a preference:
`bw serve` exposes **no export route**, so `dotf secrets backup` needs
`BW_SESSION="$(bw unlock --raw)"` — an interactive master password. **A systemd timer cannot supply
one.**

The promising path, to be evaluated on #1008: synthesize the export from the daemon's
`/list/object/items` + `/list/object/folders`, which a `bitwardenjson` export is essentially a view
over. The daemon is already unlocked, so this needs **no new credential and no stored master
password**. Its acceptance test must be fidelity, verified without exposing anything: decrypt the
real escrow to tmpfs and compare against the synthesized one by item-id set, key-path set, and
per-field sha256 — never by value.

Once that lands, the mechanism is already precedented: `systemd/dotfiles-selfupdate.timer` is the
exact shape, deployed as a `--user` unit with an opt-in runbook.

**Storing the master password to automate this is explicitly rejected.** It converts a read-only
backup capability into full account takeover, which changes the threat model the whole of ADR-028
is built on.

### D4 — The geographic tier is a git mirror, not a bucket, when one exists

The escrow is *already committed to git*. Any git mirror is therefore a complete ciphertext tier
for **zero new code and zero new credentials**, 24/7 and off-site. That makes a self-hosted Gitea
mirror on the VPS strictly better than an object-storage integration for this purpose.

It is not available today: kubelab#1077 (IDP-034, making Gitea usable as the private forge) is
**open**. So:

- **Declared direction:** Gitea mirror on the VPS = tier 5.
- **Interim:** MinIO on the Beelink = tier 4, which closes account-loss but not house-loss.

If MinIO is used, two constraints are not optional:

- **Write-only credential against a versioned bucket.** The pushing machine must not be able to
  delete history, so a compromised laptop cannot destroy the backups it uploaded. Bucket versioning
  must be confirmed enabled in the kubelab MinIO deployment — MinIO supports it; that it is *on* is
  a config fact to verify, not to assume.
- **The MinIO credential is a secret**, so it enters `secrets/registry.yaml` through the ADR-028
  flow like any other. Not a loose env var.

kubelab provides the bucket and nothing else; dotfiles remains the only producer. The repos do not
become entangled.

### D5 — Staleness is defined against the registry, not only the calendar

A pure calendar rule cannot see the failure that matters. The definition:

> The escrow is **stale** when the last commit touching `sensitive/dr/` predates the last commit
> touching `secrets/registry.yaml`.

A secret added after the last export means the escrow is *definitely* incomplete — a fact, not a
heuristic. A calendar backstop (proposed: 90 days) covers value rotation, which the registry does
not record. `dotf doctor` already carries a 180-day drill marker; this is the escrow's own clock.

## Accepted risk, stated rather than solved

**Git history is immutable.** If the age key is ever rotated, every escrow already committed
remains decryptable with the old key, forever, by anyone who has both. This is accepted: the key is
the crown jewel, it lives offline, and the alternative (history rewriting on a repo with mirrors)
is worse. It is recorded so that a future key rotation is not mistaken for retroactive protection.

## Decisions this ADR does NOT take — they are Manu's

This ADR is `proposed`, and it stays there until these four are answered:

1. **Escrow commits vs the PR gate.** dotfiles is PR-gated, so a *periodic* escrow means periodic
   PRs competing for the same review slot — the bottleneck measured on 2026-08-16, when three PRs
   sat unreviewed simultaneously. Either escrow-path commits are exempted from the gate, or the
   frequent tier bypasses git entirely (USB/MinIO) and git receives only a monthly commit.
2. **Which tier is the geographic one:** wait for Gitea (kubelab#1077), pay for a bucket, or accept
   house-loss risk with MinIO alone.
3. **Drill cadence** — `doctor`'s existing marker assumes 180 days.
4. **Whether the escrow goes on the USB now.** It interacts with #971: `migrate` drops the `age:`
   pointer, so the per-secret age floor **no longer exists for the 28 migrated secrets**. That makes
   escrow-on-USB more urgent, not less — the USB currently carries a floor that has been hollowed out.

## References

- [ADR-028](adr-028-secrets-two-tier-bitwarden-age.md) — the two-tier model this extends
- `docs/runbooks/guide-secrets-governance.md` § RECOVER + § Drill it
- `#1000` (OPS-030) — the key's circular dependency, and the USB/ciphertext gap measured on it
- `#1008` (OPS-031) — no export route in `bw serve`; the Phase 2 blocker
- `#997` (BUG-085) — escrow absence reported as SKIP
- `#971` (BUG-078) — `migrate` drops the `age:` pointer
- kubelab#1077 (IDP-034) — Gitea as private forge, gating D4's declared direction
- kubelab ADR-028 — Beelink as an on-demand platform node (why tier 4 is best-effort)
