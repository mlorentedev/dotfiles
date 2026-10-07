---
id: "lesson-361-a-remote-url-is-a-credential-carrier"
type: lesson
status: active
title: "A remote URL is a credential carrier: report its host, never the URL"
created: "2026-10-07"
---

# A remote URL is a credential carrier: report its host, never the URL

## Context
#2116 and #2117 taught `dotf init` and `dotf vault project` to read the origin remote: one to
classify the forge, the other to find a vault entry by `repo_url`. Review found the same exposure
three times. The init report printed the origin in its skip message. The vault entry recorded it
verbatim as `repo_url` in `context.md`. `ParseOriginRepo` quoted it whole in its error, which the
init report prints as a warning.

## The Trap
A self-hosted or CI remote often embeds a token: `https://oauth2:<token>@git.example.lan/o/r.git`
is how Gitea, GitLab and GitHub Actions hand out push access. Code that reads `git remote get-url`
gets the token along with the address, and nothing marks it as a secret. Every place the URL is
printed, written or wrapped into an error then leaks it: a terminal, an agent transcript, a synced
vault repository. A test on an ordinary `git@host:o/r` or `https://github.com/o/r` origin cannot
see the problem.

## The Solution
- Reports name the host only. `originHost` strips the scheme, the user info and the port, and
  answers "a local path" when there is no host.
- Anything recorded keeps the URL without its user info. `withoutCredentials` drops `user:token@`
  and any `@` user info on http(s), and keeps a password-less ssh user such as `git@`, which is
  part of the address.
- Errors name the host, never `%q` of the URL.
- Tests feed a credentialed origin and assert the token is absent from the report, the error and
  the written file. Build the fixture at run time (`"https://oauth2:" + fakeToken + "@host/..."`):
  a literal `user:password@host` in a test file is itself reported by secret scanners (GitGuardian
  flagged #2117).

The rule for new code: treat a remote URL like an environment variable that may hold a secret.
Before it reaches output or disk, reduce it to the part the reader needs, which is almost always
the host.
