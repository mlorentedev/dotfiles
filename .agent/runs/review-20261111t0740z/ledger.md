# Ledger — review-20261111t0740z

| # | Item | Outcome | Evidence |
|---|---|---|---|
| 1 | merge-grant owner paths | PR #3308 open, owner merges (changes the grant) | go test ./internal/prland ok; mutation against main's grant fails |
| 2 | #3165, #3204 linked to #3221 | comments posted; close after a detached review on the release that carries #3221 | issuecomment-6790175826, -6790176129 |
| 3 | #3258 review base (spec BUG-166) | merged #3314 (0c6892f7), issue closed; independent review PASS-WITH-GAPS, its 3 Minors applied; PR-Agent's 4 findings declined with reasons; #3308 triaged, owner lands | go test ./... ok; lint 0 issues (linux, windows); mutations killed; dry-run names the fork point |
| 4 | #1551 | comment: conflicts with BUG-166's widen-only invariant; owner picks (A) close as superseded or (B) redefine | issuecomment-6791218244 |
| 5 | #1511 archive refusal quotes the dead reviewer | merged #3322 (d49de62, Refs #1511); spec BUG-167 signed PASS on round 2 (round 1 FAIL, both Majors applied) and stays active until the PR closing #1511 archives it; contract shapes A/B left to the owner; live repro denied by the permission classifier, measured by code read | go test spec+cmd ok; lint 0 issues (linux, windows); rune-cut and wrap mutations red; issuecomment-6795326258, -6796083045; triage issuecomment-6796235409 |
