# Ledger tool031-20261022T0735Z

- 2026-10-22T07:35Z armed. merge: never (owner planned on merging A1 themselves; grant exists since #2229 but was not offered tonight).
- 07:50Z A1 stack verified on final tips (48 pkgs ok, -race ok, exec LOC 184/293/128) and opened as drafts #2940 -> #2941 -> #2942. lesson-516 in #2941. Debt #2928 filed. Next: CI, mark ready, triage.
- 08:07Z A2 (dotf pr preflight) verified (48 pkgs ok, -race ok, bats 10/10, exec 225) and opened as draft #2943 on #2940. Script defects 6-9 added to #2928.
- 08:19Z test (windows-latest) red on #2941-#2943: jq on Windows writes CRLF so the bash script rejects its own allowlist ids; differential tests now skip on Windows (repo idiom). Pushed fa80b50c (#2941), merge 4944ce5d (#2942), preflight fix (#2943).
- 08:39Z CI green on #2940-#2943 (review pending, cli-gate queued on #2943). Marked all four ready for review.
- 09:05Z Reviews on #2940-#2943 queued in PR-Agent's one-at-a-time lane (queue: max) behind other sessions' PRs; draft-time runs cancelled by ready_for_review as designed. CodeRabbit: review limit reached on #2940.
