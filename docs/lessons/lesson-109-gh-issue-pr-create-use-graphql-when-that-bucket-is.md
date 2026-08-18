---
id: lesson-109-gh-issue-pr-create-use-graphql-when-that-bucket-is
type: lesson
status: active
created: "2026-06-18"
owner: manu
tags: [lesson, dotfiles]
---

# Lesson 109: `gh issue/pr create` use GraphQL — when that bucket is rate-limited, `gh api -X POST` (REST) still works

**Context**: Mid-session, `gh repo view --json` and `gh label list` failed with "GraphQL: API rate limit already exceeded", blocking issue/PR creation.

**Problem**: GitHub keeps *separate* rate-limit buckets for REST (5000/h) and GraphQL (5000/h). `gh issue create`, `gh pr create`, `gh label list` and `gh project` go through GraphQL; when that bucket is exhausted they all fail — while the REST bucket can be completely fresh.

**Solution**: Create the issue/PR via REST — `gh api -X POST /repos/<owner>/<repo>/issues --input -` (and `/pulls`), feeding a JSON body; check both buckets with `gh api /rate_limit --jq '.resources'`. PowerShell gotchas: capture a fetched body as a single string (`-join "\n"`, since the tool splits multiline output into an array) and use case-sensitive `-creplace` — plain `-replace` is case-insensitive and corrupts e.g. `adr-023` → `ADR-025`; build the payload with a single-quoted here-string `@'...'@` so `$`-vars stay literal.

**Rule**: On a `gh` "GraphQL rate limit" error, don't wait — fall back to `gh api` REST endpoints (separate bucket), verified via `gh api /rate_limit`. For scripted body edits: single-string + case-sensitive replace + literal here-string.
