---
id: lesson-149-validating-config-files-in-isolation-cannot-catch-
type: lesson
status: active
created: "2026-08-04"
owner: manu
tags: [lesson, dotfiles]
---

# Lesson 149: Validating config files in isolation cannot catch a broken reference between them

**Context**: Adding the `deepseek-v4-flash-0731` model to pi touches two files that must agree: `ai/pi/models.json` declares the model `id`, and `ai/pi/settings.json` enables it as `nan/<id>`. `tests/pi-config.bats` already had seven assertions over these files.

**Problem**: The settings entry read `nan/deepseek-v4-flash 0731` — a space where the id has a hyphen — so the model would never have resolved. The new model also carried the display name of the model beside it (`"deepseek (NaN)"`), which would have shown two indistinguishable entries in the picker. All seven tests passed: both files were valid JSON, neither held a literal API key, the `{env:NAN_API_KEY}` placeholder was present, no banned OpenRouter provider was named. Every assertion validated one file **in isolation**, and both defects lived in the relationship *between* the files, which nothing checked.

**Solution**: Four guards in `tests/pi-config.bats` (#749): every `nan/*` entry in `enabledModels` resolves to an id in `models.json`; `defaultModel` resolves; model ids are unique; model display names are unique. The id-uniqueness and name-uniqueness checks are deliberately separate — the two ids differed (`deepseek-v4-flash` vs `…-0731`) and only the names collided, so one check could not have caught the other's defect. The reference check reads its input line by line rather than word-splitting: an earlier draft split on whitespace and reported only the fragment `0731`, hiding the shape of the bug; it now reports `'nan/deepseek-v4-flash 0731'` whole.

**Rule**: When two config files must agree, per-file validity checks (valid JSON, no secret, no banned value) prove nothing about the pairing — add an explicit assertion that every cross-file reference **resolves**, plus uniqueness on every field a human or UI selects by. Derive the referenced set from the source of truth at test time rather than restating it, so the guard cannot drift. And when a guard reports a bad value, print it quoted and whole: a message mangled by word splitting sends the reader after the wrong defect.
