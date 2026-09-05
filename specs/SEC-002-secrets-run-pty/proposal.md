---
id: "SEC-002-secrets-run-pty"
type: spec
status: draft # draft | implementing | verifying | archived
created: "2026-09-05"
issue: "mlorentedev/dotfiles#1506"
tags: [spec, proposal]
template_version: "1.0"
---

# SEC-002-secrets-run-pty

## Why

`dotf secrets run` is the **only sanctioned way to hand a secret to a process** — it is what the
doctrine points every agent and every shell wrapper at, instead of exporting a credential into the
ambient environment. Since `a720b9d` (#1459) it silently breaks every interactive child.

`pi`, wrapped in `.zshrc:92` and `.bashrc:114`, returns to the prompt in ~2s with **exit code 0 and
zero bytes**. The wrapper is the supported path; the unsupported path (`command pi`, ambient
`$NAN_API_KEY`) is the one that works. **A security control that breaks the tool it protects gets
routed around**, and the workaround is exactly the thing ADR-028 exists to prevent.

### Root cause

`cli/internal/cmd/secrets.go:504-506` wraps the child's streams for redaction and hands them to
`exec.Cmd` (`:618`):

```go
stdout := newRedactWriter(cmd.OutOrStdout(), injected)   // io.Writer, NOT *os.File
c.Stdin, c.Stdout, c.Stderr = stdin, stdout, stderr
```

`exec.Cmd` passes a descriptor through **only** when the dynamic type is `*os.File`; otherwise it
allocates an `os.Pipe`. Measured at the fd level:

```
DIRECT:    in=0  out=0  err=0
VIA DOTF:  in=0  out=1  err=1        fd1 -> pipe:[...], fd2 -> pipe:[...]
```

`pi` sees a non-TTY stdout, declines to start its TUI, exits 0. `runChild`'s own comment names the
assumption that broke: *"a wrapper that owns no terminal."*

**Why no reviewer caught it:** `pi -p '...'` is headless and needs no TTY, and that is the only form
the reviewer pool runs.

### The second defect, which a pty alone does not fix

`redactWriter.Write` (`secrets.go:750-756`) withholds a **fixed** `maxSecretLen - 1` bytes on *every*
write, unconditionally:

```go
if len(data) >= r.maxSecretLen && r.maxSecretLen > 1 {
    split := len(data) - (r.maxSecretLen - 1)
    r.tail = data[split:]          // invisible until the NEXT write
```

With a ~64-character API key that is **63 bytes of every frame withheld** until more output arrives.
That is correct for a batch pipe terminated by `Flush()`, and wrong for an interactive stream where
the tail of a frame is typically the cursor positioning. Fixing only the pty yields a TUI that
renders late and corrupt instead of not at all.

## What

Give the child a **pty when the parent has a terminal**, and keep today's pipe otherwise. Narrow the
redaction hold-back from a fixed window to *only bytes that could still become a secret*.

### Decisions — settled here, not left open

1. **The branch condition is `isTerminal(os.Stdout.Fd())`, nothing else.** Not a flag, not the
   child's name. This is what keeps `pi -p` and the whole reviewer pool on the existing, proven pipe
   path: they have no terminal, so nothing about them changes. `isTerminal` is a **package-level
   variable**, so CI — which has no TTY — can exercise both branches.
2. **A pty merges stdout and stderr onto one stream. Accepted.** A terminal does this anyway; that
   is what the user sees when running the child directly, so the pty path is *closer* to the
   unwrapped behaviour, not further. It also means **one** redactWriter on that path rather than
   two, which removes a race that exists today: two independent writers, each holding their own
   tail, can interleave a secret across the boundary between them. Recorded as a deliberate
   behaviour change: on the pty path, stderr is no longer separable by the caller.
3. **The redaction hold-back becomes prefix-aware.** Withhold the longest suffix of the buffer that
   is a **proper prefix of some secret**, and nothing else. In the common case that suffix is empty
   and the frame is written whole. This is the correct rule for the pipe path too, so it is not a
   pty-only concession — it removes a fixed-latency window that was always wrong and merely
   invisible in batch use.
4. **Redaction stays on, always.** The cheap alternative — pass `os.Stdout` through unredacted when
   it is a terminal — is refused. That would drop the #1459 guarantee precisely where the session
   transcript is captured, which is the case the guarantee exists for.
5. **The placeholder is a different length than the secret, which corrupts a TUI's cursor
   arithmetic.** Accepted and stated: a TUI that renders a live credential to the screen is the case
   redaction exists for, and a garbled frame is the correct outcome over a leaked key.
6. **Raw mode via `golang.org/x/term`** (already a dependency), restored by `defer` **and** from the
   signal handler. In raw mode the parent no longer receives Ctrl-C — the child gets `^C` through the
   pty's line discipline, which is the behaviour of running it directly. `SIGTERM` forwarding is
   kept; `SIGWINCH` is added and resizes the pty.
7. **Cross-OS by build tag.** `secrets_child_unix.go` (`//go:build !windows`) holds the pty branch;
   `secrets_child_windows.go` keeps today's `runChild` verbatim as the fallback. `GOOS=windows go
   vet ./...` must stay clean — a Windows build break is invisible to a Linux-only loop and fails the
   whole package (#1075).
8. **`assertSafeChildCommand` (#1459) runs on both paths.** The environment-introspection guard is
   not weakened by the new path; a test asserts it on the pty branch specifically.

### Open question the implementation must close

**`creack/pty` as a new dependency, or a pty implementation over `golang.org/x/sys` (already a
dep).** `creack/pty` is small, standard and maintained; `x/sys` avoids a new supply-chain edge for
~40 lines of `/dev/ptmx` ioctls. Decide in `tasks.md` before writing the code, and pin whichever is
chosen in `versions.conf` terms if it is new.

## Out of scope

- **`cmd/spec.go:92`** (`io.MultiWriter(os.Stdout, f)`) is the same mechanism and is **deliberately
  not fixed**: its child is the reviewer, which is headless by design, so the defect is latent
  rather than active. Fixing it would change how every adversarial review's transcript is captured —
  a separate blast radius, and this PR is on the secrets boundary.
- **Anything under `~/.zshrc` / `~/.bashrc`.** They are deployed from the repo and the wrappers are
  already correct; the defect is entirely below them.
- **`opencode`.** It renders 14383 bytes through the pipe today, so it tolerates a non-TTY. It is
  the *canary* for not regressing the pipe path, not a thing to change.
- **The `script(1)` stopgap.** Never shipped: `AGENTS.md:63` forbids the wrapper form, and without
  `/dev/null` `script` writes a `typescript` file holding everything the child printed — a secrets
  sink, which is the failure this whole command exists to avoid.

## Risks / open questions

- **The reviewer pool runs `pi -p` across parallel worktrees.** If the headless path regresses,
  adversarial reviews stop working repo-wide — including the review for this very spec. The pipe
  path is therefore touched only by the hold-back change, and that change is unit-tested independent
  of the pty.
- **The installed `dotf` is `vcs.modified=true`**, built from a dirty tree. All evidence must come
  from a binary rebuilt from this branch to a scratchpad path, and the report must say which binary
  produced it.
- **Evidence must not paste the captured stream.** ~40KB of TUI output in a PR or transcript is both
  noise and precisely the stream `secrets-never-in-output` is about. Report exit codes and byte
  counts.
- **`pty.Open()` in CI**: available on Linux runners, absent on Windows — which is why the Windows
  leg keeps the pipe path rather than skipping the tests.

## Acceptance criteria

- [ ] A committed test asserts the child gets a **pty** when the parent has one, and a **pipe** when
      it does not, by exercising the `isTerminal` seam in both directions.
- [ ] A committed test asserts an injected secret routed through the pty is redacted before reaching
      the parent's stdout, **including when split across two writes**.
- [ ] A committed test asserts the prefix-aware hold-back releases a frame whose tail is not a
      secret prefix **immediately**, i.e. without waiting for a subsequent write.
- [ ] `assertSafeChildCommand` is asserted to run on the pty path.
- [ ] Headless is unregressed: `pi -p` and the pipe path behave exactly as before.
- [ ] `GOOS=windows go vet ./...` clean; the Windows path is today's code, unchanged.
- [ ] Evidence from a binary rebuilt from this branch, reported as exit status and byte counts.

## References

- mlorentedev/dotfiles#1506 — the ticket, with the fd-level diagnosis
- #1459 / `a720b9d` — introduced the redacting writer, and the guard that must survive
- ADR-028 — the two-tier secrets model that makes `secrets run` the only sanctioned path
- `harness/enforced/secrets-never-in-output.md` — why redaction cannot be conditionally dropped
- #1075 — a Windows build break invisible to a Linux-only loop
- lesson 267 — a mutation that silently fails to apply reads as a passing test
