# Lesson 270 — A security wrapper that breaks the tool it protects gets routed around

**Date:** 2026-09-05
**Context:** SEC-002 (#1506) — `dotf secrets run` gave every child a pipe, so `pi` exited silently

## What happened

`dotf secrets run` is the *only sanctioned way* to hand a secret to a process: the whole point of
ADR-028 is that a credential is injected into one child rather than exported into the ambient shell.
Since #1459 it silently broke every interactive child. `pi`, wrapped in `.zshrc:92`, returned to the
prompt in ~2 seconds with **exit code 0 and zero bytes**.

So the *supported* path failed and the *unsupported* one — `command pi` with an exported key — worked.
That is the whole exposure ADR-028 exists to prevent, reintroduced by the control meant to prevent
it.

## The mechanism, which is a Go footgun worth knowing on its own

```go
stdout := newRedactWriter(cmd.OutOrStdout(), injected)   // an io.Writer
c.Stdin, c.Stdout, c.Stderr = stdin, stdout, stderr
```

`exec.Cmd` passes a **file descriptor** straight through to the child only when the writer's
**dynamic type is `*os.File`**. For any other `io.Writer` it allocates an `os.Pipe` and copies. So
wrapping stdout in anything — a redactor, an `io.MultiWriter`, a tee — silently changes what the
child *sees about its own environment*:

```
DIRECT:    in=0  out=0  err=0
VIA DOTF:  in=0  out=1  err=1        fd1 -> pipe:[...], fd2 -> pipe:[...]
```

Every TUI checks `isatty(1)` before drawing. The wrapper did not change the bytes; it changed the
answer to a question the child asks about itself.

**The subtle part: `os.Stdout` assigned into an `io.Writer` field still works**, because the dynamic
type survives the interface boundary. `cli/internal/tools/install.go` declares `Out io.Writer` and
defaults it to `os.Stdout` — and passes the fd through correctly. It *looks* identical to the broken
code and is not. You cannot audit this by reading the field's static type.

## Three failures, each a different member of a family this repo keeps meeting

### 1. The reviewer pool could not see it

`pi -p '...'` is headless and needs no TTY, and that is the **only** form the adversarial reviewers
run. A defect that only manifests interactively is invisible to an automated reviewer by
construction — not because the reviewers are weak, but because their execution mode is the one the
bug does not touch.

### 2. The fix that would not have fixed it

The obvious repair is "give the child a pty". That alone would have produced a TUI that renders
**late and misplaced** rather than not at all, because `redactWriter.Write` withheld a **fixed**
`maxSecretLen - 1` bytes on every write regardless of content:

```go
split := len(data) - (r.maxSecretLen - 1)
r.tail = data[split:]          // invisible until the NEXT write
```

With a 64-byte key that is 63 bytes of every frame held dark — and the tail of a terminal frame is
its cursor positioning. Correct against a pipe drained at `Flush()`; wrong against a terminal. **The
defect had a second layer whose symptom the first layer completely masked**, and it was found by
reading the code the fix would sit next to, not by reproducing the bug.

The right rule is prefix-aware: withhold only a trailing run that is a *proper prefix* of some
secret — zero in the common case.

### 3. The test that tested everything except the decision

The first implementation shipped with tests proving `runChildPTY` attaches a pty and `runChild` does
not. Both called their function **directly**, bypassing the one line that chooses between them. The
adversarial review caught it as a Major:

> the linchpin of the fix — the one-line `if interactiveChildSupported() && isTerminal(...)`
> selection — is unverified; an inverted condition would pass every test while silently reverting
> the fix.

This is lesson 268's family again — *"no drift" and "I did not check for drift" print identically* —
one layer up: **two green branch tests and a green branch test suite look the same whether or not
anything tests the branch.** Extracting the condition into a named function (`wantsInteractiveChild`)
and driving the seam in both directions is what closed it, and inverting the condition now fails.

## The lesson

**When you wrap a process for a security property, enumerate what the child can no longer observe
about itself.** Not what it can no longer *do* — what it can no longer *see*. A wrapper changes at
minimum: whether stdout/stderr are terminals, the process group and session, the controlling
terminal, the window size, and the signal path. Each of those is something a well-behaved program
inspects to decide how to behave.

And when the wrapper *is* the sanctioned path:

- **A broken supported path is a security regression, not a UX bug.** Its measure is not "the tool
  is annoying", it is "the user now exports the key into their shell". Rank it accordingly.
- **Test the tool the way people invoke it.** The wrapper had tests; none launched an interactive
  child, because the harness has no terminal. `pty.fork()` plus `TIOCSWINSZ` costs ~30 lines and is
  the difference between a suite that could catch this and one that cannot. A bare `pty.fork()` is a
  false negative: it leaves the window at 0x0 and the TUI refuses to draw for a different reason.
- **Never drop the security property to restore the behaviour.** The cheap fix here was to pass
  `os.Stdout` through unredacted when it is a terminal. That yields a perfect TUI and voids the
  redaction guarantee at exactly the point a session transcript is being captured — the case it
  exists for.

## Extent, for the next person auditing this

Of the 15 sites in `cli/internal` that assign a child's `Stdout`: **one active defect**
(`cmd/secrets.go`), **one latent instance of the identical shape** (`cmd/spec.go:92`, an
`io.MultiWriter` whose child is the reviewer — headless by design, so the defect cannot bite today),
and thirteen correct — either `*os.File` passthrough or `&bytes.Buffer` capture-by-design.

`cmd/spec.go:92` is deliberately not fixed: changing it changes how every adversarial review's
transcript is captured. It is recorded here so it is a known position rather than a surprise.

## See also

- Lesson 268 — a refused question and a negative answer print the same thing (this is its sibling
  one level up: an untested branch and a tested one produce identical green)
- Lesson 267 — a mutation harness must prove the mutation landed; both fixes here were proven in the
  failing direction with anchor-asserting patches
- ADR-028 — the two-tier secrets model that makes `secrets run` the only sanctioned path
- #1459 — introduced the redacting writer, and the introspection guard the fix had to preserve
