---
id: "HARNESS-111"
type: spec
status: implementing # draft | implementing | verifying | archived
created: "2026-09-05"
issue: "mlorentedev/dotfiles#1241"   # repo#NNN — GitHub issue / Project item that tracks this spec
tags: [spec, proposal, harness, doctrine]
template_version: "1.0"
---

# HARNESS-111 — the doctrine cap and its guard measure different units

## Why

`.gemini/GEMINI.md` carries the compact doctrine payload under a hard 12000 platform cap. The guard that protects it asserts `wc -m`; the payload is 11974 **characters** and **12047 bytes**. If the platform counts bytes it has been dropping the tail since before 2026-09-05, and its documented overflow behaviour is to truncate **silently** — no error, no red check. A guard measuring one unit cannot report the other one crossing, which is why nothing noticed.

## What

Normalise typographic punctuation to ASCII in a **capped** doctrine payload, so bytes and characters track each other and the guard cannot disagree with the platform whichever unit the platform counts. Assert both units. This deliberately does **not** decide what Antigravity counts — it removes the question instead, because settling it needs an experiment against the live consumer and the fix should not wait on one.

**Out of scope:** the cap value, the per-id budget policy (#1241's own decision), and any vault doctrine prose. This changes only how the payload is rendered into the two `doctrine.deploy` targets.

## How

- `deploy_doctrine` folds em/en dashes, curly quotes and the ellipsis to ASCII when `char_cap != 0`, before the payload is injected.
- Only punctuation that **does not alter the lexicon**. Accents and section signs are left alone: `bitácora` → `bitacora` changes a word and `§4` → `S4` changes a reference. Surviving non-ASCII is reported by the deploy, not silently tolerated.
- Substitutions are written as **hex escapes**, because a curly quote in shell source is SC1112 and fails CI lint. Measured: the literal form added 3 findings and turned `shellcheck` rc 1.
- The deploy's cap warning reports both units and compares against the larger.
- `tests/skills-pipeline.bats` asserts both units for every `doctrine.deploy` target that declares a cap.

## Alternatives rejected

- **Raise `char_cap`.** #1241 already rejected this and its reasoning holds: 12000 is a real platform limit, so raising the number moves the truncation from a red check to a silent drop at the destination. Same overflow, no signal.
- **Change the assertion to `wc -c` and leave the payload alone.** Turns CI red immediately on an unverified hypothesis about the platform's counting. Red is right only if bytes is right, and nobody has established that.
- **Fold to `--` or ` - `.** Measured: 12009 and 12042 characters respectively — both land back over the cap. Only the single hyphen helps in both units.
- **Trim doctrine prose to fit.** Recovers the bytes but leaves the two measures free to diverge again on the next em-dash, and the prose belongs to another lane (#1495 is parked on exactly that content).

## Acceptance criteria

- **AC1** — After `--deploy`, every `doctrine.deploy` target with a `char_cap` is under that cap in **both** `wc -m` and `wc -c`.
- **AC2** — The bats assertion fails on a tree where the normalisation is disabled, naming the byte count and the cap. Proven by mutation, not by inspection.
- **AC3** — Characters that alter the lexicon (accents, section signs) are **not** folded, and any surviving non-ASCII is reported on stderr by the deploy.
- **AC4** — `shellcheck` reports no new findings against `main`, and in particular zero SC1112.
- **AC5** — `bash -n` and `zsh -n` both parse the script; the deploy is idempotent on a second run.
- **AC6** — The cap warning reports both units, so the number a reader quotes is the binding one.
