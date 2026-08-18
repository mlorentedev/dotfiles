---
id: lesson-138-a-successful-operation-is-not-evidence-of-the-prop
type: lesson
status: active
created: "2026-06-27"
owner: manu
tags: [lesson, dotfiles]
---

# Lesson 138: A successful operation is not evidence of the property you depend on — assert the property, not the success

**Context**: Four incidents in the secrets/CI surface within two cycles: `sync ci` refreshed a PAT's `updated_at` (#639); `pat-expiry` probed `GET /user` (#647); `setup` `curl`ed a release tarball (#648); `AgeDecrypt` reported a decrypt failure (#644).

**Problem**: Each conflated "the operation reported success/failure" with "the property I actually depend on is true". A **write** landed (`gh secret set` ok, `updated_at` bumped) but the value was an **expired** token (#639 — write ≠ live). An **auth** succeeded (`GET /user` 200) but the token lacked `Pull requests: write`, so release-please's PR-create 403'd and every release silently stalled (#647 — authenticates ≠ authorized). A **download** completed (curl followed redirects, wrote a file) but, lacking `-f`, it saved a 404 "Not Found" body *as* the `.tar.xz`, detonating later at `xz` (#648 — bytes arrived ≠ bytes are the artifact). And a decrypt **failed** loudly but as opaque `exit status 1`, hiding age's actual cause (#644 — failure reported ≠ cause surfaced). In every case the misleading signal looked identical to the healthy one, and the truth surfaced far downstream (a 401 in production, a red release run, a corrupt extract) instead of at the operation.

**Solution**: Assert the downstream property explicitly, right after the operation, failing loud. #639 → opt-in pre-upload liveness (`gh api user` authenticating *as* the token under test). #647 → a capability probe (`POST /pulls` with a non-existent head; 403 = missing scope, 422 = permitted — non-destructive). #648 → `curl -fsS` so an HTTP error is the curl's failure, not the next step's. #644 → capture the child's stderr and surface its message, not the bare exit code.

**Rule**: When a step's success stands in for something you actually rely on — the value is *live*, the credential is *authorized*, the bytes are the *artifact*, the error names its *cause* — verify that property directly; it does not come for free with the operation returning 0. The cheapest place to assert it is immediately after, fail-loud; the most expensive place to discover its absence is three steps (or three days) downstream, as a silent corruption that looks exactly like success.
