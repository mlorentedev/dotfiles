---
id: "HARNESS-111"
type: verification
status: done
created: "2026-09-05"
---

# HARNESS-111 — verification

All commands run in the implementing session on 2026-09-05.

## The defect, measured

```
$ SB=$(mktemp -d); env HOME="$SB" bash scripts/compile-harness.sh --deploy >/dev/null 2>&1
$ wc -m < "$SB/.gemini/GEMINI.md"   # 11974   <- what the guard asserted
$ wc -c < "$SB/.gemini/GEMINI.md"   # 12047   <- 47 over the 12000 cap
```

Gap composition: 33 × U+2014, 4 × U+00A7, 1 × U+00E1, 1 × U+2026.

## After the fold

```
chars 11976   bytes 11985   cap 12000
```

Both under. Rejected alternatives measured rather than reasoned about: `--` → 12009 chars, ` - ` → 12042 chars, both over.

## AC2 — the assertion is proven in the failing direction

With the normalisation replaced by `if false` (equivalent to `main`):

```
not ok 18 HARNESS-056: the compact doctrine payload carries it and stays under its cap
# .gemini/GEMINI.md is 12047 BYTES, at or over its 12000 cap (chars: 11974)
```

The mutation was confirmed present (`grep -c 'if false; then'` → 1) **before** the verdict was read, per lesson 267.

## AC4 — no new lint findings

```
SC1112 on main:        0
SC1112 on the branch:  0
shellcheck rc:         1 on BOTH (pre-existing SC2016 infos)
```

The first attempt used literal characters and measured **3** SC1112 with rc 1 — the reason the escapes are hex.

## AC5 — parsers and idempotence

```
bash -n scripts/compile-harness.sh   clean
zsh  -n scripts/compile-harness.sh   clean
```

Running the script *under* `zsh` fails on `main` too; it is `#!/usr/bin/env bash`, so that is a pre-existing limitation and not a regression here.

## Suite

`bats tests/*.bats` — **1554/1554**, exit 0.

## Not verified, and stated rather than implied

**Which unit Antigravity actually counts.** The manifest declares `char_cap` citing "12000 characters", and nothing in this repository has tested that against the live consumer. This change makes the answer not matter for the payload; it does not establish it. The experiment that would — a marked sentinel at the end of the payload, deployed, then asked for — is recorded on #1241 and is not part of this spec.
