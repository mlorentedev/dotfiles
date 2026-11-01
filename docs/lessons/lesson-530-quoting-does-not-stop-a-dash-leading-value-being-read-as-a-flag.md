---
id: "lesson-530-quoting-does-not-stop-a-dash-leading-value-being-read-as-a-flag"
type: lesson
status: active
title: "Quoting does not stop a dash-leading value being read as a flag"
created: "2026-11-01"
---

# Quoting does not stop a dash-leading value being read as a flag

## Context

`dotf mem resume` prints a command for the operator to copy, built from a thread key that comes from MEMORY.md.
`mem.ShellWord` quotes the key when it holds shell syntax, so `x;echo` prints as `'x;echo'` and cannot run anything.

## Problem

A key such as `-x` matched ShellWord's bare charset and printed as `--thread -x`. Cobra parsed `-x` as a flag, so
the line failed when run as printed (#3002). Quoting it would not have helped: the shell removes the quotes before
cobra sees the argument, and `'-x'` arrives as `-x`. Quoting decides what the shell does with a word. Whether the
parser reads it as a flag is a separate question, and quoting does not answer it.

## Solution

Bind every value in a printed command to its flag with `=`: `--thread=-x`, `--memory='<path>'`, `--agent=pi`.
The parser takes everything after `=` as the value, whatever it starts with, and ShellWord still keeps shell syntax
out. A test for a printed command runs the line as printed, quotes removed the way a shell removes them, and checks
that it selects what it names. Comparing the output to a string cannot catch this.
