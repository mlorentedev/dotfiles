---
id: "MEMORY-009-thread-writer-identity"
type: spec
status: implementing # draft | implementing | verifying | archived
created: "2026-09-25"
issue: "mlorentedev/dotfiles#1690"   # repo#NNN — GitHub issue / Project item that tracks this spec
tags: [spec, proposal]
template_version: "1.0"
---

# MEMORY-009-thread-writer-identity

## Why

<!-- from issue #1690: MEMORY-009: default-branch thread keys ignore who writes them, so two agents on one host overwrite each other's handoff -->

On a default branch the thread key is `main@<host>` or `master@<host>`, and it says nothing about who writes it. Two agents on one host that both hand off from a default-branch checkout resolve to one key, and `dotf mem handoff-write` replaces the other agent's block without a word. Four weeks of vault history hold three such overwrites (pi over claude, agy over claude, claude over antigravity), each found only when a later session followed a pointer into a block that was gone.

## What

- `handoff-write --agent <name>` stamps the writer into the thread's heading: `### thread: <key> (writer: <name>)`.
- Before replacing a block, the command reads who wrote it:
  - the heading's stamp, when there is one;
  - otherwise the agent named in the block's `Journal:` line, the field every thread has carried since HARNESS-088;
  - otherwise nobody.
- **Another agent's block is never replaced.** The block is kept, the write goes to the thread `<key>+<name>`, and stderr names both agents and the key.
- **Same writer, or no known writer:** the block is replaced in place, as today, and gains the stamp.
- **Without `--agent`:** the command behaves exactly as it does today. No stamp and no fork, so nothing that calls it changes until it passes the flag.
- The handoff skill passes `--agent`, so its sessions stamp their threads from now on.

## Out of scope

- Refusing instead of forking. The issue stages that decision after two weeks of fork notices, and this spec ships only the stamp and the fork.
- Listing forked candidates at session start. That is MEMORY-008's `dotf mem resume`, which already plans to list candidate threads.
- Detecting the agent from the environment. The caller names itself; a guessed identity is the failure this spec removes.

## Risks / open questions

- **`+` rather than `/` in the fork key.** The issue wrote `<key>/<agent>`. MEMORY-008 stores each thread as `memory/threads/<key>.md`, where a `/` would create a directory. `+` is filename-safe and appears in no key `dotf mem thread` derives.
- **`(writer: <agent>)` rather than the issue's `(<agent>)`.** A bare parenthesis is the shape the older branch suffix already takes (`### thread: wt-cli-023 (feat/cli-050-crystallize-cutover)`), so `(pi)` could not be told apart from a branch named `pi`. The label declares the stamp instead of inferring it from shape, the same rule that put the `thread:` marker in the heading.
- **Counting forks needs no log.** The issue stages enforcement after two weeks of counted conflicts. Every fork leaves a `### thread: <key>+<agent>` heading in the vault's git history, so `git log -p` over the MEMORY.md files is the count.
- **The skill passes `--agent` only once the binary accepts it.** A skill that names a flag the installed `dotf` lacks fails every handoff with "unknown flag", and the skill and the binary deploy separately. So AC5 is a second slice: the vault edit lands after a release carrying the flag is installed, and the spec archives with it.
- **Reading the writer from `Journal:` is a heuristic.** A journal named `<date>-<project>-<agent>[-<thread>].md` is parsed for the first word after the date that is the writer's own name or a known agent's; a line that names none attributes nothing. The known agents are the five the vault's journal names hold (claude, pi, antigravity, agy, copilot) and the harness's other targets. No vault project's name contains one, and the agent precedes the thread in the name, so the first match is the writer. A wrong answer that names another agent costs one extra thread; one that names the writer itself, or nobody, replaces the block as the command does today.
- **Unknown writers are replaced in place.** A block with neither a stamp nor a readable `Journal:` line is treated as the writer's own, which is today's behaviour. That is the one case this spec cannot protect, and the stamp removes it for every block written from now on.

## Acceptance criteria

- [ ] **AC1** — a block written by agent A, stamped or attributed by its `Journal:` line, is never replaced by a write from agent B under the same key; B's write lands in `<key>+B`.
- [ ] **AC2** — the fork is announced on stderr, naming the key and both agents.
- [ ] **AC3** — a rewrite by the same agent, or of a block with no known writer, replaces it in place and stamps the writer.
- [ ] **AC4** — without `--agent`, the output is byte-identical to the command's output before this change.
- [ ] **AC5** — the handoff skill's command passes `--agent`.

## References

- Bitácora: `mlorentedev/dotfiles#1690`; MEMORY-008 is #1689.
- Research: vault `10_projects/dotfiles/research/2026-09-24-handoff-resume-ritual.md`, sections 1, 2 and 3.5 (the measured overwrites and option (a')).
- Code: `cli/internal/mem/handoff.go` (`WriteThread`, `isThreadHeading`, `threadHeadingKey`), `cli/internal/cmd/mem_handoff.go`.
