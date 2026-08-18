---
id: lesson-137-same-set-as-the-script-it-replaces-is-the-wrong-pa
type: lesson
status: active
created: "2026-06-27"
owner: manu
tags: [lesson, dotfiles]
---

# Lesson 137: "Same set as the script it replaces" is the wrong parity gate when the old tool was itself wrong

**Context**: `dotf secrets sync ci` replaces `github-secrets-manager.sh` as the path that uploads secrets to a repo's GitHub Actions. The instinct when retiring the script was to gate the cutover on "the new command uploads the same set of secrets the script did" — set-equality as the parity proof.

**Problem**: The old script had **no consumer filter** — it pushed every pair it knew about to the repo, over-uploading secrets no workflow in that repo consumes. Gating on set-equality would have forced the new command to faithfully reproduce that over-upload, i.e. to inherit the bug as a spec. The script's behaviour was the *defect under repair*, not the reference to match.

**Solution**: Reframe parity as **functional coverage**, not set-equality. `sync ci` selects by intent: `reg.SelectCI(repo)` returns exactly the registry entries whose `consumers:` contains `ci:<repo>` (`secrets_sync.go`), so it uploads what the repo's workflows actually consume and nothing more. The cutover gate became "every secret the target repo's workflows reference resolves and uploads" (a grep of `.github/workflows/` for consumed names), verified per-repo — not "the byte-set equals the script's output". Migration of `ci:*` entries was scoped to the same evidence: only the names the workflows actually consume.

**Rule**: When replacing a tool, do not adopt its output as the correctness oracle — first ask whether the old behaviour was right. If the legacy tool was over-broad, permissive, or buggy, set-equality parity *encodes the bug* into the replacement. Define the gate from first principles (what does the consumer actually need?) and let the new tool's narrower, correct output differ from the old — then verify functional coverage, not byte-for-byte sameness.
