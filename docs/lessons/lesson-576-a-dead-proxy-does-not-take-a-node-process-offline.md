---
id: "lesson-576-a-dead-proxy-does-not-take-a-node-process-offline"
type: lesson
status: active
title: "A dead proxy does not take a Node process offline"
created: "2026-11-10"
---

# A dead proxy does not take a Node process offline

## Context

#1810 asked for a re-runnable CI case proving pi starts with pi-nan-provider while NaN is unreachable, by
pointing `HTTPS_PROXY` and `HTTP_PROXY` at a dead local port and listing models. The one-off measurement it
replaces (AI-046 AC5, 2026-09-26) had used a Node preload that threw on `fetch`, `net.connect` and `tls.connect`.

## Problem

Two things made the proxy variables prove nothing, and only a listener could tell them apart. First, Node's
`fetch` (undici) ignores `HTTP_PROXY`/`HTTPS_PROXY` unless the process opts in (`NODE_USE_ENV_PROXY=1` or a
`ProxyAgent`), and pi-nan-provider's catalog fetch is a bare `fetch(url)`: with a local TCP listener as the
proxy, pi made zero connections to it, flag or no flag. Second, and the reason the flag changed nothing either:
`pi --list-models` in pi 1.2.0 builds its model runtime without network refresh, so the package's `fetchModels`
is never called on that path. A preload recording every `fetch` and socket connect inside pi's process logged
zero attempts. The bats header had explained the deterministic listing as "NaN answers a dummy key with 401, so
the package keeps its snapshot": there was no request to answer.

## Solution

Cut the network inside the process, not around it: `NODE_OPTIONS=--import=tests/fixtures/pi-network-cut.mjs`
replaces `globalThis.fetch`, `net.Socket.prototype.connect` and `tls.connect` with throwing stubs and writes a
marker on load. The case asserts the marker (a dropped `NODE_OPTIONS` would otherwise pass for the wrong
reason), exit 0, and the listing equal to the package snapshot's ids. Mutations through the suite: an uncaught
network call at extension load turns only this case red; hiding the snapshot turns every case red; making the
catalog fetch fatal stays green, which is the proof that the fetch is off the start path, and the case says so.

Before trusting an environment knob (a proxy, `PI_OFFLINE`, a DNS override) to simulate a failure, put a
listener or a recorder where the traffic would go and count. Zero hits with the knob on and off means the knob
is not on the code path, and a test built on it measures nothing.
