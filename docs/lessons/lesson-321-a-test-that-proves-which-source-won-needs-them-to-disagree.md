---
id: lesson-321
type: lesson
status: active
created: "2026-09-30"
owner: manu
tags: [lesson, testing, mutation, doctor, fixtures]
---

# 321 — A test that proves which source won needs the sources to disagree

## What happened

CLI-066 (#1364) exists to make `dotf doctor --fix` heal the file pwsh names as `$PROFILE`, not the first of four guessed locations. The heal test asserted the exact argv, `-ProfilePath <profile>`. It passed, `features.json` f3 pointed at it, and the spec was marked done.

The independent review (2026-09-30) applied one mutant: `--fix` heals the enumerated file instead. The suite stayed green. The fixture had put the profile in `Documents/PowerShell/`, which is the first enumerated root, and the fake pwsh named that same file. Both sources gave the same answer, so the assertion could not tell which one the code had used.

## The rule

- When a test exists to prove that A is used rather than B, its fixture must make A and B give **different** answers. Here, the real profile sits outside every enumerated root, and a decoy sits in the first one.
- An exact-value assertion does not help if the wrong source produces that exact value.
- Check it with the mutant that swaps the sources. If that mutant survives, the test only shows the code runs, not that it picks the right source.

## Evidence

- Round 1, `specs/CLI-066-doctor-profile-target/review-round-1.md`, finding 2: the mutant survived (`ok … 0.648s`).
- After the fixture change, the same mutant fails `heal rewrites the profile → FIX…` on the argv assertion.
