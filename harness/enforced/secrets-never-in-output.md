
> Injected verbatim into every agent's instructions (harness `enforced` id `secrets-never-in-output`). Section 6 defends the commit; this defends the transcript, which no scanner reaches.

**The transcript is a durable artifact** (stored, synced, read by later sessions; nothing scans it, nothing can un-print it): everything the commit path forbids, it forbids too.

**Never dump a secrets store to stdout.** Decrypting a whole file and filtering the result still puts everything in the transcript, which captures the stream before the filter. Extract the single value you need, or inject it into the consuming child process (`dotf secrets run -- <cmd>` where it exists, the equivalent extract-or-exec form elsewhere).

**Verify a credential by consequence, never by printing it:** run the operation that uses it and report the exit status.

**No tool will stop you:** decrypting to stdout is what decryption commands do, and agent stdout cannot be intercepted; this rule is the mechanism.

**If a value does reach the output: say so immediately, name the affected credentials by type, and stop.** Then treat them as compromised and rotate: an exposed credential nobody rotated is indistinguishable from one never exposed.
