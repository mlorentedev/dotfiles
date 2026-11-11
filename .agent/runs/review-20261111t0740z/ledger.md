# Ledger — review-20261111t0740z

| # | Item | Outcome | Evidence |
|---|---|---|---|
| 1 | merge-grant owner paths | PR #3308 open, owner merges (changes the grant) | go test ./internal/prland ok; mutation against main's grant fails |
| 2 | #3165, #3204 linked to #3221 | comments posted; close after a detached review on the release that carries #3221 | issuecomment-6790175826, -6790176129 |
| 3 | #3258 review base (spec BUG-166) | implemented on fix/review-base-range (153768da, 6a18a191); detached review running; #3308 triaged, owner lands | go test ./... ok; lint 0 issues (linux, windows); 2 mutations killed; dry-run on the #3258 shape names the fork point |
