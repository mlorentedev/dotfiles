---
tags: [spec, verification, templates]
created: "2026-08-29"
---

# Verification - CLI-066-doctor-profile-target

## Evidence

Map every acceptance criterion from `proposal.md` to concrete proof (commit hash, test name, or observed behavior).

- [x] AC1 (doctor measures the pwsh-resolved `$PROFILE`, outside the enumerated roots) -> commit `1925da6` / `TestCheckProfileFiles_MeasuresThePwshResolvedProfile` cases "redirected Documents…" and "pwsh answers a path that does not exist yet…"
- [x] AC2 (no pwsh, or pwsh silent → enumeration, row says so) -> same test, cases "no pwsh on PATH…" and "pwsh present but fails to answer…"
- [x] AC3 (`--fix` invokes the heal with `-ProfilePath <measured>` through the bounded seam) -> `TestCheckProfileFiles_FixRunsTheHealAndVerifiesByConsequence` (argv assertion updated; the unbounded seam is a `t.Fatalf` in the harness)
- [x] AC4 (`profile-heal.ps1 -ProfilePath`, `$PROFILE` default, ASCII, analyzer-clean) -> `tests/profile-heal-ps1.bats` (16/16), PSScriptAnalyzer 1.25.0 0 findings, box transcript below
- [x] AC5 (thresholds linked by test) -> `TestProfileHealThresholdsMatchTheScript`
- [x] AC6 (box) -> transcript below, Windows work box, 2026-08-29

## Test status

- Test suite: `cd cli && go test ./... -count=1` -> every package `ok`, `FAIL_COUNT=0`; `go vet` clean under `GOOS=windows` and `GOOS=linux`; `golangci-lint run` (pinned 2.12.2) `0 issues`
- `bats tests/profile-heal-ps1.bats` -> 16/16; `Invoke-ScriptAnalyzer scripts/profile-heal.ps1` -> 0 findings, 0 non-ASCII characters
- Manual smoke test (AC6), binary built from this branch:

  ```text
  --- doctor row (dotf doctor --verbose) ---
  [ OK ] PowerShell profile exists (C:\Users\<user>\Documents\PowerShell\Microsoft.PowerShell_profile.ps1; resolved by pwsh $PROFILE)
  --- scratch heal via -ProfilePath ---
  scratch START markers before: 2
  [profile-heal] corruption detected: marker counts (start=2, end=2) exceed 1
  [profile-heal] heal complete -- restart PowerShell or dot-source the profile
  scratch START markers after:  1
  backup beside scratch: 1
  real profile untouched: True  (SHA256 before == after)
  ```

  The box's Documents is not redirected (`[Environment]::GetFolderPath('MyDocuments')` is
  `~\Documents`), so the redirected case is proven by the Go test with a fake `$PROFILE`
  answer outside every enumerated root; the box proves the real question-and-answer path
  and that the heal targets the file it is given.
- No regressions in existing test suite: yes. Doctor's default output summarises an all-OK
  section as `(3 checks, all ok)`; the row is visible under `--verbose`, which is what f6's
  command uses.

## Decisions made during implementation

- **Ask pwsh, do not guess.** The heal resolves `$PROFILE` inside pwsh, so doctor asks pwsh the same question through the bounded seam (10 s) and measures the answer. The four-root enumeration is kept only as the fallback for a box without pwsh or a pwsh that does not answer, and the row always says which of the two produced the path — a fallback that hides itself is the CLI-064 split again in a new coat.
- **The heal is told which file.** `-ProfilePath` on the script, passed by doctor; without it the script behaves exactly as before, so a hand run needs no new knowledge. `TestProfileHealThresholdsMatchTheScript` also pins the parameter's presence, since doctor's argv depends on it.
- **Bounded, like every other probe.** The heal runs through `CommandOutputBounded` (60 s). The test harness makes the unbounded seam a `t.Fatalf`, so a future edit cannot quietly go back.
- **Outside-marker content is documented, not preserved.** The heal rewrites the whole file from the SSOT; the FAIL/FIX lines and the script's synopsis say that only the backup keeps what lived outside the markers. Preserving it is a different feature no real profile has asked for.

## Review round 1 (FAIL, 2026-09-30) — dispositions

`review-round-1.md` (`nan/qwen3.8-flash`) failed on two Majors. Every finding is dispositioned here:

- **Major 1, a missing profile reported as corruption: applied.** A pwsh-named `$PROFILE` that does not exist is now `PowerShell profile missing: <path> (resolved by pwsh $PROFILE) — run setup-windows.ps1`. It carries no BUG-020 remedy, and `--fix` does not run the heal. This is what `proposal.md` Risks already stated. The subtest pins `PowerShell profile missing: ` and rejects `BUG-020`. Mutant: dropping the branch turns that subtest red.
- **Major 2, the `--fix` path could not tell the two sources apart: applied.** The heal fixture's profile is now in `Redirected/Docs/…`, outside the four roots, and a decoy sits in `Documents/PowerShell/`. The review's mutant (`profile = split` before the heal) now fails `heal rewrites the profile → FIX…` on the argv assertion. In round 1 it passed.
- **Minor 3, unquoted remedy: applied.** Both paths are double-quoted. `TestCheckProfileFiles_DetectsBUG020Corruption` asserts `-File "` and `-ProfilePath "` on every corruption FAIL.
- **Minor 4, the answer's shape is trusted: applied.** `profileTarget` takes the last non-blank line and accepts it only if it names a `.ps1` file; anything else falls back to the enumeration, with the reason. New cases: noise before the path (resolved, and the noise is not in the row), and noise only (enumerated).
- **Question 5, the launcher's base range: answered, out of scope.** The reviewer measured the spec's own change (`dab595e`) and found no later edit to its files, so the verdict covers this spec. The stale inferred base is tracked in #1551, and the missing way to target a landed commit in #1645.

## Promotion candidates

Before archiving, flag what (if anything) should be promoted to the vault. If all three are "no", archive in repo is the only persistence.

- [ ] Lesson for the repo's `docs/lessons/`? no — the class ("detect and heal must name the same target") is CLI-064's review finding, recorded there and in this spec; nothing new was learned beyond applying it
- [ ] ADR-worthy decision for the repo's `docs/adr/adr-XXX.md`? no
- [ ] New pattern candidate for `00_meta/patterns/`? no

## Archive checklist

- [ ] `proposal.md` frontmatter set to `status: archived`
- [ ] Folder moved: `specs/CLI-066-doctor-profile-target/` -> `specs/archive/CLI-066-doctor-profile-target/`
- [ ] Bitácora board ticket for this spec moved to Done / closed with PR link (ADR-018)
- [ ] Promotions above executed (if any)
