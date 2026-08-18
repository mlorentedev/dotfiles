---
id: lesson-123-a-go-vs-shell-byte-equivalence-gate-is-posix-only-
type: lesson
status: active
created: "2026-06-24"
owner: manu
tags: [lesson, dotfiles]
---

# Lesson 123: A Go-vs-shell byte-equivalence gate is POSIX-only, and it retires at cutover

**Context**: CLI-025 ported the session-start hooks (`session-brief.sh`, `claude-session-start.sh`) to `dotf mem session-start`. Each port shipped a "golden" test that diffs the Go output against the live shell script across representative CWDs.

**Problem**: Two Windows-only divergences make a Go-vs-shell diff impossible to pass there even when the logic is byte-identical. (1) `jq` — the shell hook's JSON encoder — emits **CRLF** on a Windows build, while Go's `encoding/json` emits LF, so every line "differs". (2) The shell runs under Git Bash with MSYS `/tmp/...` paths, but the native Go binary on Windows cannot resolve `/tmp`, so any emitted absolute path (a vault headline, a "not found" line) renders differently (`/c/...` vs `C:\...`). The diff is therefore meaningful **only on Linux**, where both sides use LF and native `/tmp`.

**Solution**: Guard such gates with `if runtime.GOOS == "windows" { t.Skip(...) }` and let them run on the Linux CI job — the POSIX shell is the only equivalence target anyway. And **delete the gate at cutover**: once the shell script is `git rm`'d the gate has no referent to diff against. A byte-equivalence gate proves *fidelity during migration*; the Go unit tests are the *ongoing* regression net. A forever-skipped gate is dead weight — retire it with the shell it compared to.

---
