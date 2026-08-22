# ORCA.md

> **First, read `AGENTS.md` at the repo root** — canonical SSOT for all agents (Standing Orders, Decision Hierarchy, Neural Hive, MCP, Operational Rules). This file holds Orca ADE-specific tooling extensions and agent guidelines.

## Orca ADE Operating Notes

* **Worktree Environment:** Worktree root is at `$ORCA_WORKTREE_PATH`; repository root is at `$ORCA_ROOT_PATH`.
* **External Placement:** Never nest worktrees inside a repository. All worktrees live as sibling directories (e.g. `/home/manu/workspaces/orca-agent/<repo>-<slug>`).
* **Hooks:** Follow `orca.yaml` setup and archive hooks. Pinned toolchains are managed via `mise`.

<!-- BEGIN HARNESS GENERATED (sha256:6d01176da5d23b03) — SSOT: vault 00_meta/patterns; edit there + re-run setup, do NOT edit between markers -->
- **No AI attribution** in git history or GitHub messages (commits, PRs, issues).
- No `Co-Authored-By` trailers referencing AI agents.
- No bot-style emojis or "Generated with" footers.
- All artifacts must appear human-authored.
- **English only** in git/GitHub artifacts: commit messages, branch names, PR/issue titles and bodies, and code comments. Conversation with the user may be in any language; the durable record is English.
- **No internal phase/milestone references** in branch names, commit messages, or PR titles.
  - Bad: `feat/phase-3.1-scaffold`, `chore: scaffold repo (Phase 3.1)`
  - Good: `feat/scaffold-pyhydra3d`, `chore: scaffold PyHydra3D repository`
- Phase/milestone tracking belongs in the bitácora GitHub Project (issues + board), not in git history or the vault (per ADR-018).
- **Auto-merge is forbidden in every repository.** Never run `gh pr merge --auto`, never enable "Auto-merge" in the GitHub UI, and keep the repo setting `allow_auto_merge=false`. Auto-merge lands a PR the instant CI goes green — bypassing the human review gate in §1.
- Every PR merges deliberately, after a human has reviewed it and CI is green (squash or rebase per §4, diff verified per §5). Merge is a supervised action, never a queued automatic one. An agent merges only when the user has authorized merging that specific PR.
- **Zero manual operations:** Never perform ad-hoc out-of-band manual changes or temporary fixes on remote systems, servers, clusters, or cloud environments.
- **Strict IaC & Idempotence:** Every configuration, provision, deployment, or environment change MUST be codified in the repository as reproducible Infrastructure as Code (e.g. Ansible playbooks/roles, Terraform, Kubernetes manifests, dotfiles scripts) and verified to be 100% idempotent (`changed=0` on re-run).
- **In-flight documentation & zero debt:** Lessons, ADRs, and operational insights must be persisted in real-time as they occur (`docs/lessons/`, `docs/adr/`), never deferred to the end of the session. Defects noticed along the way must be fixed in scope or immediately filed as a GitHub issue with verified root cause.

> Injected verbatim into every agent's instructions (harness `enforced` id `definition-of-done`) and executed by the `verification-before-completion` skill. It **binds** existing standing orders to the moment of closing; it does not restate them.

Working code is not a finished change. Before saying done, each of these is true:

1. **Debt** — every defect noticed along the way is fixed in scope or filed as a ticket with its root cause. A mention in conversation is not an exit.
2. **Knowledge** — what was learned is written where it belongs, this session: build/operate detail in the repo (docs/lessons/, docs/adr/), cross-project insight in the store.
3. **Board** — the ticket matches reality: picked up when you start, blocked when blocked, closed with the change that closed it.
4. **Review** — an open PR is not finished work. Its checks and its reviewer comments are triaged, and each comment is applied, ticketed, or declined with a reason.
5. **Evidence** — no completion claim without the command output that proves it, produced in this session (e.g. test runs, and for PR work, `dotf pr triage-queue` returning exit 0 / queue clear).

Any of the five may be skipped, but only as a stated decision naming which one and why. Silence is not a skip.

> Injected verbatim into every agent's instructions (harness `enforced` id `pr-stewardship`). It elaborates Definition of Done §4 — "an open PR is not finished work" — into what that item leaves implicit: what you still owe a PR after you push it, and what does not count as having been reviewed.

**What binds is the disposition, not the waiting.** Before the change is called done, the PR's checks and its reviewer output are dispositioned — each one applied, ticketed, or declined with a reason. *How* you learn they arrived is not prescribed: a project that already tells you when to look back — the human notifies, a hook fires — has met this, and its instruction wins. Absent such a signal the default mechanism is to stay: the window closes at the first of an actionable reviewer comment or ten minutes after the checks settle, and pushing a fix reopens it, because the reviewer re-reviews. Leaving with nothing dispositioned hands the next session a change nobody read.

