---
id: "lesson-427-apt-get-update-exits-0-on-a-mirror-it-cannot-reach"
type: lesson
status: active
title: "apt-get update exits 0 on a mirror it cannot reach"
created: "2026-10-14"
owner: manu
tags: [ci, apt, docker, retry, integration]
---

# apt-get update exits 0 on a mirror it cannot reach

**Context:** #1498 asked for the integration image's `apt-get update` to retry the way the `test` job's "Install zsh" step already did. That step wraps `apt-get update` in a three-attempt loop that breaks on exit 0.

**Problem:** On 2026-10-14 the integration container failed with `Could not resolve 'archive.ubuntu.com'`. A retry loop would not have helped, and the `test` job's loop does not help either. Measured in an `ubuntu:26.04` container with a source whose host does not resolve:

- with apt's defaults, `apt-get update` printed the failure and exited **0**;
- with `APT::Update::Error-Mode "any"`, it exited 100, and the wrapper retried three times and then failed.

A failed index fetch is a warning by default. The update "succeeds" on stale or missing lists, so the loop breaks on the first attempt. The install that follows then fails on a missing package, which reads as a packaging problem rather than a network one. A mirror mid-sync (the #1498 case) is reported as an error, so only part of the failure class was ever retried.

**Solution:** Set `APT::Update::Error-Mode "any"` everywhere a loop retries `apt-get update`: in an `apt.conf.d` file in `tests/Dockerfile.integration`, and as `-o APT::Update::Error-Mode=any` in `ci.yml`'s `apt_opts`. In the Dockerfile, every `apt-get update` goes through one `apt-update-retry` script. `tests/ci-download-retry.bats` fails on a bare `apt-get update` there, and on a missing error mode in either place.

**Why:** A retry loop is only as good as the exit status it keys on. Before trusting one, provoke the failure it is meant to absorb and read the status. Here the only failure anyone had tested was the one apt already reported as an error.
