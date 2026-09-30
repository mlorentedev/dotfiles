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

## Review round 1 (FAIL, 2026-09-30) — dispositions

`review-round-1.md` (`agy/gemini-3.1-pro-high`):

- **Blocker, the code folds accents and section signs against AC3: resolved by amending the contract, not the code.** The fold was widened on purpose in #1685 (2026-09-24). At that point the deployed file still carried 7 non-ASCII characters, so a cap enforced in two units depended on which unit a consumer counted. The reasoning is at `fold_to_ascii`. What AC3 protected still holds: nothing outside the fixed table is guessed at, and whatever survives is reported by its bytes. `proposal.md` AC3 now says that and names #1685. `features.json` f3 now runs the two tests that assert it. Its old command ended in `rm`, so it always exited 0.
- **Major, the cap warning printed one unit: applied.** Both warnings now print `N characters / M bytes`, and the comparison uses the larger. Before this, the user-content warning printed the promoted byte count and labelled it "characters". Tests: "the over-cap warning for the generated doctrine names characters and bytes" and "the over-cap warning for a user's file names characters and bytes, and they differ", both red before the fix. f6 runs them. Its old grep matched the fold warning, not a cap warning.
- Checks: `tests/compile-harness.bats` 83/83; shellcheck has the same 7 SC2016 infos as `main` and no new finding; `bash -n` and `zsh -n` are clean.

## Review round 2 (PASS-WITH-GAPS, `nan/glm5.3-flash`, 2026-09-30) — dispositions

Round 2's first draw, `nan/qwen3.8-flash`, ended without a review: NaN refused it with `402 monthly_cap_reached` (#1875). The round was re-run with `--reviewer nan/glm5.3-flash`.

| # | Finding | Disposition |
|---|---|---|
| 1 | Minor: AC2's mutation proof no longer reproduces at HEAD. The committed payload is 8111 bytes, far under the 12000 cap, so the byte assertion cannot fire | apply: the evidence above is historical, recorded on 2026-09-05 at 11974 chars and 12047 bytes. Today's guard chain is the 8000-character budget, then `doctrine: a capped surface is folded to pure ASCII, marker and preamble included` (mutation-verified this round), then the byte assertion as a backstop. The `tests/skills-pipeline.bats` comment now says so |
| 2 | Minor (theoretical): `deploy_doctrine` is about 100 lines, over the function-length rule | defer: #1876 |
| 3 | Minor (speculative): a string-valued `char_cap` would break the arithmetic | decline: the manifest is authored in this repo and every `char_cap` is a JSON number |

## Promotion candidates

- [x] Lesson for the repo's `docs/lessons/`? no: the failure class, a verification command that cannot fail, is lesson 309, already written.
- [x] ADR-worthy decision for the repo's `docs/adr/adr-XXX.md`? no: the fold decision is recorded at `fold_to_ascii` and in #1685.
- [x] New pattern candidate for `00_meta/patterns/`? no: nothing here is cross-project.
