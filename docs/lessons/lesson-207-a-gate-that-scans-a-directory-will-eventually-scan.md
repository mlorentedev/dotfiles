---
id: lesson-207-a-gate-that-scans-a-directory-will-eventually-scan
type: lesson
status: active
created: "2026-08-15"
owner: manu
tags: [lesson, dotfiles]
---

# Lesson 207: A gate that scans a directory will eventually scan its own evidence

**Context**: HARNESS-072 (#963) had a passing adversarial review and one step left — `dotf spec archive`. The archive refused, listing dozens of unresolved `[AGENT-DRAFT]` markers in a spec whose artifacts a manual `grep` showed were clean.

**Problem**: `FindUnresolvedTags` walked *every file* in `specs/<id>/`. The review machinery writes two of its outputs into that same folder, and the adversarial-review skill instructs the reviewer to check the spec for those very markers — so checking writes the literals into the reviewer's own output. Measured per file: 3593 hits in `review-transcript.jsonl` (552 MB), **1 hit in `review.md`**, and 0 in every authored artifact. The `review.md` hit is the sharp one: it is the row reading `| No [AGENT-DRAFT] tags | ✅ | ... → none |`. The reviewer certifying the spec was clean is what made the scan call it dirty. The gate required a review, the review's output failed the gate, and the only exit — `--force-with-drafts` — also disables the real check. So every spec this tool reviewed either could not archive or archived unguarded, and the workaround people reached for (move the transcript out) could not work either, because `review.md` is mandatory and stays.

The same day produced the same shape in a second place. `.gitignore` held `specs/*/review-transcript.jsonl`, and a `*` does not cross a `/` — so the rule stopped applying the instant `Archive()` renamed the spec into `specs/archive/<id>/`, which is precisely the moment the file gets committed. Archiving HARNESS-072 would have tried to push 552 MB, over GitHub's 100 MB hard limit, and the failure would have surfaced as a rejected push with no visible connection to the archive.

**Solution**: scan by artifact identity, not by location. `FindUnresolvedTags` now skips `review.md`, the transcript and its `.stderr` sibling by exact name, and keeps walking everything else — a deny-list, so an artifact added later is guarded with no code change, and the failure direction stays "refuse the archive" rather than "stop looking". The ignore rule was widened to `specs/**/` so it survives the rename, verified against a real archived path rather than reasoned about. Before/after on the live folder: 3594 hits → 0, and 1 ms instead of reading 552 MB into memory — the walk had also been `os.ReadFile`-ing a half-gigabyte log on every archive attempt.

**Rule**: a directory is a shared workspace, so any check written as "everything under here" silently assumes nothing else writes there — and the tools in the pipeline always do. Before scanning a folder, ask which processes emit into it and whether the check would flag their output; a guard that reads the evidence it demanded is self-refuting, and it fails at the end of the workflow, where the cost of a wrong answer is highest. The related tell is a rule keyed to a *path* for a thing that later *moves*: ignore rules, allow-lists and CI path filters all expire silently at a rename, and `*` not crossing `/` makes the expiry invisible in review. Two cheap habits cover both — express the rule over the set of artifacts it means, not the place they currently sit, and where a path is unavoidable, test it at the destination as well as the origin (`git check-ignore -v` against the post-move path costs one command). A corollary worth keeping: **gitignored is not invisible.** An untracked file is absent from `git status` and from every diff, and still fully present to `filepath.WalkDir` — which is exactly how a 552 MB file nobody could see in a PR became the thing that blocked one.

**Tags**: `spec-driven-development`, `verification`, `git`, `guards`, `false-positive`
