---
id: "lesson-437-a-drift-check-must-know-who-reads-the-variable"
type: lesson
status: active
title: "A drift check must know who reads the variable"
created: "2026-10-14"
owner: manu
tags: [go, doctor, env-contract, false-positive]
---

# A drift check must know who reads the variable

**Context:** #2501. On every Mac session start `dotf doctor` warned `HERDR_CONFIG_DIR unset; default ... would be reported with --fix`, and `--fix` answered with `add to your shell profile: export HERDR_CONFIG_DIR=...`. The contract entry says the variable is optional, has a default on every OS and is read by `dotf deploy` only. `dotf deploy herdr --dry-run` reported `in sync` with the variable unset.

**Problem:** the check had two inputs, "optional" and "has a default", and read both as "the shell should export it". That holds for a variable a foreign program reads (`CLAUDE_CONFIG_DIR`, `SOPS_AGE_KEY_FILE`): the program cannot apply our default, so an unset value really is a different behaviour. For a variable only dotf reads, dotf applies the default itself and unset is the designed state. The warning described a correct machine, and its fix was a manual profile edit, which the repo forbids.

**Solution:** the contract states who resolves the default: `dotf_resolves_default: true`. Doctor reports such an unset variable as a PASS, in check and in `--fix` mode alike, and never as a WARN or a profile line. The flag is opt-in and scoped to optional variables, so a variable a foreign program reads still warns. Two tests pin it: a unit test with a flagged and an unflagged twin, so the rule is the flag and not "optional with a default", and a test over the shipped contract so `HERDR_CONFIG_DIR` keeps the flag.

**Why:** a check that compares a value against a default is only right if it knows whether anything other than the default's owner reads the value. Record the reader in the data (the contract) rather than guessing it in the check, or every new internal variable repeats the warning.
