---
id: lesson-172-gh-splits-its-subcommands-across-two-rate-limit-po
type: lesson
status: active
created: "2026-08-08"
owner: manu
tags: [lesson, dotfiles]
---

# Lesson 172: `gh` splits its subcommands across two rate-limit pools, so a polling loop can exhaust the one you need

**Context**: Waiting on a PR's CI with a background loop calling `gh pr checks` every 20 seconds, while separately reading the issue backlog.

**Problem**: `gh issue list`, `gh pr checks` and `gh pr view` travel over GraphQL; `gh api repos/...` travels over REST, and the two carry independent hourly quotas. The polling loop drained GraphQL to 0/5000 while REST sat untouched near 4900/5000 — so `gh pr checks` and `gh issue create` both failed while every equivalent REST call kept working. The same exhaustion had already broken an unrelated `release-please` run that hour: the release published correctly and only its *second* phase, building the next release PR, died on the limit, leaving a red run beside a perfectly good release.

**Solution**: Route status polling and issue creation through `gh api` — `repos/{o}/{r}/commits/{sha}/check-runs` to read checks, `POST repos/{o}/{r}/issues` to file — and set the interval from how fast the state actually changes: 60s+ for a CI run, never 20s.

**Rule**: Before writing any loop against a hosted API, establish which transport each call uses and what it costs; a wait loop is the highest-volume caller in a session and must always take the cheapest path. And when a tool starts failing on rate limits, check the *other* pool before concluding you are blocked — a red run under an exhausted quota is often a partial success whose remaining phase never ran, not a failure of the work itself.
