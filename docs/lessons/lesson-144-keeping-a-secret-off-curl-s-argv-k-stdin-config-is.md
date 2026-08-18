---
id: lesson-144-keeping-a-secret-off-curl-s-argv-k-stdin-config-is
type: lesson
status: active
created: "2026-07-08"
owner: manu
tags: [lesson, dotfiles]
---

# Lesson 144: Keeping a secret off curl's argv: `-K -` (stdin config) is portable; process-substitution and `mktemp` are not

**Context**: #687 (audit C26) required moving a bearer token out of `curl -H "Authorization: Bearer $KEY"` — argv is world-readable via `/proc/<pid>/cmdline` for the call's duration — in the `nan-*` benchmark scripts. The issue suggested curl's `-H @file` form. The scripts run on Linux but were being edited and tested from a Windows box, so the mechanism had to survive both.

**Problem**: The obvious "no temp file" idiom — process substitution `-H @<(printf 'Authorization: Bearer %s' "$KEY")` — failed on Windows curl with `curl: Failed to open /proc/<pid>/fd/63`: a native Windows curl (Schannel/mingw32) cannot resolve the MSYS `/dev/fd` pseudo-path that Git Bash hands it. The temp-file idiom (`mktemp` + `-H @file`) worked, but `mktemp` creates the file `0644` on MSYS (not `0600` as on Linux), so the header file holding the secret is briefly world-readable unless an explicit `chmod` follows. Both facts were found empirically against a local listener — not from docs, which describe none of these platform quirks.

**Solution**: Feed the whole auth header to curl through a config read from stdin: `curl -K - <<CFG` / `header = "Authorization: Bearer $KEY"` / `CFG`. The secret lives only in stdin (never argv, never disk), it is portable (no `/dev/fd`, no temp-file perms), and needs no cleanup. Verified against a local HTTP listener that the header arrives and the token never reaches any process's argv. For `gh secret set`, the equivalent is piping the value on stdin — `printf '%s' "$PAT" | gh secret set NAME` — using `printf` (not `echo`) so no trailing newline corrupts the token.

**Rule**: To keep a secret off a child process's argv, prefer feeding it via **stdin** (`curl -K -`, `gh secret set` piped) over a temp file — and never use process substitution for it in code that may run under Windows curl, where the `/dev/fd` path is unresolvable. When a temp file is unavoidable, remember `mktemp` is not `0600` on every platform (MSYS makes it `0644`); `chmod 600` before the secret is written. And prove the mechanism empirically (a local listener + an argv check) rather than trusting that a documented flag behaves identically across curl builds.
