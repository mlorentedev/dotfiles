---
id: "lesson-564-land-the-capability-before-data-that-activates-it"
type: lesson
status: active
title: "Land the capability before data that activates it"
created: "2026-11-06"
---

# Land the capability before data that activates it

## Context

A stacked rollout first taught dotf to install checksummed archive releases, then
planned to publish a ts-bridge catalog entry with a new `roles` field. The first
binary understood the archive source but its JSON decoder silently ignored the
unknown role field.

## Problem

Source-type compatibility alone looked fail-closed: older binaries skipped an
unknown archive source. The intermediate binary was more dangerous because it
recognized the source while ignoring the selector, so a no-role machine could
install a role-bound tool during a mixed-version checkout/binary rollout.

## Solution

Land and test the selector capability before publishing data that depends on it.
Compatibility tests must exercise every relevant reader generation, not only the
oldest one: pre-capability, intermediate and current. Treat silently ignored JSON
fields as fail-open unless the behavior-activating reader proves otherwise.
