---
id: "lesson-536-a-default-read-from-the-cwd-follows-the-shell-not-the-change"
type: lesson
status: active
title: "A default read from the cwd follows the shell, not the change"
created: "2026-11-02"
---

# A default read from the cwd follows the shell, not the change

## Context

`dotf forge pr create` takes `--head` from the current branch when the flag is absent. An unattended agent
commits in an item worktree, but its harness resets the shell to the launch worktree between calls.

## Problem

Run as its own call, `pr create` read the launch worktree's branch. In kubelab that was the run branch, which
carries `.agent/unattended.yml` and the ledger and is never merged. The command opened kubelab#2524 from it
(#2784). PR-Agent caught it before a merge; nothing in dotf did.

## Solution

`pr create` refuses a head whose tree holds `.agent/unattended.yml`, checked on the local branch and on
`origin/<head>`, and names `--head` as the fix. The refusal reads the branch's content, not its name, so it
also covers an attended session that is in the wrong directory.

The shape is a head whose envelope differs from the base's, not a head that holds one. A repository whose
merge guard reads the envelope from its default branch commits it there, and a check on presence alone
refused every pull request in it (#3125). A head that adds or changes the envelope still refuses, and so does
any envelope when the base is unknown.

## Why

A default derived from the working directory is a guess about which change the caller means. When the caller
is an agent, its directory is set by the harness, not by the work, so the guess can be wrong while every other
input is right. Where the wrong value has a recognisable shape, refuse on that shape. Matching on content is
safe here because it can only withhold a pull request, never grant one.
