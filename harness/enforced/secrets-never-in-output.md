
> Injected verbatim into every agent's instructions (harness `enforced` id `secrets-never-in-output`). Section 6 defends the commit; this defends the transcript, which no scanner reaches.

**The transcript is a durable artifact.** It is stored on disk, it may be synced, and later sessions read it. Everything the commit path forbids, it forbids too — and unlike a commit, nothing scans it and nothing can un-print it.

**Never dump a secrets store to standard output.** Decrypting a whole file and filtering the result is the shape that bites: the filter narrows what a human *reads*, never what was decrypted and emitted, and the transcript captures the stream before the filter runs. Measured 2026-08-20: one such command put a cloud access key pair, two control-plane keys and an admin password into a session transcript at once. Extract the single value you need, or keep the value out of stdout entirely by injecting it into the child process that consumes it — `dotf secrets run -- <cmd>` where it exists, the equivalent extract-or-exec form of your secrets tool everywhere else.

**Verify a credential by consequence, never by printing it.** To establish that a secret works, run the operation that uses it and report the exit status. Printing it to prove it exists is not verification; it is the failure. The same reasoning that makes a guard check whether a review was published rather than whether a known error appeared.

**This is not a tool defect and no tool will stop you.** Decrypting to stdout is exactly what a decryption command is for. There is no deterministic pre-exposure hook available either — agent stdout cannot be intercepted across every harness, and a scrubber that works in one is absent in the rest. This paragraph is the mechanism.

**If a value does reach the output: say so immediately, name the affected credentials by type, and stop.** Disclosure over silence, the same posture as an unreviewed merge. Then treat them as compromised and rotate — an exposed credential in a transcript nobody rotated is indistinguishable from one that was never exposed, right up until it is not.
