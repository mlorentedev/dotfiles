---
id: "lesson-344-concurrent-landers-each-update-every-pr-after-every-merge"
type: lesson
status: active
title: "Concurrent landers each update every PR after every merge"
created: "2026-10-07"
---

# Concurrent landers each update every PR after every merge

## Context
On 2026-10-07, four `dotf pr land --wait --update-branch` processes ran at once against five PRs, to
save wall-clock time. Branch protection on `main` is `strict`, so a PR must be up to date with its
base to merge. `--update-branch` handles that: when BEHIND is the only reason against a PR, it merges
the base in, waits for CI and decides again.

## The Trap
Each process looked after its own PR and was right to. Together they were not.

- Every merge into `main` marked every other open PR BEHIND.
- Every running process then pushed a merge commit of its own, to a PR that would be BEHIND again
  after the next merge.
- Each push started a full CI run and a pr-agent re-review. The re-review queue is repo-wide, with a
  median of 445 s, so the reviews of PRs nobody was about to land delayed those that were.
- Each re-review came back as reviewer output in the triage queue, which `land` refuses on.

With n PRs landing concurrently, merge k leaves n-k PRs BEHIND, and every one of them is updated
again: O(n²) CI runs and re-reviews. The bound of three updates per PR (`maxUpdates`) turns the cost
into a failure as well: the PR last in line is overtaken by every merge ahead of it, uses its three
updates, and exits unmerged.

Nothing reported it. Each process saw one PR, a base that moved, and a reasonable response to it.

## The Solution
Land the PRs as one queue: `dotf pr land <n> [<n>...]`, in the order given, one at a time. A PR is
updated only during its own turn, so it pays one update per base move while it is the one landing, and
the others stay quiet until then. A PR that stops is reported and the queue moves on; one summary
lists what merged and, for each PR left, why.

The queue only holds while it is the only lander, so `pr land` takes a per-repository lock under the
state dir. A second one refuses and names the first one's PID. The lock is a kernel lock
(`internal/filelock`), which the kernel frees when its holder exits: a lander that was killed cannot
leave the repository locked, and the PID file it leaves behind is taken over with a note. A lock file
created with O_EXCL and deleted on release does not have that property.

The general rule: an operation that is safe for one PR and that every other merge invalidates must not
run from several processes. Serialise it at the point where its cost is paid, and make the
serialisation a part of the tool, not a habit of whoever is running it.