**"Hand the PR over; don't watch CI" is this rule's escape being exercised, not a contradiction of it.** Where a project names the signal, its instruction wins — and that rule names one: *the human reviews the PR and reports a red build*. So in a repository carrying it the timed window never opens, and what that rule forbids is the watch loop, never the disposition. Read in that order the two are one instruction: don't sit and watch, and don't leave the reviewer's output unread.

**A comment is not a review, and green checks are not the end of one.** Both halves have been observed failing here. On one PR every check went green and the reviewer then posted four Major findings. On another, checks went green and the reviewer posted *"review limit reached — we couldn't start this review"*: a comment arrived, and nobody looked. **A notice that no review ran leaves the PR unreviewed.** Tell the two apart by content, never by author — a review names files, lines, or claims; a notice talks about the review itself. Proceeding on an unreviewed PR is allowed; proceeding silently is not. "Merged unreviewed, reviewer quota exhausted" is a disclosure; saying nothing is a claim of review that never happened.

**Ask the queue rather than remember it, and write the answer where it survives.** Two commands make the disposition mechanical instead of conscientious, and they work from any shell and any agent because they are the same binary everywhere:

- **`dotf pr triage-queue`** — run it at session start, and again before reporting any PR work complete, in a repository that carries a reviewer registry. It lists pull requests whose newest reviewer output is newer than their newest recorded triage. **A non-zero exit is a queue you must read, never an empty one**: it exits non-zero both when work is pending *and* when the question could not be answered, precisely so an unanswerable queue is never mistaken for a clear one.
- **The `## Review triage` comment** — record the dispositions on the PR itself under that heading, *including when there was nothing to dispose of*. "CI green, no review findings" is a disposition; leaving it unwritten is indistinguishable from nobody having looked, and it leaves the PR queued forever. The heading is declared once, in the repository's reviewer registry, and read back by the queue.

Where a harness offers a session-time execution surface, wire the first one into it and stop relying on the instruction — a hook that fires beats an agent that remembers. Where it offers none, this paragraph *is* the mechanism, which is why it lives in the always-injected doctrine and not in a skill: an instruction every agent receives is the floor that survives a harness with no hooks at all.

**A change that closes a spec gets an independent adversarial review before it archives.** The trigger is the archive gate and nothing wider — not every PR that touches a spec folder. It names an obligation that already binds mechanically, so the only question is whether you meet it deliberately or discover it as a refusal: the spec gate declines to merge a PR closing a spec's issue without archiving it, `spec archive` declines without a passing review, and the reviewer pool declines one signed by the wrong model. The reviewer must not be the implementer; that independence is the entire value.

> Injected verbatim into every agent's instructions (harness `enforced` id `secrets-never-in-output`). Section 6 defends the commit; this defends the transcript, which no scanner reaches.

**The transcript is a durable artifact.** It is stored on disk, it may be synced, and later sessions read it. Everything the commit path forbids, it forbids too — and unlike a commit, nothing scans it and nothing can un-print it.

**Never dump a secrets store to standard output.** Decrypting a whole file and filtering the result is the shape that bites: the filter narrows what a human *reads*, never what was decrypted and emitted, and the transcript captures the stream before the filter runs. Measured 2026-08-20: one such command put a cloud access key pair, two control-plane keys and an admin password into a session transcript at once. Extract the single value you need, or keep the value out of stdout entirely by injecting it into the child process that consumes it — `dotf secrets run -- <cmd>` where it exists, the equivalent extract-or-exec form of your secrets tool everywhere else.

**Verify a credential by consequence, never by printing it.** To establish that a secret works, run the operation that uses it and report the exit status. Printing it to prove it exists is not verification; it is the failure. The same reasoning that makes a guard check whether a review was published rather than whether a known error appeared.

**This is not a tool defect and no tool will stop you.** Decrypting to stdout is exactly what a decryption command is for. There is no deterministic pre-exposure hook available either — agent stdout cannot be intercepted across every harness, and a scrubber that works in one is absent in the rest. This paragraph is the mechanism.

**If a value does reach the output: say so immediately, name the affected credentials by type, and stop.** Disclosure over silence, the same posture as an unreviewed merge. Then treat them as compromised and rotate — an exposed credential in a transcript nobody rotated is indistinguishable from one that was never exposed, right up until it is not.
<!-- END HARNESS GENERATED -->
