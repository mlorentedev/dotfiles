---
tags: [spec, tasks]
created: "2026-09-05"
---

# Tasks - SEC-002-secrets-run-pty

> TDD order. `[AC<n>]` maps to `proposal.md`'s acceptance criteria.

## Settled before implementation

**The dependency question is closed: `github.com/creack/pty` v1.1.24, not a hand-rolled
`x/sys` implementation.** Opening a pty correctly differs across Linux (`/dev/ptmx` +
`TIOCSPTLCK`/`TIOCGPTN`) and the BSDs/macOS (`posix_openpt`), and a per-platform ioctl path that
only ever runs on the author's box is the same class of defect as the one being fixed — untested
on the platforms it claims to support. `x/term` and `x/sys` were already dependencies; this adds
one small, standard, maintained module and confines it behind a build tag.

**The redactWriter fix is the blocking work, not the pty.** Discovered while reading, before any
code: `Write` withheld a fixed `maxSecretLen - 1` bytes on every call. A pty alone would have given
a TUI that renders late and misplaced instead of not at all.

## Implementation

- [x] [AC3] Failing test: a frame with no secret prefix in its tail is released on the write
- [x] [AC3] Failing test: a genuine trailing secret prefix is still withheld and later redacted
- [x] [AC3] Replace the fixed window with `holdBack` — longest trailing run that is a **proper
      prefix** of some secret, else zero
- [x] **Prove the mutation lands** (lesson 267): revert `holdBack` to the fixed rule with an
      anchor-asserting patch, watch both tests go red, restore
- [x] [AC1] `isTerminal` as a package-level seam so CI, which has no TTY, can reach both branches
- [x] [AC1] `secrets_child_unix.go` (`//go:build !windows`): `runChildPTY` — `pty.Start`, raw mode
      via `x/term` restored by defer **and** from the signal handler, SIGWINCH → `pty.InheritSize`,
      SIGTERM forwarded, exit code propagated
- [x] [AC1] `secrets_child_windows.go`: `interactiveChildSupported() == false`, pipe path kept
      verbatim, unreachable `runChildPTY` returns a loud error rather than panicking
- [x] [AC1] Call site chooses the path; **one** redactWriter on the pty path because a pty carries a
      single stream
- [x] [AC1] Tests: `test -t 1` answers yes on the pty path and no on the pipe path
- [x] [AC2] Test: secret split across two child writes is redacted through the pty
- [x] [AC4] Test: the #1459 introspection guard runs on the pty path
- [x] [AC6] `GOOS=windows go vet ./...` clean
- [x] [AC7] End-to-end evidence from a binary rebuilt from this branch

## Closing

- [x] `go build`, `go vet`, `GOOS=windows go vet`, `go test ./...` — all clean
- [x] `golangci-lint run` at the pinned 2.12.2 — 0 issues
- [x] `verification.md` filled in
- [ ] PR opened referencing this spec folder
- [ ] Independent adversarial review (reviewer != implementer) before archive
- [ ] Lesson written: an `io.Writer` that is not an `*os.File` silently becomes a pipe

## Deliberately not done

- **`cmd/spec.go:92`** — same `io.MultiWriter` shape, but its child is headless by design, so the
  defect is latent. Fixing it changes how every adversarial review's transcript is captured, which
  is a different blast radius from the secrets boundary.
- **A Windows ConPTY path** — see `secrets_child_windows.go` for the reasoning.
